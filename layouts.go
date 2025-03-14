package main

import (
	"fyne.io/fyne/v2"
)

type infoLayout struct {
	content fyne.CanvasObject
}

func newInfoLayout(content fyne.CanvasObject) fyne.Layout {
	return &infoLayout{content: content}
}

func (l *infoLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	l.content.Resize(fyne.NewSize(size.Width, 10))
}

func (l *infoLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	border := fyne.NewSize(l.content.MinSize().Width, l.content.MinSize().Height)
	return border.AddWidthHeight(20, 20)
}
