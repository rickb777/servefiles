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
	"io/fs"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rickb777/path"
	"github.com/spf13/afero"
)

// This needs to track the same string in net/http (which is unlikely ever to change)
const IndexPage = "index.html"

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

	// Configurable http.Handler which is called when the request method is neither HEAD nor GET. If it is not
	// set a basic handler like http.NotFound is used.
	MethodNotAllowed http.Handler

	// DisableDirListing prevents directory listings being generated with the URL path ends with '/'.
	// If an index.html file is present, it is served for its directory path regardless of this setting.
	// Otherwise, a directory listing page will be generated if this flag is false, or when it is true
	// a 404-not found is given.
	DisableDirListing bool

	// the local filesystem (remember that all paths are relative to its root)
	fs               fs.FS
	server           http.Handler
	expiryElasticity time.Duration
	timestamp        int64
	timestampExpiry  string
	maxAgeS          int // max age in seconds (pre-calculated)
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
	cleanAssetPath := path.Clean(assetPath)
	Debugf("NewAssetHandler %s\n", cleanAssetPath)
	filesystem := os.DirFS(cleanAssetPath).(fs.StatFS)
	return NewAssetHandlerIoFS(filesystem)
}

// NewAssetHandlerFS creates an Assets value for a given filesystem.
func NewAssetHandlerFS(fs afero.Fs) *Assets {
	return &Assets{
		fs:     afero.NewIOFS(fs),
		server: http.FileServer(afero.NewHttpFs(fs)),
		lock:   &sync.Mutex{},
	}
}

// NewAssetHandlerIoFS creates an Assets value for a given filesystem.
// Implementations include os.DirFS.
func NewAssetHandlerIoFS(fs fs.FS) *Assets {
	return &Assets{
		fs:     fs,
		server: http.FileServer(http.FS(fs)),
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
		panic("Negative unwantedPrefixSegments")
	}
	a.UnwantedPrefixSegments = unwantedPrefixSegments
	return &a
}

// WithMaxAge alters the handler to set the specified max age on the served assets.
//
// The returned handler is a new copy of the original one.
func (a Assets) WithMaxAge(maxAge time.Duration) *Assets {
	if maxAge < 0 {
		panic("Negative maxAge")
	}
	a.MaxAge = maxAge
	a.maxAgeS = int(maxAge / time.Second)
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

// WithMethodNotAllowed alters the handler so that 405-method not allowed cases are passed
// to a specified handler. Without this, the default handler is like the one provided in the
// net/http package (see http.NotFound).
//
// The returned handler is a new copy of the original one.
func (a Assets) WithMethodNotAllowed(notAllowed http.Handler) *Assets {
	a.MethodNotAllowed = notAllowed
	return &a
}

//-------------------------------------------------------------------------------------------------

// Printer is something that allows formatted printing. This is only used for diagnostics.
type Printer func(format string, v ...interface{})

// Debugf is a function that allows diagnostics to be emitted. By default it does very
// little and has almost no impact. Set it to some other function (e.g. using log.Printf) to
// see the diagnostics.
var Debugf Printer = func(format string, v ...interface{}) {}

// example (paste this into setup code elsewhere)
//var servefiles.Debugf Printer = log.Printf
