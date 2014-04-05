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
