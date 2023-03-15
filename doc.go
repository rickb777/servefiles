// MIT License
//
// Copyright (c) 2016 Rick Beton
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

/*
Package servefiles provides a static asset handler for serving files such as images, stylesheets and
javascript code. This is an enhancement to the standard net/http ServeFiles, which is used internally.
Care is taken to set headers such that the assets will be efficiently cached by browsers and proxies.

	assets := servefiles.NewAssetHandler("./assets/").WithMaxAge(time.Hour)

Assets is an http.Handler and can be used alongside your other handlers.

# Gzipped Content

The Assets handler serves gzipped content when the browser indicates it can accept it. But it does not
gzip anything on-the-fly. Nor does it create any gzipped files for you.

During the preparation of your web assets, all text files (CSS, JS etc) should be accompanied by their gzipped
equivalent; your build process will need to do this. The Assets handler will first look for the gzipped file,
which it will serve if present. Otherwise it will serve the 'normal' file.

This has many benefits: fewer bytes are read from the disk, a smaller memory footprint is needed in the server,
less data copying happens, fewer bytes are sent across the network, etc.

You should not attempt to gzip already-compressed files, such as PNG, JPEG, SVGZ, etc.

Very small files (e.g. less than 1kb) gain little from compression because they may be small enough to fit
within a single TCP packet, so don't bother with them. (They might even grow in size when gzipped.)

# Conditional Request Support

The Assets handler sets 'Etag' headers for the responses of the assets it finds. Modern browsers need this: they
are then able to send conditional requests that very often shrink responses to a simple 304 Not Modified. This
improves the experience for users and leaves your server free to do more of other things.

The Etag value is calculated from the file size and modification timestamp, a commonly used approach. Strong
or weak tags are used for plain or gzipped files respectively (the reason is that a given file can be
compressed with different levels of compression, a weak Etag indicates there is not a strict match for the
file's content).

For further information see RFC9110 https://tools.ietf.org/html/rfc9110.

# Cache Control

To go even further, the 'far-future' technique can and should often be used. Set a long expiry time, e.g.
ten years via `time.Hour * 24 * 365 * 10`.
Browsers will cache such assets and not make requests for them for the next ten years (or whatever). Not even
conditional requests are made. There is clearly a big benefit in page load times after the first visit.

No in-memory caching is performed server-side. This is needed less due to far-future caching being
supported, but might be added in future.

For further information see RFC9111 https://tools.ietf.org/html/rfc9111.

# Path Stripping

The Assets handler can optionally strip some path segments from the URL before selecting the asset to be served.

This means, for example, that the URL

	http://example.com/e3b1cf/css/style1.css

can map to the asset files

	./assets/css/style1.css
	./assets/css/style1.css.gz

without the /e3b1cf/ segment. The benefit of this is that you can use a unique number or hash in that segment (chosen
for example each time your server starts). Each time that number changes, browsers will see the asset files as
being new, and they will later drop old versions from their cache regardless of their ten-year lifespan.

So you get the far-future lifespan combined with being able to push out changed assets as often as you need to.

# Example Usage

To serve files with a ten-year expiry, this creates a suitably-configured handler:

	assets := servefiles.NewAssetHandler("./assets/").StripOff(1).WithMaxAge(10 * 365 * 24 * time.Hour)

The first parameter names the local directory that holds the asset files. It can be absolute or relative to
the directory in which the server process is started.

Notice here the StripOff parameter is 1, so the first segment of the URL path gets discarded. A larger number
is permitted.

The WithMaxAge parameter is the maximum age to be specified in the cache-control headers. It can be any duration
from zero upwards.
*/
package servefiles
