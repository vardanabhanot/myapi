package ui

import (
	"fyne.io/fyne/v2"
)

type footerLayout struct {
	top fyne.CanvasObject
}

func newFooterLayout(top fyne.CanvasObject) fyne.Layout {
	return &footerLayout{top}
}

func (b *footerLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {

	var topSize fyne.Size
	if b.top != nil && b.top.Visible() {
		topHeight := b.top.MinSize().Height
		b.top.Resize(fyne.NewSize(size.Width, topHeight))
		b.top.Move(fyne.NewPos(0, 0))
		topSize = fyne.NewSize(size.Width, topHeight)
	}

	middleSize := fyne.NewSize(size.Width, size.Height-topSize.Height)
	var rightSize float32

	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if child != b.top {
			rightSize += child.MinSize().Width
			middlePos := fyne.NewPos(size.Width-float32(rightSize)-10, topSize.Height)
			child.Resize(middleSize)
			child.Move(middlePos)
		}
	}
}

func (b *footerLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if child != b.top {
			minSize = minSize.Max(child.MinSize())
		}
	}

	if b.top != nil && b.top.Visible() {
		topMin := b.top.MinSize()
		minWidth := fyne.Max(minSize.Width, topMin.Width)
		minSize = fyne.NewSize(minWidth, minSize.Height+topMin.Height)
	}

	return minSize
}
