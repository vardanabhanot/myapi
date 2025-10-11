package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/vardanabhanot/myapi/ui"
)

var version string

func main() {
	a := app.New()
	window := a.NewWindow("MyAPI")

	version = a.Metadata().Version
	a.Settings().SetTheme(&ui.BaseTheme{})
	window.Resize(fyne.NewSize(1024, 600))
	window.CenterOnScreen()
	window.SetContent(ui.MakeGUI(&window, version))
	window.ShowAndRun()
}
