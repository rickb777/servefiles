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

package gin_adapter_test

import (
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rickb777/servefiles/v3/afero2"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/gomega"
	"github.com/rickb777/servefiles/v3/gin_adapter"
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
	// RFC7234 https://tools.ietf.org/html/rfc7234#section-5.2.2.8)
	maxAge := time.Hour

	// define the URL pattern that will be routed to the asset handler
	// (optional)
	const path = "/files/*filepath"

	router := gin.Default()
	// ... add other routes / handlers / middleware as required

	h := gin_adapter.NewAssetHandler(localPath).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1).
		HandlerFunc("filepath")

	router.GET(path, h)
	router.HEAD(path, h)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func TestHandlerAferoFunc(t *testing.T) {
	g := NewGomegaWithT(t)

	maxAge := time.Hour
	files := afero2.AferoAdapter{Inner: afero.NewMemMapFs()}
	files.MkdirAll("/foo/bar", 0755)
	afero.WriteFile(files, "/foo/bar/x.txt", []byte("hello"), 0644)

	const assetPath = "/files/*filepath"

	h := gin_adapter.NewAssetHandlerFS(files).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1).
		HandlerFunc("filepath")

	router := gin.Default()
	// ... add other routes / handlers / middleware as required
	router.GET(assetPath, h)
	router.HEAD(assetPath, h)

	r, _ := http.NewRequest(http.MethodGet, "http://localhost/files/101/foo/bar/x.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	g.Expect(w.Code).To(Equal(200))
	g.Expect(w.Header().Get("Content-Type")).To(Equal("text/plain; charset=utf-8"))
	g.Expect(w.Header().Get("Expires")).NotTo(Equal(""))
	g.Expect(w.Body.Len()).To(Equal(5))

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/foo/bar/x.txt", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	g.Expect(w.Code).To(Equal(200))
	g.Expect(w.Header().Get("Content-Type")).To(Equal("text/plain; charset=utf-8"))
	g.Expect(w.Header().Get("Expires")).NotTo(Equal(""))
	g.Expect(w.Body.Len()).To(Equal(0))

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/foo/baz.png", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	g.Expect(w.Code).To(Equal(404))
}

func TestHandlerFunc_with_EmbedFS(t *testing.T) {
	g := NewGomegaWithT(t)

	maxAge := time.Hour

	const assetPath = "/files/*filepath"

	sub, err := fs.Sub(testdata.TestDataFS, "assets")
	g.Expect(err).NotTo(HaveOccurred())

	h := gin_adapter.NewAssetHandlerIoFS(sub).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1).
		HandlerFunc("filepath")

	router := gin.Default()
	// ... add other routes / handlers / middleware as required
	router.GET(assetPath, h)
	router.HEAD(assetPath, h)

	r, _ := http.NewRequest(http.MethodGet, "http://localhost/files/101/js/script1.js", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	g.Expect(w.Code).To(Equal(200))
	g.Expect(w.Header().Get("Content-Type")).To(Equal(javascriptMimeType))
	g.Expect(w.Header().Get("Expires")).NotTo(Equal(""))
	g.Expect(w.Body.Len()).To(Equal(19))

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/js/script1.js", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	g.Expect(w.Code).To(Equal(200))
	g.Expect(w.Header().Get("Content-Type")).To(Equal(javascriptMimeType))
	g.Expect(w.Header().Get("Expires")).NotTo(Equal(""))
	g.Expect(w.Body.Len()).To(Equal(0))

	r, _ = http.NewRequest(http.MethodHead, "http://localhost/files/101/img/baz.png", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	g.Expect(w.Code).To(Equal(404))
}
