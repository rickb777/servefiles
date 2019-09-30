package servefiles_test

import (
	"log"
	"net/http"
	"time"

	"github.com/rickb777/servefiles/v3"
)

func ExampleNewAssetHandler() {
	// where the assets are stored
	localPath := "."

	// how long we allow user agents to cache assets
	maxAge := time.Hour

	h := servefiles.NewAssetHandler(localPath).WithMaxAge(maxAge)

	log.Fatal(http.ListenAndServe(":8080", h))
}
