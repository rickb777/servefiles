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
	"slices"
	"strings"
)

func commaSeparatedList(s string) List[string] {
	parts := strings.Split(s, ",")
	list := make(List[string], len(parts))
	for i, part := range parts {
		list[i] = strings.TrimSpace(part)
	}
	return list
}

type List[T comparable] []T

func (list List[T]) Contains(wanted T) bool {
	for _, part := range list {
		if part == wanted {
			return true
		}
	}
	return false
}

//-------------------------------------------------------------------------------------------------

type code int

const (
	Directory          code = 0
	OK                 code = 200
	Forbidden          code = 403
	NotFound           code = 404
	MethodNotAllowed   code = 405
	ServiceUnavailable code = 503
)

func (code code) String() string {
	switch code {
	case OK:
		return "200 OK"
	case Forbidden:
		return "403 Forbidden"
	case NotFound:
		return "404 Not found"
	case MethodNotAllowed:
		return "405 Method Not Allowed"
	case ServiceUnavailable:
		return "503 Service unavailable"
	}
	panic(code)
}

//-------------------------------------------------------------------------------------------------

// headerStringer makes debug output clearer, without imposing a performance hit if debugging
// is not enabled.
type headerStringer http.Header

func (h headerStringer) String() string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	buf := &strings.Builder{}
	buf.WriteString("[")
	spacer := ""
	for _, k := range keys {
		buf.WriteString(spacer)
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(strings.Join(h[k], ", "))
		spacer = ". "
	}
	buf.WriteString("]")
	return buf.String()
}
