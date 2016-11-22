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
	"net/http"
	. "net/url"
	"os"
	"reflect"
	"testing"
	"time"
	"fmt"
	"strings"
)

var emptyStrings []string

func init() {
	err := os.Chdir("test")
	if err != nil {
		panic(err)
	}
}

func TestMapper(t *testing.T) {
	a0 := AssetHandler(0, "./assets/", time.Hour)
	p0 := a0.removeUnwantedSegments("/a/b/c/x.png")
	isEqual(t, p0, "a/b/c/x.png", "a/b/c/x.png")

	a1 := AssetHandler(1, "./assets/", time.Hour)
	p1 := a1.removeUnwantedSegments("/a/b/c/x.png")
	isEqual(t, p1, "b/c/x.png", "b/c/x.png")

	a2 := AssetHandler(2, "./assets/", time.Hour)
	p2 := a2.removeUnwantedSegments("/a/b/c/x.png")
	isEqual(t, p2, "c/x.png", "c/x.png")
}

func TestSimpleNoGzip(t *testing.T) {
	cases := []struct {
		n int
		maxAge time.Duration
		url, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=1"},
		{0, 3671, "http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=3671"},
	}

	for _, test := range cases {
		etag := etag(test.path)
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := AssetHandler(test.n, "./assets/", test.maxAge * time.Second)
		headers := make(http.Header)
		resource, code, message := a.chooseResource(headers, request)
		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGt(t, len(headers["Expires"][0]), 25, test.path)
		//fmt.Println(headers["Expires"])
		isEqual(t, resource, test.path, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
	}
}

func TestSimpleNonExistent(t *testing.T) {
	cases := []struct {
		n int
		maxAge time.Duration
		url, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/img/nonexisting.png", "assets/img/nonexisting.png", "public, maxAge=1"},
		{1, 1, "http://localhost:8001/a/img/nonexisting.png", "assets/img/nonexisting.png", "public, maxAge=1"},
		{2, 1, "http://localhost:8001/a/b/img/nonexisting.png", "assets/img/nonexisting.png", "public, maxAge=1"},
	}

	for _, test := range cases {
		url := mustUrl(test.url)
		request := &http.Request{Method: "GET", URL: url}
		a := AssetHandler(test.n, "./assets/", test.maxAge * time.Second)
		headers := make(http.Header)
		resource, code, message := a.chooseResource(headers, request)
		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers), 2, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGt(t, len(headers["Expires"][0]), 25, test.path)
		isEqual(t, resource, test.path, test.path)
	}
}

func TestPathWithGzipAndGzipWithAcceptHeader(t *testing.T) {
	cases := []struct {
		n int
		maxAge time.Duration
		url, mime, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css.gz", "public, maxAge=1"},
		{2, 1, "http://localhost:8001/a/b/css/style1.css", "text/css; charset=utf-8", "assets/css/style1.css.gz", "public, maxAge=1"},
		{0, 1, "http://localhost:8001/js/script1.js", "application/javascript", "assets/js/script1.js.gz", "public, maxAge=1"},
		{2, 1, "http://localhost:8001/a/b/js/script1.js", "application/javascript", "assets/js/script1.js.gz", "public, maxAge=1"},
	}

	for _, test := range cases {
		etag := etag(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "xxx, gzip, zzz")
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := AssetHandler(test.n, "./assets/", test.maxAge * time.Second)
		headers := make(http.Header)
		resource, code, message := a.chooseResource(headers, request)
		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers), 6, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Content-Type"], []string{test.mime}, test.path)
		isEqual(t, headers["Content-Encoding"], []string{"gzip"}, test.path)
		isEqual(t, headers["Vary"], []string{"Accept-Encoding"}, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGt(t, len(headers["Expires"][0]), 25, test.path)
		isEqual(t, resource, test.path, test.path)
	}
}

func TestPathWithGzipAndGzipNoAcceptHeader(t *testing.T) {
	cases := []struct {
		n int
		maxAge time.Duration
		url, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/css/style1.css", "assets/css/style1.css", "public, maxAge=1"},
		{2, 2, "http://localhost:8001/a/b/css/style1.css", "assets/css/style1.css", "public, maxAge=2"},
		{0, 3, "http://localhost:8001/js/script1.js", "assets/js/script1.js", "public, maxAge=3"},
		{2, 4, "http://localhost:8001/a/b/js/script1.js", "assets/js/script1.js", "public, maxAge=4"},
	}

	for _, test := range cases {
		etag := etag(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "xxx, yyy, zzz")
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := AssetHandler(test.n, "./assets/", test.maxAge * time.Second)
		headers := make(http.Header)
		resource, code, message := a.chooseResource(headers, request)
		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers), 3, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Content-Type"], emptyStrings, test.path)
		isEqual(t, headers["Content-Encoding"], emptyStrings, test.path)
		isEqual(t, headers["Vary"], emptyStrings, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGt(t, len(headers["Expires"][0]), 25, test.path)
		isEqual(t, resource, test.path, test.path)
	}
}

func TestPathWithGzipAcceptHeaderButNoGzippedFile(t *testing.T) {
	cases := []struct {
		n int
		maxAge time.Duration
		url, path, cacheControl string
	}{
		{0, 1, "http://localhost:8001/css/style2.css", "assets/css/style2.css", "public, maxAge=1"},
		{2, 2, "http://localhost:8001/a/b/css/style2.css", "assets/css/style2.css", "public, maxAge=2"},
		{0, 3, "http://localhost:8001/js/script2.js", "assets/js/script2.js", "public, maxAge=3"},
		{2, 4, "http://localhost:8001/a/b/js/script2.js", "assets/js/script2.js", "public, maxAge=4"},
		{0, 5, "http://localhost:8001/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=5"},
		{2, 6, "http://localhost:8001/a/b/img/sort_asc.png", "assets/img/sort_asc.png", "public, maxAge=6"},
	}

	for _, test := range cases {
		etag := etag(test.path)
		url := mustUrl(test.url)
		header := newHeader("Accept-Encoding", "xxx, gzip, zzz")
		request := &http.Request{Method: "GET", URL: url, Header: header}
		a := AssetHandler(test.n, "./assets/", test.maxAge * time.Second)
		headers := make(http.Header)
		resource, code, message := a.chooseResource(headers, request)
		isEqual(t, code, 0, test.path)
		isEqual(t, message, "", test.path)
		isEqual(t, len(headers), 3, test.path)
		isEqual(t, headers["Cache-Control"], []string{test.cacheControl}, test.path)
		isEqual(t, headers["Content-Type"], emptyStrings, test.path)
		isEqual(t, headers["Content-Encoding"], emptyStrings, test.path)
		isEqual(t, headers["Vary"], emptyStrings, test.path)
		isEqual(t, headers["Etag"], []string{etag}, test.path)
		isEqual(t, len(headers["Expires"]), 1, test.path)
		isGt(t, len(headers["Expires"][0]), 25, test.path)
		isEqual(t, resource, test.path, test.path)
	}
}

func BenchmarkPathWithGzipAndGzipAcceptHeaderCSS(t *testing.B) {
	url := mustUrl("http://localhost:8001/a/b/css/style1.css")
	header := newHeader("Accept-Encoding", "xxx, gzip, zzz")
	request := &http.Request{Method: "GET", URL: url, Header: header}
	a := AssetHandler(2, "./assets/", time.Hour)
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
	a := AssetHandler(2, "./assets/", time.Hour)
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		headers := make(http.Header)
		a.chooseResource(headers, request)
	}
}

func isEqual(t *testing.T, a, b, hint interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Got %#v; expected %#v - at %v\n", a, b, hint)
	}
}

func isGt(t *testing.T, a, b int, hint interface{}) {
	if a <= b {
		t.Errorf("Got %d; expected greater than %d - at %v\n", a, b, hint)
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

func etag(name string) string {
	d := mustStat(name)
	t := ""
	if strings.HasSuffix(name, ".gz") {
		t = "W/" // weak etag
	}
	return fmt.Sprintf(`%s"%x-%x"`, t, d.ModTime().Unix(), d.Size())
}
