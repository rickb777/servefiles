package servefiles

import (
	"fmt"
	"net/http"
)

type no404Writer struct {
	w           http.ResponseWriter
	Code        int
	codeSent    bool
	headerCache http.Header
}

func newNo404Writer(w http.ResponseWriter) *no404Writer {
	ww := &no404Writer{}
	ww.w = w
	ww.headerCache = http.Header{}
	copyHeaders(ww.w.Header(), ww.headerCache)
	return ww
}

var _ http.ResponseWriter = &no404Writer{}

func (ww *no404Writer) Header() http.Header {
	return ww.headerCache
}

func (ww *no404Writer) WriteHeader(code int) {
	ww.Code = code
}

func (ww *no404Writer) Write(bytes []byte) (int, error) {
	if ww.Code == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		ww.Code = http.StatusOK
	} else if ww.Code == http.StatusNotFound {
		// consume the content from the Go built-in 404 handler
		return len(bytes), nil
	}

	ww.LazyWriteHeaders()
	return ww.w.Write(bytes)
}

func (ww *no404Writer) LazyWriteHeaders() {
	// lazily pass on headers - allows them to be dropped when 404 happens
	if !ww.codeSent {
		ww.codeSent = true
		copyHeaders(ww.headerCache, ww.w.Header())
		ww.w.WriteHeader(ww.Code)
	}
}

func (ww *no404Writer) CloseNotify() <-chan bool {
	c, ok := ww.w.(http.CloseNotifier)
	if ok {
		return c.CloseNotify()
	}
	return nil
}

func (ww *no404Writer) Flush() {
	f, ok := ww.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func (ww *no404Writer) String() string {
	if ww.codeSent {
		return fmt.Sprintf("%d %+v\n", ww.Code, ww.headerCache)
	}
	return fmt.Sprintf("--- %+v\n", ww.headerCache)
}

func copyHeaders(from, to http.Header) {
	for k, v := range from {
		to[k] = v
	}
}
