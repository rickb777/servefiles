// MIT License
//
// Copyright (c) 2016 Rick Beton
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package servefiles

import (
	"errors"
	"fmt"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/spf13/afero"
)

// This needs to track the same string in net/http (which is unlikely ever to change)
const indexPage = "index.html"

// Assets sets the options for asset handling. Use AssetHandler to create the handler(s) you need.
type Assets struct {
	// Choose a number greater than zero to strip off some leading segments from the URL path. This helps if
	// you want, say, a sequence number in the URL so that only has the effect of managing far-future cache
	// control. Use zero for default behaviour.
	UnwantedPrefixSegments int

	// Set the expiry duration for assets. This will be set via headers in the response. This should never be
	// negative. Use zero to disable asset caching in clients and proxies.
	MaxAge time.Duration

	// Configurable http.Handler which is called when no matching route is found. If it is not set,
	// http.NotFound is used.
	NotFound http.Handler

	fs               afero.Fs
	server           http.Handler
	expiryElasticity time.Duration
	timestamp        int64
	timestampExpiry  string
	lock             *sync.Mutex
}

// Type conformance proof
var _ http.Handler = &Assets{}

//-------------------------------------------------------------------------------------------------

// NewAssetHandler creates an Assets value. The parameter is the directory containing the asset files;
// this can be absolute or relative to the directory in which the server process is started.
//
// This function cleans (i.e. normalises) the asset path.
func NewAssetHandler(assetPath string) *Assets {
	cleanAssetPath := cleanPathAndAppendSlash(assetPath)
	fs := afero.NewBasePathFs(afero.NewOsFs(), cleanAssetPath)
	return NewAssetHandlerFS(fs)
}

// NewAssetHandlerFS creates an Assets value for a given filesystem.
func NewAssetHandlerFS(fs afero.Fs) *Assets {
	return &Assets{
		fs:     fs,
		server: http.FileServer(afero.NewHttpFs(fs)),
		lock:   &sync.Mutex{},
	}
}

// StripOff alters the handler to strip off a specified number of segments from the path before
// looking for the matching asset. For example, if StripOff(2) has been applied, the requested
// path "/a/b/c/d/doc.js" would be shortened to "c/d/doc.js".
//
// The returned handler is a new copy of the original one.
func (a Assets) StripOff(unwantedPrefixSegments int) *Assets {
	if unwantedPrefixSegments < 0 {
		panic(errors.New("Negative unwantedPrefixSegments"))
	}
	a.UnwantedPrefixSegments = unwantedPrefixSegments
	return &a
}

// WithMaxAge alters the handler to set the specified max age on the served assets.
//
// The returned handler is a new copy of the original one.
func (a Assets) WithMaxAge(maxAge time.Duration) *Assets {
	if maxAge < 0 {
		panic(errors.New("Negative maxAge"))
	}
	a.MaxAge = maxAge
	return &a
}

// WithNotFound alters the handler so that 404-not found cases are passed to a specified
// handler. Without this, the default handler is the one provided in the net/http package.
//
// The returned handler is a new copy of the original one.
func (a Assets) WithNotFound(notFound http.Handler) *Assets {
	a.NotFound = notFound
	return &a
}

//-------------------------------------------------------------------------------------------------

// Calculate the 'Expires' value using an approximation that reduces unimportant re-calculation.
// We don't need to do this accurately because the 'Cache-Control' maxAge value takes precedence
// anyway. So the value is cached and shared between requests for a short while.
func (a *Assets) expires() string {
	if a.expiryElasticity == 0 {
		// lazy initialisation
		a.expiryElasticity = 1 + a.MaxAge/100
	}

	now := time.Now().UTC()
	unix := now.Unix()

	if unix > a.timestamp {
		later := now.Add(a.MaxAge + a.expiryElasticity) // add expiryElasticity to avoid negative expiry
		a.lock.Lock()
		defer a.lock.Unlock()
		// cache the formatted string for one second to avoid repeated formatting
		// race condition is ignored here, but note the order below
		a.timestampExpiry = later.Format(time.RFC1123)
		a.timestamp = unix + int64(a.expiryElasticity)
	}

	return a.timestampExpiry
}

func (a *Assets) removeUnwantedSegments(path string) string {
	for i := a.UnwantedPrefixSegments; i > 0; i-- {
		slash := strings.IndexByte(path[1:], '/') + 1
		if slash <= 0 {
			debugf("removeUnwantedSegments => /\n")
			return "/"
		}
		debugf("removeUnwantedSegments %d %d %s\n", i, slash, path)
		path = path[slash:]
	}
	debugf("removeUnwantedSegments = %s\n", path)
	return path
}

//-------------------------------------------------------------------------------------------------

type fileData struct {
	resource string
	code     code
	fi       os.FileInfo
}

func calculateEtag(fi os.FileInfo) string {
	if fi == nil {
		return ""
	}
	return fmt.Sprintf(`"%x-%x"`, fi.ModTime().Unix(), fi.Size())
}

func handleSaturatedServer(header http.Header, resource string, err error) fileData {
	// Possibly the server is under heavy load and ran out of file descriptors
	backoff := 2 + rand.Int31()%4 // 2–6 seconds to prevent a stampede
	header.Set("Retry-After", strconv.Itoa(int(backoff)))
	//log.Printf("Failed to stat %s: %v\n", resource, err)
	debugf("handleSaturatedServer 503 %s\n", resource)
	return fileData{resource, ServiceUnavailable, nil}
}

func (a *Assets) checkResource(resource string, header http.Header) fileData {
	d, err := a.fs.Stat(resource)
	if err != nil {
		if os.IsNotExist(err) {
			// gzipped does not exist; original might but this gets checked later
			debugf("checkResource 404 %s\n", resource)
			return fileData{"", NotFound, nil}

		} else if os.IsPermission(err) {
			// incorrectly assembled gzipped asset is treated as an error
			debugf("checkResource 403 %s\n", resource)
			return fileData{resource, Forbidden, nil}
		}

		return handleSaturatedServer(header, resource, err)
	}

	if d.IsDir() {
		// directory edge case is simply passed on to the standard library
		return fileData{resource, Directory, nil}
	}

	debugf("checkResource 100 %s\n", resource)
	return fileData{resource, Continue, d}
}

func (a *Assets) chooseResource(header http.Header, req *http.Request) (string, code) {
	resource := a.removeUnwantedSegments(req.URL.Path)
	if strings.HasSuffix(resource, "/") {
		resource += indexPage
	}
	debugf("chooseResource %s %s %s\n", req.Method, req.URL.Path, resource)

	if a.MaxAge > 0 {
		header.Set("Expires", a.expires())
		header.Set("Cache-Control", fmt.Sprintf("public, maxAge=%d", a.MaxAge/time.Second))
	}

	acceptEncoding := commaSeparatedList(req.Header.Get("Accept-Encoding"))
	if acceptEncoding.Contains("gzip") {
		gzipped := resource + ".gz"

		fdgz := a.checkResource(gzipped, header)

		if fdgz.code == Continue {
			ext := filepath.Ext(resource)
			header.Set("Content-Type", mime.TypeByExtension(ext))
			// the standard library sometimes overrides the content type via sniffing
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("Content-Encoding", "gzip")
			header.Add("Vary", "Accept-Encoding")
			// weak etag because the representation is not the original file but a compressed variant
			header.Set("ETag", "W/"+calculateEtag(fdgz.fi))
			return gzipped, Continue
		}
	}

	// no intervention; the file will be served normally by the standard api
	fd := a.checkResource(resource, header)

	if fd.code > 0 {
		// strong etag because the representation is the original file
		header.Set("ETag", calculateEtag(fd.fi))
	}

	return fd.resource, fd.code
}

// ServeHTTP implements the http.Handler interface.
func (a *Assets) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	resource, code := a.chooseResource(w.Header(), req)
	debugf("ServeHTTP %s %s %+v -> %d %s\n", req.Method, req.URL.Path, req.Header, code, resource)

	if code == NotFound && a.NotFound != nil {
		debugf("ServeFile (2) %s %s W:%+v\n", req.Method, req.URL.Path, w.Header())

		// ww has silently dropped the headers and body from the built-in handler in this case,
		// so complete the response using the original handler.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		a.NotFound.ServeHTTP(w, req)
		return
	}

	if code >= 400 {
		http.Error(w, code.String(), int(code))
		return
	}

	req.URL.Path = resource

	// Conditional requests and content negotiation are handled in ServeFile.
	// Note that req.URL remains unchanged, even if prefix stripping is turned on, because the resource is
	// the only value that matters.
	debugf("ServeFile (1) %s %s W:%+v\n", req.Method, req.URL.Path, w.Header())
	a.server.ServeHTTP(w, req)
}

func cleanPathAndAppendSlash(s string) string {
	clean := path.Clean(s)
	return string(append([]byte(clean), '/'))
}

func debugf(msg string, args ...interface{}) {
	//fmt.Printf(msg, args...)
}
