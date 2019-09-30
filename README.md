# servefiles

[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg?style=flat-square)](https://godoc.org/github.com/rickb777/servefiles)
[![Build Status](https://travis-ci.org/rickb777/servefiles.svg?branch=master)](https://travis-ci.org/rickb777/servefiles)
[![Coverage Status](https://coveralls.io/repos/rickb777/servefiles/badge.svg?branch=master&service=github)](https://coveralls.io/github/rickb777/servefiles?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/rickb777/servefiles)](https://goreportcard.com/report/github.com/rickb777/servefiles)
[![Issues](https://img.shields.io/github/issues/rickb777/servefiles.svg)](https://github.com/rickb777/servefiles/issues)

Serve static files from a Go http server, including performance-enhancing features.

 * Based on the standard net/http ServeFiles, with gzip/brotli and cache performance enhancements.

Please see the [GoDoc](https://godoc.org/github.com/rickb777/servefiles) for more.

## Installation

    go get -u github.com/rickb777/servefiles/v3

## v3

Version 3 brings Go module support. Also, `brotli` encoding is supported alongside `gzip` encoding. Brotli now has widespread implementation in most browsers. You can compress your textual assets (including Javascript, CSS, HTML, SVG etc) using Brotli and/or Gzip as part of your build pipeline, uploading both the original and compressed files to your production server's asset directories. Brotli compression takes longer than Gzip but produces more compact files. Compression is, of course, optional.
 
## Status

This library has been in reliable production use for some time. Versioning follows the well-known semantic version pattern.

## Licence

[MIT](LICENSE)
