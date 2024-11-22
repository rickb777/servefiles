package servefiles

import (
	"net/http"
	"testing"
)

func TestHeaderStringer(t *testing.T) {
	h := make(http.Header)
	h.Set(ContentEncoding, "br")
	h.Set(Vary, AcceptEncoding)
	s := headerStringer(h).String()
	isEqual(t, s, "[Content-Encoding: br. Vary: Accept-Encoding]", 0)
}
