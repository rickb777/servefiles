package gin_adapter

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rickb777/servefiles/v3"
	"github.com/spf13/afero"
)

// GinAssets is merely an adapter for servefiles.Assets with the same API and with an
// additional HandlerFunc method.
type GinAssets servefiles.Assets

// NewAssetHandler creates an Assets value. The parameter is the directory containing the asset files;
// this can be absolute or relative to the directory in which the server process is started.
//
// This function cleans (i.e. normalises) the asset path.
func NewAssetHandler(assetPath string) *GinAssets {
	return (*GinAssets)(servefiles.NewAssetHandler(assetPath))
}

// NewAssetHandlerFS creates an Assets value for a given filesystem.
func NewAssetHandlerFS(fs afero.Fs) *GinAssets {
	return (*GinAssets)(servefiles.NewAssetHandlerFS(fs))
}

// StripOff alters the handler to strip off a specified number of segments from the path before
// looking for the matching asset. For example, if StripOff(2) has been applied, the requested
// path "/a/b/c/d/doc.js" would be shortened to "c/d/doc.js".
//
// The returned handler is a new copy of the original one.
func (a GinAssets) StripOff(unwantedPrefixSegments int) *GinAssets {
	return (*GinAssets)((servefiles.Assets)(a).StripOff(unwantedPrefixSegments))
}

// WithMaxAge alters the handler to set the specified max age on the served assets.
//
// The returned handler is a new copy of the original one.
func (a GinAssets) WithMaxAge(maxAge time.Duration) *GinAssets {
	return (*GinAssets)((servefiles.Assets)(a).WithMaxAge(maxAge))
}

// WithNotFound alters the handler so that 404-not found cases are passed to a specified
// handler. Without this, the default handler is the one provided in the net/http package.
//
// The returned handler is a new copy of the original one.
func (a GinAssets) WithNotFound(notFound http.Handler) *GinAssets {
	a.NotFound = notFound
	return &a
}

// HandlerFunc gets the asset handler as a Gin handler. The handler is
// registered using a catch-all path such as "/files/*filepath". The name
// of the catch-all parameter is passed in here (for example "filepath").
func (a *GinAssets) HandlerFunc(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		req.URL.Path = c.Param(paramName)
		(*servefiles.Assets)(a).ServeHTTP(c.Writer, c.Request)
	}
}
