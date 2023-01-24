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

package echo_adapter

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/blib/servefiles/v3"
	"github.com/labstack/echo/v4"
	"github.com/spf13/afero"
)

// EchoAssets is merely an adapter for servefiles.Assets with the same API and with an
// additional HandlerFunc method.
type EchoAssets servefiles.Assets

// NewAssetHandler creates an Assets value. The parameter is the directory containing the asset files;
// this can be absolute or relative to the directory in which the server process is started.
//
// This function cleans (i.e. normalises) the asset path.
func NewAssetHandler(assetPath string) *EchoAssets {
	return (*EchoAssets)(servefiles.NewAssetHandler(assetPath))
}

// NewAssetHandlerFS creates an Assets value for a given filesystem.
func NewAssetHandlerFS(fs afero.Fs) *EchoAssets {
	return (*EchoAssets)(servefiles.NewAssetHandlerFS(fs))
}

// NewAssetHandlerIoFS creates an Assets value for a given filesystem.
func NewAssetHandlerIoFS(fs fs.FS) *EchoAssets {
	return (*EchoAssets)(servefiles.NewAssetHandlerIoFS(fs))
}

// StripOff alters the handler to strip off a specified number of segments from the path before
// looking for the matching asset. For example, if StripOff(2) has been applied, the requested
// path "/a/b/c/d/doc.js" would be shortened to "c/d/doc.js".
//
// The returned handler is a new copy of the original one.
func (a EchoAssets) StripOff(unwantedPrefixSegments int) *EchoAssets {
	return (*EchoAssets)((servefiles.Assets)(a).StripOff(unwantedPrefixSegments))
}

// WithMaxAge alters the handler to set the specified max age on the served assets.
//
// The returned handler is a new copy of the original one.
func (a EchoAssets) WithMaxAge(maxAge time.Duration) *EchoAssets {
	return (*EchoAssets)((servefiles.Assets)(a).WithMaxAge(maxAge))
}

// WithNotFound alters the handler so that 404-not found cases are passed to a specified
// handler. Without this, the default handler is the one provided in the net/http package.
//
// The returned handler is a new copy of the original one.
func (a EchoAssets) WithNotFound(notFound http.Handler) *EchoAssets {
	a.NotFound = notFound
	return &a
}

// HandlerFunc gets the asset handler as an Echo handler. The handler is
// registered using a catch-all path such as "/files/*". The same
// match-any pattern can be passed in, in which case it is stripped off
// the leading part of the URL path seem by the asset handler.
func (a *EchoAssets) HandlerFunc(path string) echo.HandlerFunc {
	trim := 0
	if strings.HasSuffix(path, "/*") {
		trim = len(path) - 2
	} else {
		panic(path + ": path must end /* or be blank")
	}

	return func(c echo.Context) error {
		req := c.Request()
		req.URL.Path = req.URL.Path[trim:]
		(*servefiles.Assets)(a).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// Register registers the asset handler with an Echo engine using the specified
// path to handle GET and HEAD requests.
//
// The handler is registered using a catch-all path such as "/files/*". This
// pattern will be stripped off the leading part of the URL path seem by the
// asset handler when determining the file to be served.
func (a *EchoAssets) Register(e *echo.Echo, path string) {
	if !strings.HasSuffix(path, "/*") {
		panic(path + ": path must end /*")
	}
	h := a.HandlerFunc(path)
	e.GET(path, h)
	e.HEAD(path, h)
}
