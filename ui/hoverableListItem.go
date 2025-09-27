package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

//var _ desktop.Hoverable = (*hoverableContainer)(nil)

type hoverableListItem struct {
	widget.BaseWidget // Embeds BaseWidget
	hovered           bool
	content           fyne.CanvasObject
}

// NewhoverableContainer creates a new instance of our custom widget
func newHoverableListItem(object fyne.CanvasObject) *hoverableListItem {
	item := &hoverableListItem{
		content: object,
	}
	item.ExtendBaseWidget(item) // Important: sets up the widget
	return item
}

func (h *hoverableListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.content)
}

func (h *hoverableListItem) MouseIn(e *desktop.MouseEvent) {
	h.hovered = true
	paddedC, _ := h.content.(*fyne.Container)
	grid, _ := paddedC.Objects[0].(*fyne.Container)
	grid.Objects[1].(*fyne.Container).Objects[1].(*widget.Button).Show()
	h.Refresh()
}

func (h *hoverableListItem) MouseOut() {
	h.hovered = false
	paddedC, _ := h.content.(*fyne.Container)
	grid, _ := paddedC.Objects[0].(*fyne.Container)
	grid.Objects[1].(*fyne.Container).Objects[1].(*widget.Button).Hide()
	h.Refresh()
}

func (h *hoverableListItem) MouseMoved(*desktop.MouseEvent) {
	// not using
}
