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

package echo_adapter_test

import (
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rickb777/expect"
	"github.com/rickb777/servefiles/v3/afero2"
	"github.com/rickb777/servefiles/v3/echo_adapter"
	"github.com/rickb777/servefiles/v3/testdata"
	"github.com/spf13/afero"
)

const (
	cssMimeType        = "text/css; charset=utf-8"
	javascriptMimeType = "text/javascript; charset=utf-8"
)

func ExampleHandlerFunc() {
	// This is a webserver using the asset handler provided by
	// github.com/rickb777/servefiles/v3, which has enhanced
	// HTTP expiry, cache control, compression etc.
	// 'Normal' bespoke handlers are included as needed.

	// where the assets are stored (replace as required)
	localPath := "./assets"

	// how long we allow user agents to cache assets
	// (this is in addition to conditional requests, see
	// RFC9111 https://www.rfc-editor.org/rfc/rfc9111#section-5.2.2.1)
	maxAge := time.Hour

	// define the URL pattern that will be routed to the asset handler
	// (optional)
	const path = "/files/*"

	router := echo.New()
	// ... add other routes / handlers / middleware as required

	h := echo_adapter.NewAssetHandler(localPath).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1).
		HandlerFunc(path)

	router.GET(path, h)
	router.HEAD(path, h)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func TestHandlerFunc_with_AferoFS(t *testing.T) {
	maxAge := time.Hour
	files := afero2.AferoAdapter{Inner: afero.NewMemMapFs()}
	files.MkdirAll("/foo/bar", 0755)
	afero.WriteFile(files, "/foo/bar/x.txt", []byte("hello"), 0644)

	const assetPath = "/files/*"

	h := echo_adapter.NewAssetHandlerFS(files).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1)

	router := echo.New()
	// ... add other routes / handlers / middleware as required
	h.Register(router, assetPath)

	r, _ := http.NewRequest(http.MethodGet, "http://localhost/files/101/foo/bar/x.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	expect.Number(w.Code).ToBe(t, 200)
	expect.String(w.Header().Get("Content-Type")).ToBe(t, "text/plain; charset=utf-8")
	expect.String(w.Header().Get("Expires")).Not().ToBe(t, "")
	expect.Number(w.Body.Len()).ToBe(t, 5)

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/foo/bar/x.txt", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	expect.Number(w.Code).ToBe(t, 200)
	expect.String(w.Header().Get("Content-Type")).ToBe(t, "text/plain; charset=utf-8")
	expect.String(w.Header().Get("Expires")).Not().ToBe(t, "")
	expect.Number(w.Body.Len()).ToBe(t, 0)

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/foo/baz.png", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	expect.Number(w.Code).ToBe(t, 404)
}

func TestHandlerFunc_with_EmbedFS(t *testing.T) {
	maxAge := time.Hour

	const assetPath = "/files/*"

	sub, err := fs.Sub(testdata.TestDataFS, "assets")
	expect.Error(err).Not().ToHaveOccurred(t)

	h := echo_adapter.NewAssetHandlerIoFS(sub).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1)

	router := echo.New()
	// ... add other routes / handlers / middleware as required
	h.Register(router, assetPath)

	r, _ := http.NewRequest(http.MethodGet, "http://localhost/files/101/js/script1.js", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	expect.Number(w.Code).ToBe(t, 200)
	expect.String(w.Header().Get("Content-Type")).ToBe(t, javascriptMimeType)
	expect.String(w.Header().Get("Expires")).Not().ToBe(t, "")
	expect.Number(w.Body.Len()).ToBe(t, 19)

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/js/script1.js", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	expect.Number(w.Code).ToBe(t, 200)
	expect.String(w.Header().Get("Content-Type")).ToBe(t, javascriptMimeType)
	expect.String(w.Header().Get("Expires")).Not().ToBe(t, "")
	expect.Number(w.Body.Len()).ToBe(t, 0)

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/img/baz.png", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	expect.Number(w.Code).ToBe(t, 404)
}
