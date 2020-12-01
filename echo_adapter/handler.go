package echo_adapter

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rickb777/servefiles/v3"
	"github.com/spf13/afero"
)

// EchoAssets is merely an adapter for servefiles.Assets with the same API and with an
// additional HandlerFunc method.
type EchoAssets servefiles.Assets

// NewAssetHandler creates an Assets value. The parameter is the directory containing the asset files;
// this can be absolute or relative to the directory in which the server process is started.
//
// This function cleans (i.e. normalises) the asset path.
func NewAssetHandler(assetPath string) *EchoAssets {
	return (*EchoAssets)(servefiles.NewAssetHandler(assetPath))
}

// NewAssetHandlerFS creates an Assets value for a given filesystem.
func NewAssetHandlerFS(fs afero.Fs) *EchoAssets {
	return (*EchoAssets)(servefiles.NewAssetHandlerFS(fs))
}

// StripOff alters the handler to strip off a specified number of segments from the path before
// looking for the matching asset. For example, if StripOff(2) has been applied, the requested
// path "/a/b/c/d/doc.js" would be shortened to "c/d/doc.js".
//
// The returned handler is a new copy of the original one.
func (a EchoAssets) StripOff(unwantedPrefixSegments int) *EchoAssets {
	return (*EchoAssets)((servefiles.Assets)(a).StripOff(unwantedPrefixSegments))
}

// WithMaxAge alters the handler to set the specified max age on the served assets.
//
// The returned handler is a new copy of the original one.
func (a EchoAssets) WithMaxAge(maxAge time.Duration) *EchoAssets {
	return (*EchoAssets)((servefiles.Assets)(a).WithMaxAge(maxAge))
}

// WithNotFound alters the handler so that 404-not found cases are passed to a specified
// handler. Without this, the default handler is the one provided in the net/http package.
//
// The returned handler is a new copy of the original one.
func (a EchoAssets) WithNotFound(notFound http.Handler) *EchoAssets {
	a.NotFound = notFound
	return &a
}

// HandlerFunc gets the asset handler as an Echo handler. The handler is
// registered using a catch-all path such as "/files/*". The same
// match-any pattern can be passed in, in which case it is stripped off
// the leading part of the URL path seem by the asset handler.
func (a *EchoAssets) HandlerFunc(path string) echo.HandlerFunc {
	trim := 0
	if strings.HasSuffix(path, "/*") {
		trim = len(path) - 2
	} else if path != "" {
		panic("Path must end /* or be blank")
	}

	return func(c echo.Context) error {
		req := c.Request()
		req.URL.Path = req.URL.Path[trim:]
		(*servefiles.Assets)(a).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
