package servefiles_test

import (
	"log"
	"net/http"
	"time"

	"github.com/spf13/afero"

	"github.com/rickb777/servefiles/v3"
)

func ExampleNewAssetHandler_simple_web_server() {
	// where the assets are stored (replace as required)
	localPath := "."

	// how long we allow user agents to cache assets
	// (this is in addition to conditional requests, see
	// RFC7234 https://tools.ietf.org/html/rfc7234#section-5.2.2.8)
	maxAge := time.Hour

	h := servefiles.NewAssetHandler(localPath).WithMaxAge(maxAge)

	log.Fatal(http.ListenAndServe(":8080", h))
}

func ExampleNewAssetHandlerFS_simple_web_server() {
	// where the assets are stored (replace as required)
	fs := afero.NewOsFs()

	// how long we allow user agents to cache assets
	// (this is in addition to conditional requests, see
	// RFC7234 https://tools.ietf.org/html/rfc7234#section-5.2.2.8)
	maxAge := time.Hour

	h := servefiles.NewAssetHandlerFS(fs).WithMaxAge(maxAge)

	log.Fatal(http.ListenAndServe(":8080", h))
}
