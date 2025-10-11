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
	paddedContainer, _ := h.content.(*fyne.Container)
	borderContainer, _ := paddedContainer.Objects[0].(*fyne.Container)
	optionsStack := borderContainer.Objects[2].(*fyne.Container)
	optionsStack.Objects[0].(*fyne.Container).Hide() // The container of the elpased time
	optionsStack.Objects[1].(*tappableIcon).Show()
	h.Refresh()
}

func (h *hoverableListItem) MouseOut() {
	h.hovered = false
	paddedContainer, _ := h.content.(*fyne.Container)
	borderContainer, _ := paddedContainer.Objects[0].(*fyne.Container)
	optionsStack := borderContainer.Objects[2].(*fyne.Container)
	optionsStack.Objects[0].(*fyne.Container).Show() // The container of the elpased time
	optionsStack.Objects[1].(*tappableIcon).Hide()
	h.Refresh()
}

func (h *hoverableListItem) MouseMoved(*desktop.MouseEvent) {
	// not used
}
