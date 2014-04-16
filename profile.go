// Copyright (c) 2014 Josh Rickmar.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

// RunProfiler runs a pprof HTTP server on the interface and port
// described by connect.
func RunProfiler(connect string) {
	go func() {
		profileRedirect := http.RedirectHandler("/debug/pprof",
			http.StatusSeeOther)
		http.Handle("/", profileRedirect)
		log.Println(http.ListenAndServe(connect, nil))
	}()
}
