package servefiles

import (
	"fmt"
	"io/fs"
	"math/rand/v2"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rickb777/path"
)

//-------------------------------------------------------------------------------------------------

// Calculate the 'Expires' value using an approximation that reduces unimportant re-calculation.
// We don't need to do this accurately because the 'Cache-Control' maxAge value takes precedence
// anyway. So the value is cached and shared between requests for a short while.
func (a *Assets) expires() string {
	if a.expiryElasticity == 0 {
		// lazy initialisation
		a.expiryElasticity = 1 + a.MaxAge/100
	}

	now := time.Now().UTC()
	unix := now.Unix()

	if unix > a.timestamp {
		later := now.Add(a.MaxAge + a.expiryElasticity) // add expiryElasticity to avoid negative expiry

		a.lock.Lock()
		defer a.lock.Unlock()

		// cache the formatted string for one second to avoid repeated formatting
		// race condition is ignored here, but note the order below
		a.timestampExpiry = later.Format(time.RFC1123)
		a.timestamp = unix + int64(a.expiryElasticity)

		// ensure that maxAgeS is set
		if a.maxAgeS == 0 {
			a.maxAgeS = int(a.MaxAge / time.Second)
		}
	}

	return a.timestampExpiry
}

//-------------------------------------------------------------------------------------------------

type fileData struct {
	resource string
	code     code
	fi       os.FileInfo
}

func calculateEtag(fi os.FileInfo) string {
	if fi == nil {
		return ""
	}
	return fmt.Sprintf(`"%x-%x"`, fi.ModTime().Unix(), fi.Size())
}

func handleSaturatedServer(wHeader http.Header, resource string) fileData {
	// Possibly the server is under heavy load and ran out of file descriptors
	backoff := 2 + rand.IntN(4) // 2–6 seconds to prevent a stampede
	wHeader.Set("Retry-After", strconv.Itoa(int(backoff)))
	return fileData{resource, ServiceUnavailable, nil}
}

func (a *Assets) checkResource(resource string, wHeader http.Header) fileData {
	d, err := fs.Stat(a.fs, removeLeadingSlash(resource))
	if err != nil {
		if os.IsNotExist(err) {
			// gzipped does not exist; original might but this gets checked later
			return fileData{"", NotFound, nil}

		} else if os.IsPermission(err) {
			// incorrectly assembled gzipped asset is treated as an error
			return fileData{resource, Forbidden, nil}
		}

		return handleSaturatedServer(wHeader, resource)
	}

	if d.IsDir() {
		// directory edge case is simply passed on to the standard library
		return fileData{resource, Directory, nil}
	}

	return fileData{resource, OK, d}
}

func removeLeadingSlash(name string) string {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	return name
}

func removeTrailingSlash(name string) string {
	last := len(name) - 1
	if len(name) > 0 && name[last] == '/' {
		name = name[:last]
	}
	return name
}

func httpError(w http.ResponseWriter, code code, method string) {
	if method == http.MethodHead {
		w.WriteHeader(int(code))
	} else {
		http.Error(w, code.String(), int(code))
	}
}

func (a *Assets) chooseResource(wHeader http.Header, req *http.Request, resource string) (string, code) {

	if strings.HasSuffix(resource, "/") {
		indexPath, indexCode := a.chooseResource(wHeader, req, resource+IndexPage)
		if indexCode == OK {
			if strings.HasSuffix(indexPath, "/"+IndexPage) {
				// needed because http.FileServer causes redirection in this case
				return resource, indexCode
			} else {
				return indexPath, indexCode
			}
		} else if a.DisableDirListing {
			delete(wHeader, "Expires")
			delete(wHeader, "Cache-Control")
			return indexPath, indexCode
		}
		resource = removeTrailingSlash(resource)
	}

	if a.MaxAge > 0 {
		wHeader.Set("Expires", a.expires())
		wHeader.Set("Cache-Control", fmt.Sprintf("public, max-age=%d", a.maxAgeS))
	}

	acceptEncoding := commaSeparatedList(req.Header.Get("Accept-Encoding"))

	if acceptEncoding.Contains("br") {
		brotli := resource + ".br"

		fdbr := a.checkResource(brotli, wHeader)

		if fdbr.code == OK {
			ext := filepath.Ext(resource)
			wHeader.Set("Content-Type", mime.TypeByExtension(ext))
			// the standard library sometimes overrides the content type via sniffing
			wHeader.Set("X-Content-Type-Options", "nosniff")
			wHeader.Set("Content-Encoding", "br")
			wHeader.Add("Vary", "Accept-Encoding")
			// weak etag because the representation is not the original file but a compressed variant
			wHeader.Set("ETag", "W/"+calculateEtag(fdbr.fi))
			return brotli, OK
		}
	}

	if acceptEncoding.Contains("gzip") {
		gzipped := resource + ".gz"

		fdgz := a.checkResource(gzipped, wHeader)

		if fdgz.code == OK {
			ext := filepath.Ext(resource)
			wHeader.Set("Content-Type", mime.TypeByExtension(ext))
			// the standard library sometimes overrides the content type via sniffing
			wHeader.Set("X-Content-Type-Options", "nosniff")
			wHeader.Set("Content-Encoding", "gzip")
			wHeader.Add("Vary", "Accept-Encoding")
			// weak etag because the representation is not the original file but a compressed variant
			wHeader.Set("ETag", "W/"+calculateEtag(fdgz.fi))
			return gzipped, OK
		}
	}

	// no intervention; the file will be served normally by the standard api
	fd := a.checkResource(resource, wHeader)

	if fd.code == Directory {
		// add trailing slash because we stripped it above and it allows the
		// standard file handler to create a directory listing
		fd.resource += "/"
	} else if fd.code < 300 {
		// strong etag because the representation is the original file
		wHeader.Set("ETag", calculateEtag(fd.fi))
	}

	return fd.resource, fd.code
}

// ServeHTTP implements the http.Handler interface. Note that it (a) handles
// headers for compression, expiry etc, and then (b) calls the standard
// http.ServeHTTP handler for each request. This ensures that it follows
// all the standard logic paths implemented there, including conditional
// requests and content negotiation.
func (a *Assets) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodHead && req.Method != http.MethodGet {
		// use the provided not-found handler
		Debugf("Assets ServeHTTP (method not allowed) %s %s R:%s W:%s\n", req.Method, req.URL.Path,
			headerStringer(req.Header), headerStringer(w.Header()))
		if a.MethodNotAllowed != nil {
			a.MethodNotAllowed.ServeHTTP(w, req)
		} else {
			httpError(w, MethodNotAllowed, req.Method)
		}
		return
	}

	resource, code := a.chooseResource(w.Header(), req, path.Drop(req.URL.Path, a.UnwantedPrefixSegments))

	if code == NotFound && a.NotFound != nil {
		// use the provided not-found handler
		Debugf("Assets ServeHTTP (not found) %s %s R:%s W:%s\n", req.Method, req.URL.Path,
			headerStringer(req.Header), headerStringer(w.Header()))
		a.NotFound.ServeHTTP(w, req)
		return
	}

	if code >= 400 {
		Debugf("Assets ServeHTTP (error %d) %s %s R:%s W:%s\n", code, req.Method, req.URL.Path,
			headerStringer(req.Header), headerStringer(w.Header()))
		httpError(w, code, req.Method)
		return
	}

	original := req.URL.Path
	req.URL.Path = resource

	// Conditional requests and content negotiation are handled in the standard net/http API.
	// Note that req.URL remains unchanged, even if prefix stripping is turned on, because the resource is
	// the only value that matters.
	a.server.ServeHTTP(w, req)

	Debugf("Assets (ok %d) %s %s (was %s) R:%s W:%s\n", code, req.Method, req.URL.Path, original,
		headerStringer(req.Header), headerStringer(w.Header()))

	// leave the path as we found it, in case middleware depends on the original value
	req.URL.Path = original
}
