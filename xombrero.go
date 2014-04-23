// Copyright (c) 2014 Josh Rickmar.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"runtime"

	"github.com/conformal/gotk3/gtk"
	"github.com/jrick/go-webkit2/wk2"
)

const HomePage HTMLPageDescription = "https://www.duckduckgo.com/lite"

const (
	defaultWinWidth  = 1024
	defaultWinHeight = 768
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
	window.SetDefaultGeometry(defaultWinWidth, defaultWinHeight)
	window.Show()

	wc := wk2.DefaultWebContext()
	wc.SetProcessModel(wk2.ProcessModelMultipleSecondaryProcesses)

	session := []PageDescription{HomePage}
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
