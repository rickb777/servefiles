package echo_adapter_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
	"github.com/rickb777/servefiles/v3/echo_adapter"
	"github.com/spf13/afero"
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

func TestHandlerFunc(t *testing.T) {
	g := NewGomegaWithT(t)

	maxAge := time.Hour
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/foo/bar", 0755)
	afero.WriteFile(fs, "/foo/bar/x.txt", []byte("hello"), 0644)

	const assetPath = "/files/*"

	h := echo_adapter.NewAssetHandlerFS(fs).
		WithMaxAge(maxAge).
		WithNotFound(http.NotFoundHandler()). // supply your own
		StripOff(1)

	router := echo.New()
	// ... add other routes / handlers / middleware as required
	h.Register(router, assetPath)

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
