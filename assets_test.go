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
	"net/http"
	"net/http/httptest"
	. "net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var emptyStrings []string

func init() {
	err := os.Chdir("test")
	if err != nil {
		panic(err)
	}
}

func TestRemoveUnwantedSegments(t *testing.T) {
	cases := []struct {
		n               int
		input, expected string
	}{
		{0, "/a/b/c/x.png", "a/b/c/x.png"},
		{1, "/a/b/c/x.png", "b/c/x.png"},
		{2, "/a/b/c/x.png", "c/x.png"},
		{3, "/a/b/c/x.png", "x.png"},
		{3, "/a/b/c/", ""},
	}

	for _, test := range cases {
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(time.Hour)

		p0 := a.removeUnwantedSegments(test.input)

		isEqual(t, p0, test.expected, test.input)
	}
}

func TestChooseResourceDirs(t *testing.T) {
	cases := []struct {
		n                       int
		expectEtag              bool
		url, path, cacheControl string
	}{
		{0, true, "http://localhost:8001/", "assets/", "public, maxAge=1"},
		{0, false, "http://localhost:8001/img/", "assets/img/", "public, maxAge=1"},
		{3, false, "http://localhost:8001/x/y/z/img/", "assets/img/", "public, maxAge=1"},
	}

	for _, test := range cases {
		etag := ""
		if test.expectEtag {
			etag = etagFor(test.path + "index.html")
		}
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(time.Second)
		headers := make(http.Header)

		resource, code, message := a.chooseResource(headers, request)

		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGte(t, len(headers["Expires"][0]), 25, test.path)
		//fmt.Println(headers["Expires"])
		isEqual(t, resource, test.path + "index.html", test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		if test.expectEtag {
			isEqual(t, headers["Etag"], []string{etag}, test.path)
		} else {
			isEqual(t, headers["Etag"], emptyStrings, test.path)
		}
	}
}

func TestChooseResourceSimpleNoGzip(t *testing.T) {
	cases := []struct {
		n                       int
		maxAge                  time.Duration
		url, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=1"},
		{0, 3671, "http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=3671"},
		{3, 3671, "http://localhost:8001/x/y/z/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=3671"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		headers := make(http.Header)

		resource, code, message := a.chooseResource(headers, request)

		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGte(t, len(headers["Expires"][0]), 25, test.path)
		//fmt.Println(headers["Expires"])
		isEqual(t, resource, test.path, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
	}
}

func TestChooseResourceSimpleNonExistent(t *testing.T) {
	cases := []struct {
		n                       int
		maxAge                  time.Duration
		url, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/img/nonexisting.png", "assets/img/nonexisting.png", "public, maxAge=1"},
		{1, 1, "http://localhost:8001/a/img/nonexisting.png", "assets/img/nonexisting.png", "public, maxAge=1"},
		{2, 1, "http://localhost:8001/a/b/img/nonexisting.png", "assets/img/nonexisting.png", "public, maxAge=1"},
	}

	for _, test := range cases {
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		headers := make(http.Header)

		resource, code, message := a.chooseResource(headers, request)

		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers), 2, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGte(t, len(headers["Expires"][0]), 25, test.path)
		isEqual(t, resource, test.path, test.path)
	}
}

func TestServeHTTP200WithGzipAndGzipWithAcceptHeader(t *testing.T) {
	cases := []struct {
		n                             int
		maxAge                        time.Duration
		url, mime, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css.gz", "public, maxAge=1"},
		{2, 1, "http://localhost:8001/a/b/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css.gz", "public, maxAge=1"},
		{0, 1, "http://localhost:8001/js/script1.js", "application/javascript", "assets/js/script1.js.gz", "public, maxAge=1"},
		{2, 1, "http://localhost:8001/a/b/js/script1.js", "application/javascript", "assets/js/script1.js.gz", "public, maxAge=1"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "xxx, gzip, zzz")
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 200, test.path)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		isGte(t, len(headers), 7, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Content-Type"], []string{test.mime}, test.path)
		isEqual(t, headers["X-Content-Type-Options"], []string{"nosniff"}, test.path)
		isEqual(t, headers["Content-Encoding"], []string{"gzip"}, test.path)
		isEqual(t, headers["Vary"], []string{"Accept-Encoding"}, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGte(t, len(headers["Expires"][0]), 25, test.path)
	}
}

func TestServeHTTP200WithGzipButNoAcceptHeader(t *testing.T) {
	cases := []struct {
		n                             int
		maxAge                        time.Duration
		url, mime, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css", "public, maxAge=1"},
		{2, 2, "http://localhost:8001/a/b/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css", "public, maxAge=2"},
		{0, 3, "http://localhost:8001/js/script1.js", "application/javascript", "assets/js/script1.js", "public, maxAge=3"},
		{2, 4, "http://localhost:8001/a/b/js/script1.js", "application/javascript", "assets/js/script1.js", "public, maxAge=4"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "xxx, yyy, zzz")
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 200, test.path)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		isGte(t, len(headers), 6, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Content-Type"], []string{test.mime}, test.path)
		isEqual(t, headers["Content-Encoding"], emptyStrings, test.path)
		isEqual(t, headers["Vary"], emptyStrings, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGte(t, len(headers["Expires"][0]), 25, test.path)
	}
}

func TestServeHTTP200WithGzipAcceptHeaderButNoGzippedFile(t *testing.T) {
	cases := []struct {
		n                             int
		maxAge                        time.Duration
		url, mime, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/css/style2.css", "text/css; charset=utf-8", "assets/css/style2.css", "public, maxAge=1"},
		{2, 2, "http://localhost:8001/a/b/css/style2.css", "text/css; charset=utf-8", "assets/css/style2.css", "public, maxAge=2"},
		{0, 3, "http://localhost:8001/js/script2.js", "application/javascript", "assets/js/script2.js", "public, maxAge=3"},
		{2, 4, "http://localhost:8001/a/b/js/script2.js", "application/javascript", "assets/js/script2.js", "public, maxAge=4"},
		{0, 5, "http://localhost:8001/img/sort_asc.png", "image/png", "assets/img/sort_asc.png", "public, maxAge=5"},
		{2, 6, "http://localhost:8001/a/b/img/sort_asc.png", "image/png", "assets/img/sort_asc.png", "public, maxAge=6"},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "xxx, gzip, zzz")
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 200, test.path)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		isGte(t, len(headers), 6, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Content-Type"], []string{test.mime}, test.path)
		isEqual(t, headers["Content-Encoding"], emptyStrings, test.path)
		isEqual(t, headers["Vary"], emptyStrings, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGte(t, len(headers["Expires"][0]), 25, test.path)
	}
}

type h404 struct{}

func (h *h404) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(404)
	w.Write([]byte("<html>foo</html>"))
}

func Test404Handler(t *testing.T) {
	cases := []struct {
		path, conType, response string
		notFound                http.Handler
	}{
		{"/img/nonexisting.png", "text/plain; charset=utf-8", "404 page not found\n", nil},
		{"/img/nonexisting.png", "text/html", "<html>foo</html>", &h404{}},
	}

	for _, test := range cases {
		url := mustUrl("http://localhost:8001" + test.conType)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").WithNotFound(test.notFound)
		isEqual(t, a.NotFound, test.notFound, test.conType)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 404, test.conType)
		isEqual(t, w.Header()["Content-Type"], []string{test.conType}, test.conType)
		isEqual(t, w.Body.String(), test.response, test.conType)
	}
}

func TestServeHTTP304(t *testing.T) {
	cases := []struct {
		url, mime, path string
		notFound        http.Handler
	}{
		{"http://localhost:8001/css/style2.css", "text/css; charset=utf-8", "assets/css/style2.css", nil},
		{"http://localhost:8001/js/script2.js", "application/javascript", "assets/js/script2.js", nil},
		{"http://localhost:8001/img/sort_asc.png", "image/png", "assets/img/sort_asc.png", nil},
		{"http://localhost:8001/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css", nil},
		{"http://localhost:8001/js/script1.js", "application/javascript", "assets/js/script1.js", nil},

		{"http://localhost:8001/css/style2.css", "text/css; charset=utf-8", "assets/css/style2.css", &h404{}},
		{"http://localhost:8001/js/script2.js", "application/javascript", "assets/js/script2.js", &h404{}},
		{"http://localhost:8001/img/sort_asc.png", "image/png", "assets/img/sort_asc.png", &h404{}},
		{"http://localhost:8001/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css", &h404{}},
		{"http://localhost:8001/js/script1.js", "application/javascript", "assets/js/script1.js", &h404{}},
	}

	for _, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("If-None-Match", etag)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").WithNotFound(test.notFound)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 304, test.path)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		isGte(t, len(headers), 1, test.path)
		isEqual(t, headers["Cache-Control"], emptyStrings, test.path)
		isEqual(t, headers["Content-Type"], emptyStrings, test.path)
		isEqual(t, headers["Content-Length"], emptyStrings, test.path)
		isEqual(t, headers["Content-Encoding"], emptyStrings, test.path)
		isEqual(t, headers["Vary"], emptyStrings, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
	}
}

func BenchmarkPathWithGzipAndGzipAcceptHeaderCSS(t *testing.B) {
	url := mustUrl("http://localhost:8001/a/b/css/style1.css")
	header := newHeader("Accept-Encoding", "xxx, gzip, zzz")
	request := &http.Request{Method: "GET", URL: url, Header: header}
	a := NewAssetHandler("./assets/").StripOff(2).WithMaxAge(time.Hour)
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		headers := make(http.Header)
		a.chooseResource(headers, request)
	}
}

func BenchmarkPathWithoutGzip(t *testing.B) {
	url := mustUrl("http://localhost:8001/a/b/css/style1.css")
	header := newHeader("Accept-Encoding", "xxx, yyy, zzz")
	request := &http.Request{Method: "GET", URL: url, Header: header}
	a := NewAssetHandler("./assets/").StripOff(2).WithMaxAge(time.Hour)
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		headers := make(http.Header)
		a.chooseResource(headers, request)
	}
}

func isEqual(t *testing.T, a, b, hint interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %#v; expected %#v - for %v\n", a, b, hint)
	}
}

func isGte(t *testing.T, a, b int, hint interface{}) {
	if a < b {
		t.Errorf("Got %d; expected at least %d - for %v\n", a, b, hint)
	}
}

func mustUrl(s string) *URL {
	parsed, err := Parse(s)
	must(err)
	return parsed
}

func newHeader(s string, v string) http.Header {
	header := make(http.Header)
	header[s] = []string{v}
	return header
}

// checkErrPanic abort the program on error, printing a stack trace.
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
	if strings.HasSuffix(name, ".gz") {
		t = "W/" // weak etag
	}
	return fmt.Sprintf(`%s"%x-%x"`, t, d.ModTime().Unix(), d.Size())
}
