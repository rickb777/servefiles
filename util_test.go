package servefiles

import (
	"github.com/rickb777/expect"
	"net/http"
	"testing"
)

func TestHeaderStringer(t *testing.T) {
	h := make(http.Header)
	h.Set(ContentEncoding, "br")
	h.Set(Vary, AcceptEncoding)
	s := headerStringer(h).String()
	expect.String(s).ToBe(t, "[Content-Encoding: br. Vary: Accept-Encoding]")
}
