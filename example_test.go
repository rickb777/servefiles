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

package servefiles_test

import (
	"log"
	"net/http"
	"time"

	"github.com/spf13/afero"

	"github.com/rickb777/servefiles/v3"
)

func ExampleNewAssetHandler() {
	// A simple webserver

	// where the assets are stored (replace as required)
	localPath := "./assets"

	// how long we allow user agents to cache assets
	// (this is in addition to conditional requests, see
	// RFC9111 https://www.rfc-editor.org/rfc/rfc9111#section-5.2.2.1)
	maxAge := time.Hour

	h := servefiles.NewAssetHandler(localPath).WithMaxAge(maxAge)

	log.Fatal(http.ListenAndServe(":8080", h))
}

func ExampleNewAssetHandlerFS() {
	// A simple webserver

	// where the assets are stored (replace as required)
	fs := afero.NewOsFs()

	// how long we allow user agents to cache assets
	// (this is in addition to conditional requests, see
	// RFC9111 https://www.rfc-editor.org/rfc/rfc9111#section-5.2.2.1)
	maxAge := time.Hour

	h := servefiles.NewAssetHandlerFS(fs).WithMaxAge(maxAge)

	log.Fatal(http.ListenAndServe(":8080", h))
}
