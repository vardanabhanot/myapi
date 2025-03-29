package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/vardanabhanot/myapi/ui"
)

const VERSION = "0.0.1"

var window fyne.Window

func main() {
	a := app.New()
	window = a.NewWindow("MyAPI")

	window.Resize(fyne.NewSize(1024, 600))
	window.CenterOnScreen()
	window.SetContent(ui.MakeGUI(&window))
	window.ShowAndRun()
}
