// Copyright (c) 2014 Josh Rickmar.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/conformal/gotk3/gtk"
	"runtime"
)

// RunGUI initializes GTK, creates the toplevel window and all child widgets,
// opens the pages for the default session, and runs the Glib main event loop.
// This function blocks until the toplevel window is destroyed and the event
// loop exits.
func RunGUI() {
	gtk.Init(nil)

	window, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	window.Connect("destroy", func() {
		gtk.MainQuit()
	})
	window.Show()

	home := &HTMLPageDescription{"https://www.duckduckgo.com/lite"}
	session := []PageDescription{home}

	pm := NewPageManager(session)
	window.Add(pm)
	pm.Show()

	gtk.Main()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	RunProfiler("localhost:7070")
	RunGUI()
}
