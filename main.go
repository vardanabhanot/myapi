package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/vardanabhanot/myapi/ui"
)

var version string

func main() {
	// MYAPI_PPROF=1 exposes Go heap/cpu profiles on localhost for
	// memory debugging: go tool pprof http://localhost:6060/debug/pprof/heap
	if os.Getenv("MYAPI_PPROF") != "" {
		go http.ListenAndServe("localhost:6060", nil)
	}

	a := app.New()
	window := a.NewWindow("MyAPI")

	version = a.Metadata().Version
	a.Settings().SetTheme(&ui.BaseTheme{})
	window.Resize(fyne.NewSize(1024, 600))
	window.CenterOnScreen()
	window.SetContent(ui.MakeGUI(&window, version))
	window.ShowAndRun()
}
