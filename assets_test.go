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
	mustChdir("test")
}

func TestRemoveUnwantedSegments(t *testing.T) {
	cases := []struct {
		n               int
		input, expected string
	}{
		{0, "/a/b/c/x.png", "/a/b/c/x.png"},
		{1, "/a/b/c/x.png", "/b/c/x.png"},
		{2, "/a/b/c/x.png", "/c/x.png"},
		{3, "/a/b/c/x.png", "/x.png"},
		{4, "/a/b/c/x.png", "/"},
		{3, "/a/b/c/", "/"},
		{4, "/a/b/c/", "/"},
	}

	for _, test := range cases {
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(time.Hour)

		p0 := a.removeUnwantedSegments(test.input)

		isEqual(t, p0, test.expected, test.input)
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

	for i, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge * time.Second)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 200, i)
		//isEqual(t, message, "", test.path)
		isEqual(t, len(w.Header()["Expires"]), 1, i)
		isGte(t, len(w.Header()["Expires"][0]), 25, i)
		//fmt.Println(headers["Expires"])
		isEqual(t, w.Header()["Cache-Control"], []string{test.cacheControl}, i)
		isEqual(t, w.Header()["Etag"], []string{etag}, i)
		isEqual(t, w.Body.Len(), 160, i)
	}
}

func TestChooseResourceSimpleNonExistent(t *testing.T) {
	cases := []struct {
		n      int
		maxAge time.Duration
		url    string
	}{
		{0, time.Second, "http://localhost:8001/img/nonexisting.png"},
		{1, time.Second, "http://localhost:8001/a/img/nonexisting.png"},
		{2, time.Second, "http://localhost:8001/a/b/img/nonexisting.png"},
	}

	for i, test := range cases {
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").StripOff(test.n).WithMaxAge(test.maxAge)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 404, i)
		//t.Logf("header %v", w.Header())
		isGte(t, len(w.Header()), 4, i)
		isEqual(t, w.Header().Get("Content-Type"), "text/plain; charset=utf-8", i)
		isEqual(t, w.Header().Get("Cache-Control"), "public, maxAge=1", i)
		isGte(t, len(w.Header().Get("Expires")), 25, i)
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
		header := newHeader("Accept-Encoding", "xxxx, gzip, zzz")
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
		header := newHeader("Accept-Encoding", "xxxx, yyy, zzz")
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
		header := newHeader("Accept-Encoding", "xxxx, gzip, zzz")
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

//-------------------------------------------------------------------------------------------------

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
		{"/img/nonexisting.png", "text/plain; charset=utf-8", "404 Not found\n", nil},
		{"/img/nonexisting.png", "text/html", "<html>foo</html>", &h404{}},
	}

	for i, test := range cases {
		url := mustUrl("http://localhost:8001" + test.path)
		request := &http.Request{Method: "GET", URL: url}
		a := NewAssetHandler("./assets/").WithNotFound(test.notFound)
		isEqual(t, a.NotFound, test.notFound, i)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 404, i)
		isEqual(t, w.Header().Get("Content-Type"), test.conType, i)
		isEqual(t, w.Body.String(), test.response, i)
	}
}

func Test403Handling(t *testing.T) {
	cases := []struct {
		path   string
		header http.Header
	}{
		{"http://localhost:8001/css/style1.css", newHeader()},
		{"http://localhost:8001/css/style1.css", newHeader("Accept-Encoding", "gzip")},
	}

	for i, test := range cases {
		url := mustUrl("http://localhost:8001" + test.path)
		request := &http.Request{Method: "GET", URL: url, Header: test.header}
		a := NewAssetHandlerFS(&fs403{os.ErrPermission})
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 403, i)
		isEqual(t, w.Header().Get("Content-Type"), "text/plain; charset=utf-8", i)
		isEqual(t, w.Body.String(), "403 Forbidden\n", i)
	}
}

func Test503Handling(t *testing.T) {
	cases := []struct {
		path   string
		header http.Header
	}{
		{"http://localhost:8001/css/style1.css", newHeader()},
		{"http://localhost:8001/css/style1.css", newHeader("Accept-Encoding", "gzip")},
	}

	for i, test := range cases {
		url := mustUrl("http://localhost:8001" + test.path)
		request := &http.Request{Method: "GET", URL: url, Header: test.header}
		a := NewAssetHandlerFS(&fs403{os.ErrInvalid})
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 503, i)
		isEqual(t, w.Header().Get("Content-Type"), "text/plain; charset=utf-8", i)
		isNotEqual(t, w.Header().Get("Retry-After"), "", i)
		isEqual(t, w.Body.String(), "503 Service unavailable\n", i)
	}
}

//-------------------------------------------------------------------------------------------------

func TestServeHTTP304(t *testing.T) {
	cases := []struct {
		url, path string
		notFound  http.Handler
	}{
		{"http://localhost:8001/css/style1.css", "assets/css/style1.css.gz", nil},
		{"http://localhost:8001/css/style2.css", "assets/css/style2.css", nil},
		{"http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", nil},
		{"http://localhost:8001/js/script1.js", "assets/js/script1.js.gz", nil},
		{"http://localhost:8001/js/script2.js", "assets/js/script2.js", nil},

		{"http://localhost:8001/css/style1.css", "assets/css/style1.css.gz", &h404{}},
		{"http://localhost:8001/css/style2.css", "assets/css/style2.css", &h404{}},
		{"http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", &h404{}},
		{"http://localhost:8001/js/script1.js", "assets/js/script1.js.gz", &h404{}},
		{"http://localhost:8001/js/script2.js", "assets/js/script2.js", &h404{}},
	}

	// net/http serveFiles handles conditional requests according to RFC723x specs.
	// So we only need to check that a conditional request is correctly wired in.

	for i, test := range cases {
		etag := etagFor(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "gzip", "If-None-Match", etag)
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := NewAssetHandler("./assets/").WithNotFound(test.notFound)
		w := httptest.NewRecorder()

		a.ServeHTTP(w, request)

		isEqual(t, w.Code, 304, i)
		headers := w.Header()
		//t.Logf("%+v\n", headers)
		isGte(t, len(headers), 1, i)
		isEqual(t, headers["Cache-Control"], emptyStrings, i)
		isEqual(t, headers["Content-Type"], emptyStrings, i)
		isEqual(t, headers["Content-Length"], emptyStrings, i)
		if strings.HasSuffix(test.path, ".gz") {
			isEqual(t, headers["Content-Encoding"], []string{"gzip"}, i)
			isEqual(t, headers["Vary"], []string{"Accept-Encoding"}, i)
			isEqual(t, headers["Etag"], []string{etag}, i)
		} else {
			isEqual(t, headers["Content-Encoding"], emptyStrings, i)
			isEqual(t, headers["Vary"], emptyStrings, i)
			isEqual(t, headers["Etag"], []string{etag}, i)
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
		{1, "a/css/style1.css", "gzip", "", 200},                           // has Gzip
		{2, "a/b/css/style1.css", "gzip", "", 200},                         // has Gzip
		{2, "a/b/css/style1.css", "xxxx", "", 200},                         // has Gzip
		{2, "a/b/css/style1.css", "gzip", "assets/css/style1.css.gz", 304}, // has Gzip
		{2, "a/b/css/style1.css", "xxxx", "assets/css/style1.css", 304},    // has Gzip

		{2, "a/b/css/style2.css", "gzip", "", 200},
		{2, "a/b/css/style2.css", "xxxx", "", 200},
		{2, "a/b/css/style2.css", "gzip", "assets/css/style2.css", 304},
		{2, "a/a/css/style2.css", "xxxx", "assets/css/style2.css", 304},

		{2, "a/b/js/script1.js", "gzip", "", 200},                        // has gzip
		{2, "a/b/js/script1.js", "xxxx", "", 200},                        // has gzip
		{2, "a/b/js/script1.js", "gzip", "assets/js/script1.js.gz", 304}, // has gzip
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
					url := mustUrl("http://localhost:8001/" + test.url)
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

func isEqual(t *testing.T, a, b, hint interface{}) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %#v; expected %#v - for %v\n", a, b, hint)
	}
}

func isNotEqual(t *testing.T, a, b, hint interface{}) {
	t.Helper()
	if reflect.DeepEqual(a, b) {
		t.Errorf("Got %#v; expected something else - for %v\n", a, hint)
	}
}

func isGte(t *testing.T, a, b int, hint interface{}) {
	t.Helper()
	if a < b {
		t.Errorf("Got %d; expected at least %d - for %v\n", a, b, hint)
	}
}

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
	if strings.HasSuffix(name, ".gz") {
		t = "W/" // weak etag
	}
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

func (fs fs403) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fs.err
}
