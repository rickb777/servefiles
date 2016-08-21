// Provides a static asset handler for serving files such as images, stylesheets and javascript code. Care is taken
// to set headers such that the assets will be efficiently cached by browsers and proxies.
//
// The 'far-future' technique can be used. Set a long expiry time, e.g. time.Hour * 24 * 3650
//
// No in-memory caching is performed server-side. This is less necessary due to far-future caching being supported,
// but might be added in future.

package httputil

import (
	"errors"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"fmt"
)

// Assets sets the options for asset handling. Use AssetHandler to create the handler(s) you need.
type Assets struct {
	// Choose a number greater than zero to strip off some leading segments from the URL path. This helps if
	// you want, say, a sequence number in the URL so that only has the effect of managing far-future cache
	// control. Use zero for default behaviour.
	UnwantedPrefixSegments int

	// The directory where the assets reside.
	AssetPath              string

	// Set the expiry duration for assets. This will be set via headers in the response. This should never be
	// negative. Use zero to disable asset caching in clients and proxies.
	MaxAge                 time.Duration

	expiryElasticity       time.Duration
	timestamp              int64
	timestampExpiry        string
	lock                   sync.Mutex
}

// Type conformance proof
var _ http.Handler = &Assets{}

// AssetHandler creates an Assets value. It provides some bounds checking, so use it instead of
// creating Assets values directly.
func AssetHandler(unwantedPrefixSegments int, assetPath string, maxAge time.Duration) *Assets {
	if unwantedPrefixSegments < 0 {
		panic(errors.New("Negative unwantedPrefixSegments"))
	}
	if maxAge < 0 {
		panic(errors.New("Negative maxAge"))
	}
	cleanPath := cleanPathAndAppendSlash(assetPath)
	return &Assets{unwantedPrefixSegments, cleanPath, maxAge, 0, 0, "", sync.Mutex{}}
}

// Calculate the 'Expires' value using an approximation that reduces unimportant re-calculation.
// We don't need to do this accurately because the 'Cache-Control' maxAge value takes precedence
// anyway. So the value is cached and shared between requests for a short while.
func (a *Assets) expires() string {
	if a.expiryElasticity == 0 {
		a.expiryElasticity = 1 + a.MaxAge / 100
	}
	now := time.Now()
	unix := now.Unix()
	if unix > a.timestamp {
		later := now.Add(a.MaxAge + a.expiryElasticity) // add expiryElasticity to avoid negative expiry
		a.lock.Lock()
		defer a.lock.Unlock()
		// cache the formatted string for one second to avoid repeated formatting
		// race condition is ignored here, but note the order below
		a.timestampExpiry = later.Format(time.RFC1123)
		a.timestamp = unix + int64(a.expiryElasticity)
	}
	return a.timestampExpiry
}

func (a *Assets) removeUnwantedSegments(path string) string {
	//log.Printf("mapUrlToAssetPath %s", path)
	for i := a.UnwantedPrefixSegments; i >= 0; i-- {
		slash := strings.IndexByte(path, '/') + 1
		if slash > 0 {
			path = path[slash:]
		}
	}
	return path
}

func listContainsWholeString(header, want string) bool {
	accepted := strings.Split(header, ",")
	for _, encoding := range accepted {
		if strings.TrimSpace(encoding) == want {
			return true
		}
	}
	return false
}

func (a *Assets) chooseResource(header http.Header, req *http.Request) (resource string, code int, message string) {
	resource = a.AssetPath + a.removeUnwantedSegments(req.URL.Path)
	gzipped := resource + ".gz"

	if a.MaxAge > 0 {
		header.Set("Expires", a.expires())
		header.Set("Cache-Control", fmt.Sprintf("public, maxAge=%d", a.MaxAge / time.Second))
	}


	_, err := os.Stat(gzipped)
	if err == nil {
		// gzipped file exists and is readable
		acceptEncoding, ok := req.Header["Accept-Encoding"]
		if ok {
			acceptGzip := listContainsWholeString(acceptEncoding[0], "gzip")
			if acceptGzip {
				ext := filepath.Ext(resource)
				header.Set("Content-Type", mime.TypeByExtension(ext))
				header.Set("Content-Encoding", "gzip")
				header.Add("Vary", "Accept-Encoding")
				return gzipped, 0, ""
			}
		}

	} else if os.IsNotExist(err) {
		// gzipped does not exist; original might but this gets checked later
		return resource, 0, ""

	} else if os.IsPermission(err) {
		// incorrectly assembled gzipped asset is treated as an error
		return resource, http.StatusForbidden, "403 Forbidden"
	}

	// no intervention; the file will be served by
	return resource, 0, ""
}

func (a *Assets) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	resource, code, message := a.chooseResource(w.Header(), req)
	if code >= 400 {
		http.Error(w, message, code)
		return
	}

	// Conditional requests and content negotiation are handled in ServeFile.
	// Note that req.URL remains unchanged, even if prefix stripping is turned on, because the resource is
	// the only value that matters.
	http.ServeFile(w, req, resource)
}

func cleanPathAndAppendSlash(s string) string {
	clean := path.Clean(s)
	return string(append([]byte(clean), '/'))
}
