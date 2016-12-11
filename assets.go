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
	"log"
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
)

// This needs to track the same string in net/http (which is unlikely ever to change)
const indexPage = "index.html"

// Assets sets the options for asset handling. Use AssetHandler to create the handler(s) you need.
type Assets struct {
	// Choose a number greater than zero to strip off some leading segments from the URL path. This helps if
	// you want, say, a sequence number in the URL so that only has the effect of managing far-future cache
	// control. Use zero for default behaviour.
	UnwantedPrefixSegments int

	// The directory where the assets reside.
	AssetPath string

	// Set the expiry duration for assets. This will be set via headers in the response. This should never be
	// negative. Use zero to disable asset caching in clients and proxies.
	MaxAge time.Duration

	// Configurable http.Handler which is called when no matching route is found. If it is not set,
	// http.NotFound is used.
	NotFound http.Handler

	expiryElasticity time.Duration
	timestamp        int64
	timestampExpiry  string
	lock             sync.Mutex
}

// Type conformance proof
var _ http.Handler = &Assets{}

// AssetHandler creates an Assets value. It provides some bounds checking, so use it instead of
// creating Assets values directly.
//
// This function is deprecated; use NewAssetHandler instead.
func AssetHandler(unwantedPrefixSegments int, assetPath string, maxAge time.Duration) *Assets {
	if unwantedPrefixSegments < 0 {
		panic(errors.New("Negative unwantedPrefixSegments"))
	}
	if maxAge < 0 {
		panic(errors.New("Negative maxAge"))
	}
	cleanPath := cleanPathAndAppendSlash(assetPath)
	return &Assets{unwantedPrefixSegments, cleanPath, maxAge, nil, 0, 0, "", sync.Mutex{}}
}

// NewAssetHandler creates an Assets value. It cleans the asset path, so use it instead of
// creating Assets values directly.
func NewAssetHandler(assetPath string) *Assets {
	a := &Assets{}
	a.AssetPath = cleanPathAndAppendSlash(assetPath)
	a.lock = sync.Mutex{}
	return a
}

func (a Assets) StripOff(unwantedPrefixSegments int) *Assets {
	if unwantedPrefixSegments < 0 {
		panic(errors.New("Negative unwantedPrefixSegments"))
	}
	a.UnwantedPrefixSegments = unwantedPrefixSegments
	return &a
}

func (a Assets) WithMaxAge(maxAge time.Duration) *Assets {
	if maxAge < 0 {
		panic(errors.New("Negative maxAge"))
	}
	a.MaxAge = maxAge
	return &a
}

func (a Assets) WithNotFound(notFound http.Handler) *Assets {
	a.NotFound = notFound
	return &a
}

// Calculate the 'Expires' value using an approximation that reduces unimportant re-calculation.
// We don't need to do this accurately because the 'Cache-Control' maxAge value takes precedence
// anyway. So the value is cached and shared between requests for a short while.
func (a *Assets) expires() string {
	if a.expiryElasticity == 0 {
		a.expiryElasticity = 1 + a.MaxAge/100
	}
	now := time.Now()
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
	//log.Printf("removeUnwantedSegments %s", path)
	for i := a.UnwantedPrefixSegments; i >= 0; i-- {
		slash := strings.IndexByte(path, '/') + 1
		if slash > 0 {
			path = path[slash:]
		}
	}
	return path
}

func listContainsWholeString(header, want string) bool {
	accepted := strings.Split(header, ",")
	for _, encoding := range accepted {
		if strings.TrimSpace(encoding) == want {
			return true
		}
	}
	return false
}

func checkPlainResource(resource string, header http.Header) string {
	d, err := os.Stat(resource)
	if err == nil {
		// strong etag because the representation is the original file
		header.Set("ETag", fmt.Sprintf(`"%x-%x"`, d.ModTime().Unix(), d.Size()))
	}
	return resource
}

func (a *Assets) chooseResource(header http.Header, req *http.Request) (string, int, string) {
	name := a.removeUnwantedSegments(req.URL.Path)
	if name == "" || strings.HasSuffix(name, "/") {
		name += indexPage
	}

	resource := a.AssetPath + name
	gzipped := resource + ".gz"

	if a.MaxAge > 0 {
		header.Set("Expires", a.expires())
		header.Set("Cache-Control", fmt.Sprintf("public, maxAge=%d", a.MaxAge/time.Second))
	}

	d, err := os.Stat(gzipped)
	if err != nil {
		if os.IsNotExist(err) {
			// gzipped does not exist; original might but this gets checked later
			return checkPlainResource(resource, header), 0, ""

		} else if os.IsPermission(err) {
			// incorrectly assembled gzipped asset is treated as an error
			return resource, http.StatusForbidden, "403 Forbidden"
		}

		// Possibly the server is under heavy load and ran out of file descriptors
		backoff := 2 + rand.Int31()%4 // 2–6 seconds to prevent a stampede
		header.Set("Retry-After", strconv.Itoa(int(backoff)))
		log.Printf("Failed to stat %s: %v\n", resource, err)
		return resource, http.StatusServiceUnavailable, "Currently unavailable"
	}

	if d.IsDir() {
		// this odd case is simply passed on to the standard library
		return resource, 0, ""
	}

	// gzipped file exists and is readable
	acceptEncoding, ok := req.Header["Accept-Encoding"]
	if ok {
		acceptGzip := listContainsWholeString(acceptEncoding[0], "gzip")
		if acceptGzip {
			ext := filepath.Ext(resource)
			header.Set("Content-Type", mime.TypeByExtension(ext))
			// the standard library sometimes overrides the content type via sniffing
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("Content-Encoding", "gzip")
			header.Add("Vary", "Accept-Encoding")
			// weak etag because the representation is not the original file but a compressed variant
			header.Set("ETag", fmt.Sprintf(`W/"%x-%x"`, d.ModTime().Unix(), d.Size()))
			return gzipped, 0, ""
		}
	}

	// no intervention; the file will be served normally by the standard api
	return checkPlainResource(resource, header), 0, ""
}

// ServeHTTP implements the http.Handler interface.
func (a *Assets) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	resource, code, message := a.chooseResource(w.Header(), req)
	//fmt.Printf(".......... ServeHTTP %s %s -> %d %s\n", req.Method, req.URL.Path, code, resource)
	if code >= 400 {
		http.Error(w, message, code)
		return
	}

	if a.NotFound == nil {
		// Conditional requests and content negotiation are handled in ServeFile.
		// Note that req.URL remains unchanged, even if prefix stripping is turned on, because the resource is
		// the only value that matters.
		http.ServeFile(w, req, resource)

	} else {
		ww := newNo404Writer(w)

		//fmt.Printf(".......... ServeFile %s %s %+v\n", req.Method, req.URL.Path, w.Header())
		http.ServeFile(ww, req, resource)

		if ww.Code == http.StatusNotFound {
			// ww has silently dropped the headers and body from the built-in handler in this case,
			// so complete the response using the original handler.
			w.Header().Set("X-Content-Type-Options", "nosniff")
			//fmt.Printf(">>>>>>>>> %s %s %+v\n", req.Method, req.URL.Path, w.Header())
			a.NotFound.ServeHTTP(w, req)
		}
	}
}

func cleanPathAndAppendSlash(s string) string {
	clean := path.Clean(s)
	return string(append([]byte(clean), '/'))
}
