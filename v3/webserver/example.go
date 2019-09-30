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

// Package main provides a webserver. The purpose is mostly to show by example how to serve assets.
// It supports both HTTP and HTTPS.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rickb777/servefiles/v3"
)

var path = flag.String("path", "..", "directory for the files tp be served")
var cert = flag.String("cert", "", "file containing the certificate (optional)")
var key = flag.String("key", "", "file containing the private key (optional)")
var port = flag.Int("port", 8080, "TCP port to listen on")
var maxAge = flag.String("maxage", "", "Maximum age of assets sent in response headers - causes client caching")
var verbose = flag.Bool("v", false, "Enable verbose messages")

func main() {
	flag.Parse()

	if *verbose {
		servefiles.Debugf = log.Printf
	}

	if (*cert != "" && *key == "") ||
		(*cert == "" && *key != "") {
		log.Fatal("Both certificate file (-cert) and private key file (-key) are required.")
	}

	h := servefiles.NewAssetHandler(*path)

	if *maxAge != "" {
		d, err := time.ParseDuration(*maxAge)
		log.Printf("MaxAge: %s %v\n", d, err)
		h = h.WithMaxAge(d)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: h,
	}

	if *cert != "" {
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_AES_256_GCM_SHA384,
			},
		}
		log.Printf("Access the server via: https://localhost:%d/", *port)
		log.Fatal(srv.ListenAndServeTLS(*cert, *key))

	} else {
		log.Printf("Access the server via: http://localhost:%d/", *port)
		log.Fatal(srv.ListenAndServe())
	}
}
