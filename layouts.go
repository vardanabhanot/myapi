package main

import (
	"fyne.io/fyne/v2"
)

type baseLayout struct {
	left, center, right fyne.CanvasObject
}

func newBaseLayout(left, center, right fyne.CanvasObject) fyne.Layout {
	return &baseLayout{left: left, center: center, right: right}
}

func (l *baseLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Calculate the left section width as 20% of the total width
	leftWidth := size.Width * 0.20
	l.left.Move(fyne.NewPos(0, 0))
	l.left.Resize(fyne.NewSize(leftWidth, size.Height))

	// Calculate the remaining width (80% of the total width)
	remainingWidth := size.Width - leftWidth

	// The remaining width is split equally for center and right sections
	centerWidth := remainingWidth * 0.55
	l.center.Move(fyne.NewPos(leftWidth, 0))
	l.center.Resize(fyne.NewSize(centerWidth, size.Height))

	rightWidth := remainingWidth * 0.45
	l.right.Move(fyne.NewPos(leftWidth+centerWidth, 0))
	l.right.Resize(fyne.NewSize(rightWidth, size.Height))
}

func (l *baseLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	border := fyne.NewSize(220*2, l.left.MinSize().Height)

	return border.AddWidthHeight(100, 100)
}
