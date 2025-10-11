package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tappableIcon is a simple icon that can be tapped but doesn't implement Hoverable
// This prevents it from interfering with parent hover detection
type tappableIcon struct {
	widget.BaseWidget
	icon       *canvas.Image
	background *canvas.Circle
	onTapped   func()
	hovered    bool
	pressed    bool
}

func newTappableIcon(resource fyne.Resource, tapped func()) *tappableIcon {
	t := &tappableIcon{
		onTapped: tapped,
	}
	t.icon = canvas.NewImageFromResource(resource)
	t.icon.FillMode = canvas.ImageFillContain

	t.background = canvas.NewCircle(color.Transparent)

	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableIcon) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(t.background, t.icon)
	return widget.NewSimpleRenderer(c)
}

func (t *tappableIcon) Tapped(*fyne.PointEvent) {
	t.pressed = false
	t.updateBackground()
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *tappableIcon) TappedSecondary(*fyne.PointEvent) {}

func (t *tappableIcon) MinSize() fyne.Size {
	return fyne.NewSize(theme.IconInlineSize(), theme.IconInlineSize())
}

// Cursor changes to pointer on hover to indicate it's clickable
func (t *tappableIcon) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

// MouseIn provides visual feedback when hovering
func (t *tappableIcon) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.updateBackground()
}

// MouseOut removes visual feedback
func (t *tappableIcon) MouseOut() {
	t.hovered = false
	t.pressed = false
	t.updateBackground()
}

// MouseDown provides pressed visual feedback
func (t *tappableIcon) MouseDown(*desktop.MouseEvent) {
	t.pressed = true
	t.updateBackground()
}

// MouseUp removes pressed visual feedback
func (t *tappableIcon) MouseUp(*desktop.MouseEvent) {
	t.pressed = false
	t.updateBackground()
}

func (t *tappableIcon) updateBackground() {
	if t.pressed {
		t.background.FillColor = theme.Color(theme.ColorNamePressed)
	} else if t.hovered {
		t.background.FillColor = theme.Color(theme.ColorNameHover)
	} else {
		t.background.FillColor = color.Transparent
	}
	t.Refresh()
}
