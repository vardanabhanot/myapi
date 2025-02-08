package main

import (
	"fyne.io/fyne/v2"
)

type baseLayout struct {
	left, center fyne.CanvasObject
}

func newBaseLayout(left, center fyne.CanvasObject) fyne.Layout {
	return &baseLayout{left: left, center: center}
}

func (l *baseLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Calculate the left section width as 20% of the total width
	leftWidth := size.Width * 0.20
	l.left.Move(fyne.NewPos(0, 0))
	l.left.Resize(fyne.NewSize(leftWidth, size.Height))

	// Calculate the remaining width (80% of the total width)
	remainingWidth := size.Width - leftWidth

	// The remaining width is split equally for center and right sections
	l.center.Move(fyne.NewPos(leftWidth, 0))
	l.center.Resize(fyne.NewSize(remainingWidth, size.Height))
}

func (l *baseLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	border := fyne.NewSize(220*2, l.left.MinSize().Height)

	return border.AddWidthHeight(100, 100)
}
