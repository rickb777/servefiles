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
	"fmt"
	"github.com/rickb777/expect"
	"net/http"
	"net/http/httptest"
	. "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

var emptyStrings []string

func mustChdir(dir string) {
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func init() {
	mustChdir("testdata")
}

const (
	cssMimeType        = "text/css; charset=utf-8"
	javascriptMimeType = "text/javascript; charset=utf-8"
)

func TestChooseResourceDirListingIsAllowed(t *testing.T) {
	cases := []struct {
		n            int
		maxAge       time.Duration
		method, url  string
		cacheControl string
	}{
		{maxAge: 1, method: "GET", url: "/css/", cacheControl: "public, max-age=1"},
		{maxAge: 1, method: "HEAD", url: "/css/", cacheControl: "public, max-age=1"},
	}

	for i, test := range cases {
		url := mustUrl(test.url)
		request := &http.Request{Method: test.method, URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		a.DisableDirListing = false
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusOK)
		expect.Slice(w.Header()["Expires"]).Info(i).ToHaveLength(t, 1)
		expect.Slice(w.Header()["Cache-Control"]).Info(i).ToBe(t, test.cacheControl)
		expect.Slice(w.Header()["Etag"]).Info(i).ToBeEmpty(t)
		expect.Number(w.Body.Len()).Info(i).ToBeGreaterThan(t, 10)
	}
}

func TestChooseResourceDirListingIsNotAllowed(t *testing.T) {
	cases := []struct {
		n            int
		maxAge       time.Duration
		method, url  string
		cacheControl string
		body         int
	}{
		{maxAge: 1, method: "GET", url: "/css/", cacheControl: "public, max-age=1", body: 14},
		{maxAge: 1, method: "HEAD", url: "/css/", cacheControl: "public, max-age=1", body: 0},
	}

	for i, test := range cases {
		url := mustUrl(test.url)
		request := &http.Request{Method: test.method, URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		a.DisableDirListing = true
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusNotFound)
		expect.Slice(w.Header()["Expires"]).Info(i).ToBeEmpty(t)
		expect.Slice(w.Header()["Cache-Control"]).Info(i).ToBeEmpty(t)
		expect.Slice(w.Header()["Etag"]).Info(i).ToBeEmpty(t)
		expect.Number(w.Body.Len()).Info(i).ToBe(t, test.body)
	}
}

func TestChooseResourceSimpleDirNoGzip(t *testing.T) {
	cases := []struct {
		n                  int
		maxAge             time.Duration
		method, url        string
		path, cacheControl string
		body               int
		disable            bool
		rHeaderKV          []string
	}{
		{maxAge: 1, method: "GET", url: "/", path: "assets/index.html", cacheControl: "public, max-age=1", body: 36, disable: true},
		{maxAge: 1, method: "GET", url: "/", path: "assets/index.html", cacheControl: "public, max-age=1", body: 36},
		{maxAge: 1, method: "GET", url: "/", path: "assets/index.html", cacheControl: "public, max-age=1", body: 60, rHeaderKV: []string{"Accept-Encoding", "gzip"}},
		{maxAge: 1, method: "HEAD", url: "/", path: "assets/index.html", cacheControl: "public, max-age=1", body: 0},
	}

	for i, test := range cases {
		etag := etagFor(test.path)
		request, _ := http.NewRequest(test.method, test.url, nil)
		for i := 1; i < len(test.rHeaderKV); i += 2 {
			request.Header.Set(test.rHeaderKV[i-1], test.rHeaderKV[i])
			etag = "W/" + etagFor(test.path+".gz")
		}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		a.DisableDirListing = test.disable
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusOK)
		expect.Slice(w.Header()["Expires"]).Info(i).ToHaveLength(t, 1)
		expect.Number(len(w.Header()["Expires"][0])).Info(i).ToBeGreaterThanOrEqualTo(t, 25)
		//fmt.Println(headers["Expires"])
		expect.Slice(w.Header()["Cache-Control"]).Info(i).ToBe(t, test.cacheControl)
		expect.Slice(w.Header()["Etag"]).Info(i).ToBe(t, etag)
		expect.Number(w.Body.Len()).Info(i).ToBe(t, test.body)
	}
}

func TestChooseResourceSimpleNoGzip(t *testing.T) {
	cases := []struct {
		n                  int
		maxAge             time.Duration
		method, url        string
		path, cacheControl string
		body               int
	}{
		{maxAge: 1, method: "GET", url: "/img/sort_asc.png", path: "assets/img/sort_asc.png", cacheControl: "public, max-age=1", body: 160},
		{maxAge: 3671, method: "GET", url: "/img/sort_asc.png", path: "assets/img/sort_asc.png", cacheControl: "public, max-age=3671", body: 160},
		{n: 3, maxAge: 3671, method: "GET", url: "/x/y/z/img/sort_asc.png", path: "assets/img/sort_asc.png", cacheControl: "public, max-age=3671", body: 160},
		{n: 3, maxAge: 3671, method: "HEAD", url: "/x/y/z/img/sort_asc.png", path: "assets/img/sort_asc.png", cacheControl: "public, max-age=3671", body: 0},
	}

	for i, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		request := &http.Request{Method: test.method, URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusOK)
		//expect.String(message).Info(i).ToBe(t, "")
		expect.Slice(w.Header()["Expires"]).Info(i).ToHaveLength(t, 1)
		expect.Number(len(w.Header()["Expires"][0])).Info(i).ToBeGreaterThanOrEqualTo(t, 25)
		expect.Slice(w.Header()["Cache-Control"]).Info(i).ToBe(t, test.cacheControl)
		expect.Slice(w.Header()["Etag"]).Info(i).ToBe(t, etag)
		expect.Number(w.Body.Len()).Info(i).ToBe(t, test.body)
	}
}

func TestChooseResourceSimpleNonExistent(t *testing.T) {
	cases := []struct {
		n      int
		maxAge time.Duration
		url    string
	}{
		{0, time.Second, "/img/nonexisting.png"},
		{1, time.Second, "/a/img/nonexisting.png"},
		{2, time.Second, "/a/b/img/nonexisting.png"},
	}

	for i, test := range cases {
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusNotFound)
		//t.Logf("header %v", w.Header())
		expect.Map(w.Header()).Info(i).ToHaveLength(t, 4)
		expect.String(w.Header().Get("Content-Type")).Info(i).ToBe(t, "text/plain; charset=utf-8")
		expect.String(w.Header().Get("Cache-Control")).Info(i).ToBe(t, "public, max-age=1")
		expect.Number(len(w.Header().Get("Expires"))).Info(i).ToBeGreaterThanOrEqualTo(t, 25)
	}
}

func TestServeHTTP200WithGzipAndGzipWithAcceptHeader(t *testing.T) {
	cases := []struct {
		n                                       int
		maxAge                                  time.Duration
		url, mime, encoding, path, cacheControl string
	}{
		{0, 1, "/css/style1.css", cssMimeType, "xx, gzip, zzz", "assets/css/style1.css.gz", "public, max-age=1"},
		{2, 1, "/a/b/css/style1.css", cssMimeType, "xx, gzip, zzz", "assets/css/style1.css.gz", "public, max-age=1"},
		{0, 1, "/js/script1.js", javascriptMimeType, "xx, gzip, zzz", "assets/js/script1.js.gz", "public, max-age=1"},
		{2, 1, "/a/b/js/script1.js", javascriptMimeType, "xx, gzip, zzz", "assets/js/script1.js.gz", "public, max-age=1"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", test.encoding)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(test.path).ToBe(t, http.StatusOK)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		expect.Number(len(headers)).Info(test.path).ToBeGreaterThanOrEqualTo(t, 7)
		expect.Slice(headers["Cache-Control"]).Info(test.path).ToBe(t, test.cacheControl)
		expect.Slice(headers["Content-Type"]).Info(test.path).ToBe(t, test.mime)
		expect.Slice(headers["X-Content-Type-Options"]).Info(test.path).ToBe(t, "nosniff")
		expect.Slice(headers["Content-Encoding"]).Info(test.path).ToBe(t, "gzip")
		expect.Slice(headers["Vary"]).Info(test.path).ToBe(t, "Accept-Encoding")
		expect.Slice(headers["Etag"]).Info(test.path).ToBe(t, "W/"+etag)
		expect.Slice(headers["Expires"]).Info(test.path).ToHaveLength(t, 1)
		expect.Number(len(headers["Expires"][0])).Info(test.path).ToBeGreaterThanOrEqualTo(t, 25)
	}
}

func TestServeHTTP200WithBrAndBrWithAcceptHeader(t *testing.T) {
	cases := []struct {
		n                                       int
		maxAge                                  time.Duration
		url, mime, encoding, path, cacheControl string
	}{
		{0, 1, "/css/style1.css", cssMimeType, "br, gzip, zzz", "assets/css/style1.css.br", "public, max-age=1"},
		{2, 1, "/a/b/css/style1.css", cssMimeType, "br, gzip, zzz", "assets/css/style1.css.br", "public, max-age=1"},
		{0, 1, "/js/script1.js", javascriptMimeType, "br, gzip, zzz", "assets/js/script1.js.br", "public, max-age=1"},
		{2, 1, "/a/b/js/script1.js", javascriptMimeType, "br, gzip, zzz", "assets/js/script1.js.br", "public, max-age=1"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", test.encoding)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(test.path).ToBe(t, http.StatusOK)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		expect.Number(len(headers)).Info(test.path).ToBeGreaterThanOrEqualTo(t, 7)
		expect.Slice(headers["Cache-Control"]).Info(test.path).ToBe(t, test.cacheControl)
		expect.Slice(headers["Content-Type"]).Info(test.path).ToBe(t, test.mime)
		expect.Slice(headers["X-Content-Type-Options"]).Info(test.path).ToBe(t, "nosniff")
		expect.Slice(headers["Content-Encoding"]).Info(test.path).ToBe(t, "br")
		expect.Slice(headers["Vary"]).Info(test.path).ToBe(t, "Accept-Encoding")
		expect.Slice(headers["Etag"]).Info(test.path).ToBe(t, "W/"+etag)
		expect.Slice(headers["Expires"]).Info(test.path).ToHaveLength(t, 1)
		expect.Number(len(headers["Expires"][0])).Info(test.path).ToBeGreaterThanOrEqualTo(t, 25)
	}
}

func TestServeHTTP200WithGzipButNoAcceptHeader(t *testing.T) {
	cases := []struct {
		n                                       int
		maxAge                                  time.Duration
		url, mime, encoding, path, cacheControl string
	}{
		{0, 1, "/css/style1.css", cssMimeType, "xx, yy, zzz", "assets/css/style1.css", "public, max-age=1"},
		{2, 2, "/a/b/css/style1.css", cssMimeType, "xx, yy, zzz", "assets/css/style1.css", "public, max-age=2"},
		{0, 3, "/js/script1.js", javascriptMimeType, "xx, yy, zzz", "assets/js/script1.js", "public, max-age=3"},
		{2, 4, "/a/b/js/script1.js", javascriptMimeType, "xx, yy, zzz", "assets/js/script1.js", "public, max-age=4"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", test.encoding)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(test.path).ToBe(t, http.StatusOK)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		expect.Number(len(headers)).Info(test.path).ToBeGreaterThanOrEqualTo(t, 6)
		expect.Slice(headers["Cache-Control"]).Info(test.path).ToBe(t, test.cacheControl)
		expect.Slice(headers["Content-Type"]).Info(test.path).ToBe(t, test.mime)
		expect.Slice(headers["Content-Encoding"]).Info(test.path).ToBeEmpty(t)
		expect.Slice(headers["Vary"]).Info(test.path).ToBeEmpty(t)
		expect.Slice(headers["Etag"]).Info(test.path).ToBe(t, etag)
		expect.Slice(headers["Expires"]).Info(test.path).ToHaveLength(t, 1)
		expect.Number(len(headers["Expires"][0])).Info(test.path).ToBeGreaterThanOrEqualTo(t, 25)
	}
}

func TestServeHTTP200WithGzipAcceptHeaderButNoGzippedFile(t *testing.T) {
	cases := []struct {
		n                                       int
		maxAge                                  time.Duration
		url, mime, encoding, path, cacheControl string
	}{
		{0, 1, "/css/style2.css", cssMimeType, "xx, gzip, zzz", "assets/css/style2.css", "public, max-age=1"},
		{0, 1, "/css/style2.css", cssMimeType, "br, gzip, zzz", "assets/css/style2.css", "public, max-age=1"},
		{2, 2, "/a/b/css/style2.css", cssMimeType, "xx, gzip, zzz", "assets/css/style2.css", "public, max-age=2"},
		{2, 2, "/a/b/css/style2.css", cssMimeType, "br, gzip, zzz", "assets/css/style2.css", "public, max-age=2"},
		{0, 3, "/js/script2.js", javascriptMimeType, "xx, gzip, zzz", "assets/js/script2.js", "public, max-age=3"},
		{0, 3, "/js/script2.js", javascriptMimeType, "br, gzip, zzz", "assets/js/script2.js", "public, max-age=3"},
		{2, 4, "/a/b/js/script2.js", javascriptMimeType, "xx, gzip, zzz", "assets/js/script2.js", "public, max-age=4"},
		{2, 4, "/a/b/js/script2.js", javascriptMimeType, "br, gzip, zzz", "assets/js/script2.js", "public, max-age=4"},
		{0, 5, "/img/sort_asc.png", "image/png", "xx, gzip, zzz", "assets/img/sort_asc.png", "public, max-age=5"},
		{0, 5, "/img/sort_asc.png", "image/png", "br, gzip, zzz", "assets/img/sort_asc.png", "public, max-age=5"},
		{2, 6, "/a/b/img/sort_asc.png", "image/png", "xx, gzip, zzz", "assets/img/sort_asc.png", "public, max-age=6"},
		{2, 6, "/a/b/img/sort_asc.png", "image/png", "br, gzip, zzz", "assets/img/sort_asc.png", "public, max-age=6"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", test.encoding)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(test.path).ToBe(t, http.StatusOK)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		expect.Number(len(headers)).Info(test.path).ToBeGreaterThanOrEqualTo(t, 6)
		expect.Slice(headers["Cache-Control"]).Info(test.path).ToBe(t, test.cacheControl)
		expect.Slice(headers["Content-Type"]).Info(test.path).ToBe(t, test.mime)
		expect.Slice(headers["Content-Encoding"]).Info(test.path).ToBeEmpty(t)
		expect.Slice(headers["Vary"]).Info(test.path).ToBeEmpty(t)
		expect.Slice(headers["Etag"]).Info(test.path).ToBe(t, etag)
		expect.Slice(headers["Expires"]).Info(test.path).ToHaveLength(t, 1)
		expect.Number(len(headers["Expires"][0])).Info(test.path).ToBeGreaterThanOrEqualTo(t, 25)
	}
}

//-------------------------------------------------------------------------------------------------

type h4xx struct{ code int }

func (h *h4xx) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(w.Header()) > 0 {
		panic(fmt.Sprintf("still holding headers %+v", w.Header()))
	}
	w.Header().Set(ContentType, "text/html")
	w.WriteHeader(h.code)
	if r.Method != http.MethodHead {
		w.Write([]byte("<html>foo</html>"))
	}
}

func Test405Handling(t *testing.T) {
	cases := []struct {
		method, path      string
		conType, response string
		notAllowed        http.Handler
	}{
		{method: "POST", path: "/img/nonexisting.png", conType: "text/html", response: "<html>foo</html>", notAllowed: &h4xx{code: 405}},
		{method: "POST", path: "/img/nonexisting.png", conType: "text/plain; charset=utf-8", response: "405 Method Not Allowed\n"},
		{method: "PUT", path: "/img/nonexisting.png", conType: "text/plain; charset=utf-8", response: "405 Method Not Allowed\n"},
		{method: "DELETE", path: "/img/nonexisting.png", conType: "text/plain; charset=utf-8", response: "405 Method Not Allowed\n"},
	}

	for i, test := range cases {
		url := mustUrl("" + test.path)
		request := &http.Request{Method: test.method, URL: url}
		a := NewAssetHandler("./assets/").WithMethodNotAllowed(test.notAllowed)
		expect.Any(a.MethodNotAllowed).Info(i).ToBe(t, test.notAllowed)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusMethodNotAllowed)
		expect.String(w.Header().Get("Content-Type")).Info(i).ToBe(t, test.conType)
		expect.String(w.Body.String()).Info(i).ToBe(t, test.response)
	}
}

func Test404Handling(t *testing.T) {
	cases := []struct {
		method, path      string
		conType, response string
		notFound          http.Handler
	}{
		{method: "GET", path: "/img/nonexisting.png", conType: "text/html", response: "<html>foo</html>", notFound: &h4xx{code: 404}},
		{method: "GET", path: "/img/nonexisting.png", conType: "text/plain; charset=utf-8", response: "404 Not found\n"},
	}

	for i, test := range cases {
		url := mustUrl("" + test.path)
		request := &http.Request{Method: test.method, URL: url}
		a := NewAssetHandler("./assets/").WithNotFound(test.notFound)
		expect.Any(a.NotFound).Info(i).ToBe(t, test.notFound)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusNotFound)
		expect.String(w.Header().Get("Content-Type")).Info(i).ToBe(t, test.conType)
		expect.String(w.Body.String()).Info(i).ToBe(t, test.response)
	}
}

func Test403Handling(t *testing.T) {
	cases := []struct {
		path   string
		header http.Header
	}{
		{path: "/css/style1.css", header: newHeader()},
		{path: "/css/style1.css", header: newHeader("Accept-Encoding", "gzip")},
	}

	for i, test := range cases {
		url := mustUrl("" + test.path)
		request := &http.Request{Method: "GET", URL: url, Header: test.header}
		a := NewAssetHandlerFS(&fs403{os.ErrPermission})
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusForbidden)
		expect.String(w.Header().Get("Content-Type")).Info(i).ToBe(t, "text/plain; charset=utf-8")
		expect.String(w.Body.String()).Info(i).ToBe(t, "403 Forbidden\n")
	}
}

func Test503Handling(t *testing.T) {
	cases := []struct {
		path   string
		header http.Header
	}{
		{path: "/css/style1.css", header: newHeader()},
		{path: "/css/style1.css", header: newHeader("Accept-Encoding", "gzip")},
	}

	for i, test := range cases {
		url := mustUrl("" + test.path)
		request := &http.Request{Method: "GET", URL: url, Header: test.header}
		a := NewAssetHandlerFS(&fs403{os.ErrInvalid})
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusServiceUnavailable)
		expect.String(w.Header().Get("Content-Type")).Info(i).ToBe(t, "text/plain; charset=utf-8")
		expect.String(w.Header().Get("Retry-After")).Info(i).Not().ToBe(t, "")
		expect.String(w.Body.String()).Info(i).ToBe(t, "503 Service unavailable\n")
	}
}

//-------------------------------------------------------------------------------------------------

func TestServeHTTP304(t *testing.T) {
	cases := []struct {
		url, path, encoding string
		notFound            http.Handler
	}{
		{url: "/css/style1.css", path: "assets/css/style1.css.gz", encoding: "gzip"},
		{url: "/css/style1.css", path: "assets/css/style1.css.br", encoding: "br"},
		{url: "/css/style2.css", path: "assets/css/style2.css", encoding: "xx"},
		{url: "/img/sort_asc.png", path: "assets/img/sort_asc.png", encoding: "xx"},
		{url: "/js/script1.js", path: "assets/js/script1.js.gz", encoding: "gzip"},
		{url: "/js/script1.js", path: "assets/js/script1.js.br", encoding: "br"},
		{url: "/js/script2.js", path: "assets/js/script2.js", encoding: "xx"},

		{url: "/css/style1.css", path: "assets/css/style1.css.gz", encoding: "gzip", notFound: &h4xx{code: 404}},
		{url: "/css/style1.css", path: "assets/css/style1.css.br", encoding: "br", notFound: &h4xx{code: 404}},
		{url: "/css/style2.css", path: "assets/css/style2.css", encoding: "xx", notFound: &h4xx{code: 404}},
		{url: "/img/sort_asc.png", path: "assets/img/sort_asc.png", encoding: "xx", notFound: &h4xx{code: 404}},
		{url: "/js/script1.js", path: "assets/js/script1.js.gz", encoding: "gzip", notFound: &h4xx{code: 404}},
		{url: "/js/script1.js", path: "assets/js/script1.js.br", encoding: "br", notFound: &h4xx{code: 404}},
		{url: "/js/script2.js", path: "assets/js/script2.js", encoding: "xx", notFound: &h4xx{code: 404}},
	}

	// net/http serveFiles handles conditional requests according to RFC9110 specs.
	// So we only need to check that a conditional request is correctly wired in.

	for i, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", test.encoding, "If-None-Match", etag)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").WithNotFound(test.notFound)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		expect.Number(w.Code).Info(i).ToBe(t, http.StatusNotModified)
		expect.String(request.URL.Path).Info(i).ToBe(t, test.url)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		expect.Number(len(headers), 1, i)
		expect.Slice(headers["Cache-Control"]).Info(i).ToBeEmpty(t)
		expect.Slice(headers["Content-Type"]).Info(i).ToBeEmpty(t)
		expect.Slice(headers["Content-Length"]).Info(i).ToBeEmpty(t)
		expect.Slice(headers["Content-Encoding"]).Info(i).ToBeEmpty(t)
		if strings.HasSuffix(test.path, ".gz") {
			expect.Slice(headers["Vary"]).Info(i).ToBe(t, "Accept-Encoding")
			expect.Slice(headers["Etag"]).Info(i).ToBe(t, "W/"+etag)
		} else if strings.HasSuffix(test.path, ".br") {
			expect.Slice(headers["Vary"]).Info(i).ToBe(t, "Accept-Encoding")
			expect.Slice(headers["Etag"]).Info(i).ToBe(t, "W/"+etag)
		} else {
			expect.Slice(headers["Vary"]).Info(i).ToBeEmpty(t)
			expect.Slice(headers["Etag"]).Info(i).ToBe(t, etag)
		}
	}
}

//-------------------------------------------------------------------------------------------------

func Benchmark(t *testing.B) {
	t.StopTimer()

	cases := []struct {
		strip       int
		url, enc    string
		sendEtagFor string
		code        int
	}{
		{0, "css/style1.css", "gzip", "", 200},                             // has Gzip
		{0, "css/style1.css", "br", "", 200},                               // has Brotli
		{1, "a/css/style1.css", "gzip", "", 200},                           // has Gzip
		{1, "a/css/style1.css", "br", "", 200},                             // has Brotli
		{2, "a/b/css/style1.css", "gzip", "", 200},                         // has Gzip
		{2, "a/b/css/style1.css", "br", "", 200},                           // has Brotli
		{2, "a/b/css/style1.css", "xxxx", "", 200},                         // has Gzip
		{2, "a/b/css/style1.css", "gzip", "assets/css/style1.css.gz", 304}, // has Gzip
		{2, "a/b/css/style1.css", "br", "assets/css/style1.css.br", 304},   // has Brotli
		{2, "a/b/css/style1.css", "xxxx", "assets/css/style1.css", 304},    // has Gzip

		{2, "a/b/css/style2.css", "gzip", "", 200},
		{2, "a/b/css/style2.css", "xxxx", "", 200},
		{2, "a/b/css/style2.css", "gzip", "assets/css/style2.css", 304},
		{2, "a/a/css/style2.css", "xxxx", "assets/css/style2.css", 304},

		{2, "a/b/js/script1.js", "gzip", "", 200},                        // has gzip
		{2, "a/b/js/script1.js", "br", "", 200},                          // has Brotli
		{2, "a/b/js/script1.js", "xxxx", "", 200},                        // has gzip
		{2, "a/b/js/script1.js", "gzip", "assets/js/script1.js.gz", 304}, // has gzip
		{2, "a/b/js/script1.js", "br", "assets/js/script1.js.br", 304},   // has Brotli
		{2, "a/a/js/script1.js", "xxxx", "assets/js/script1.js", 304},    // has gzip

		{2, "a/b/js/script2.js", "gzip", "", 200},
		{2, "a/b/js/script2.js", "xxxx", "", 200},
		{2, "a/b/js/script2.js", "gzip", "assets/js/script2.js", 304},
		{2, "a/a/js/script2.js", "xxxx", "assets/js/script2.js", 304},

		{2, "a/b/img/sort_asc.png", "gzip", "", 200},
		{2, "a/b/img/sort_asc.png", "xxxx", "", 200},
		{2, "a/b/img/sort_asc.png", "gzip", "assets/img/sort_asc.png", 304},
		{2, "a/a/img/sort_asc.png", "xxxx", "assets/img/sort_asc.png", 304},

		{2, "a/b/img/nonexisting.png", "gzip", "", 404},
		{2, "a/b/img/nonexisting.png", "xxxx", "", 404},
	}

	ages := []time.Duration{0, time.Hour}

	for _, test := range cases {
		header := newHeader("Accept-Encoding", test.enc)
		etagOn := "no-etag"
		if test.sendEtagFor != "" {
			header = newHeader("Accept-Encoding", test.enc, "If-None-Match", etagFor(test.sendEtagFor))
			etagOn = "etag"
		}

		for _, age := range ages {
			a := NewAssetHandler("./assets/").StripOff(test.strip).WithMaxAge(age)

			t.Run(fmt.Sprintf("%s~%s~%v~%d~%v", test.url, test.enc, etagOn, test.code, age), func(b *testing.B) {
				b.StopTimer()

				for i := 0; i < b.N; i++ {
					url := mustUrl("/" + test.url)
					request := &http.Request{Method: "GET", URL: url, Header: header}
					w := httptest.NewRecorder()

					b.StartTimer()
					a.ServeHTTP(w, request)
					b.StopTimer()

					if w.Code != test.code {
						b.Fatalf("Expected %d but got %d", test.code, w.Code)
					}
				}
			})
		}
	}
}

//-------------------------------------------------------------------------------------------------

func mustUrl(s string) *URL {
	parsed, err := Parse(s)
	must(err)
	return parsed
}

func newHeader(kv ...string) http.Header {
	header := make(http.Header)
	for i, x := range kv {
		if i%2 == 0 {
			header[x] = []string{kv[i+1]}
		}
	}
	return header
}

// must abort the program on error, printing a stack trace.
func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustStat(name string) os.FileInfo {
	d, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	return d
}

func etagFor(name string) string {
	d := mustStat(name)
	t := ""
	return fmt.Sprintf(`%s"%x-%x"`, t, d.ModTime().Unix(), d.Size())
}

//-------------------------------------------------------------------------------------------------

type fs403 struct {
	err error
}

func (fs fs403) Create(name string) (afero.File, error) {
	return nil, fs.err
}

func (fs fs403) Mkdir(name string, perm os.FileMode) error {
	return fs.err
}

func (fs fs403) MkdirAll(path string, perm os.FileMode) error {
	return fs.err
}

func (fs fs403) Open(name string) (afero.File, error) {
	return nil, fs.err
}

func (fs fs403) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, fs.err
}

func (fs fs403) Remove(name string) error {
	return fs.err
}

func (fs fs403) RemoveAll(path string) error {
	return fs.err
}

func (fs fs403) Rename(oldname, newname string) error {
	return fs.err
}

func (fs fs403) Stat(name string) (os.FileInfo, error) {
	return nil, fs.err
}

func (fs403) Name() string {
	return "dumb"
}

func (fs fs403) Chmod(name string, mode os.FileMode) error {
	return fs.err
}

func (fs fs403) Chown(name string, uid, gid int) error {
	return fs.err
}

func (fs fs403) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fs.err
}
