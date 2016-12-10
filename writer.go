package servefiles

import (
	"fmt"
	"net/http"
)

type no404Writer struct {
	w    http.ResponseWriter
	Code int
}

var _ http.ResponseWriter = &no404Writer{}

func (ww *no404Writer) Header() http.Header {
	return ww.w.Header()
}

func (ww *no404Writer) WriteHeader(code int) {
	fmt.Printf("WriteHeader %d\n", code)
	ww.Code = code
	ww.w.WriteHeader(code)
}

func (ww *no404Writer) Write(bytes []byte) (int, error) {
	if ww.Code == http.StatusNotFound {
		fmt.Printf("Write ate %s\n", string(bytes))
		return len(bytes), nil
	}
	return ww.w.Write(bytes)
}
