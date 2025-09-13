package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NOTE:: As this was taken from fyne split, and the split comments are still here will remove or change them when I am less lazy

// Declare conformity with CanvasObject interface
var _ fyne.CanvasObject = (*ResponseContainer)(nil)

// Split defines a container whose size is split between two children.
//
// Since: 1.4
type ResponseContainer struct {
	widget.BaseWidget
	Offset     float64
	Content    fyne.CanvasObject
	background fyne.CanvasObject

	// to communicate to the renderer that the next refresh
	// is just an offset update (ie a resize and move only)
	// cleared by renderer in Refresh()
	offsetUpdated bool
}

func NewResponseContainer(content fyne.CanvasObject) *ResponseContainer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))

	s := &ResponseContainer{
		Offset:     0.70, // Sensible default, can be overridden with SetOffset
		Content:    content,
		background: background,
	}
	s.BaseWidget.ExtendBaseWidget(s)
	return s
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (s *ResponseContainer) CreateRenderer() fyne.WidgetRenderer {
	s.BaseWidget.ExtendBaseWidget(s)
	d := newResponseDivider(s)

	return &responseContainerRenderer{
		container: s,
		divider:   d,
		objects:   []fyne.CanvasObject{s.background, d, s.Content},
	}
}

// ExtendBaseWidget is used by an extending widget to make use of BaseWidget functionality.
//
// Deprecated: Support for extending containers is being removed
func (s *ResponseContainer) ExtendBaseWidget(wid fyne.Widget) {
	s.BaseWidget.ExtendBaseWidget(wid)
}

// SetOffset sets the offset (0.0 to 1.0) of the Split divider.
// 0.0 - Leading is min size, Trailing uses all remaining space.
// 0.5 - Leading & Trailing equally share the available space.
// 1.0 - Trailing is min size, Leading uses all remaining space.
func (s *ResponseContainer) SetOffset(offset float64) {
	if s.Offset == offset {
		return
	}
	s.Offset = offset
	s.offsetUpdated = true
	s.Refresh()
}

var _ fyne.WidgetRenderer = (*responseContainerRenderer)(nil)

type responseContainerRenderer struct {
	container *ResponseContainer
	divider   *responseDivider
	objects   []fyne.CanvasObject
}

func (r *responseContainerRenderer) Destroy() {
}

func (r *responseContainerRenderer) Layout(size fyne.Size) {
	var dividerPos, contentPos fyne.Position
	var dividerSize, contentSize fyne.Size

	ch := r.computeContainerLengths(size.Height, r.minContentHeight())
	dividerPos.Y = ch + 1.5
	dividerSize.Width = size.Width
	dividerSize.Height = responsedividerThickness(r.divider)
	contentPos.Y = dividerPos.Y + 1.5
	contentSize.Width = size.Width
	contentSize.Height = size.Height - ch

	r.divider.Move(dividerPos)
	r.divider.Resize(dividerSize)
	r.container.Content.Move(contentPos)
	r.container.Content.Resize(contentSize)

	// The container is transparent by default so we made it have a rectangle
	// thats why we need the pos and size same as the content
	r.container.background.Move(contentPos)
	r.container.background.Resize(contentSize)
	canvas.Refresh(r.divider)
}

func (r *responseContainerRenderer) MinSize() fyne.Size {
	s := fyne.NewSize(0, 0)
	for _, o := range r.objects {
		min := o.MinSize()
		s.Width = fyne.Max(s.Width, min.Width)
		s.Height += min.Height
	}

	return s
}

func (r *responseContainerRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *responseContainerRenderer) Refresh() {
	if r.container.offsetUpdated {
		r.Layout(r.container.Size())
		r.container.offsetUpdated = false
		return
	}

	// [1] is divider which doesn't change
	r.objects[0] = r.container.background
	r.objects[2] = r.container.Content
	r.Layout(r.container.Size())

	r.divider.Refresh()
	r.container.Content.Refresh()
	canvas.Refresh(r.container)
}

func (r *responseContainerRenderer) computeContainerLengths(total, cMin float32) float32 {
	available := float64(total - 1.5 - cMin)
	if available <= 0 {
		return 0
	}
	offset := r.container.Offset

	min := 0.1
	max := float64(1)

	if offset < min {
		offset = min
	}
	if offset > max {
		offset = max
	}

	return float32(available * offset)
}

func (r *responseContainerRenderer) minContentHeight() float32 {
	if r.container.Content.Visible() {
		return r.container.Content.MinSize().Height
	}

	return 0
}

// Declare conformity with interfaces
var _ fyne.CanvasObject = (*responseDivider)(nil)
var _ fyne.Draggable = (*responseDivider)(nil)
var _ desktop.Cursorable = (*responseDivider)(nil)
var _ desktop.Hoverable = (*responseDivider)(nil)

type responseDivider struct {
	widget.BaseWidget
	split          *ResponseContainer
	hovered        bool
	startDragOff   *fyne.Position
	currentDragPos fyne.Position
}

func newResponseDivider(split *ResponseContainer) *responseDivider {
	d := &responseDivider{
		split: split,
	}
	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (d *responseDivider) CreateRenderer() fyne.WidgetRenderer {
	d.ExtendBaseWidget(d)
	th := d.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	background := canvas.NewRectangle(th.Color(theme.ColorNameShadow, v))
	return &responsedividerRenderer{
		divider:    d,
		background: background,
		objects:    []fyne.CanvasObject{background},
	}
}

func (d *responseDivider) Cursor() desktop.Cursor {
	return desktop.VResizeCursor
}

func (d *responseDivider) DragEnd() {
	d.startDragOff = nil
}

func (d *responseDivider) Dragged(e *fyne.DragEvent) {
	if d.startDragOff == nil {
		d.currentDragPos = d.Position().Add(e.Position)
		start := e.Position.Subtract(e.Dragged)
		d.startDragOff = &start
	} else {
		d.currentDragPos = d.currentDragPos.Add(e.Dragged)
	}

	_, y := d.currentDragPos.Components()
	var offset float64
	heightFree := float64(d.split.Size().Height - responsedividerThickness(d))
	offset = float64(y-d.startDragOff.Y) / heightFree

	d.split.SetOffset(offset)
}

func (d *responseDivider) MouseIn(event *desktop.MouseEvent) {
	d.hovered = true
	d.split.Refresh()
}

func (d *responseDivider) MouseMoved(event *desktop.MouseEvent) {
	d.hovered = true
}

func (d *responseDivider) MouseOut() {
	d.hovered = false
	d.split.Refresh()
}

var _ fyne.WidgetRenderer = (*responsedividerRenderer)(nil)

type responsedividerRenderer struct {
	divider    *responseDivider
	background *canvas.Rectangle

	objects []fyne.CanvasObject
}

func (r *responsedividerRenderer) Destroy() {
}

func (r *responsedividerRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
}

func (r *responsedividerRenderer) MinSize() fyne.Size {
	return fyne.NewSize(responsedividerLength(r.divider), responsedividerThickness(r.divider))
}

func (r *responsedividerRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *responsedividerRenderer) Refresh() {
	th := r.divider.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if r.divider.hovered {
		r.background.FillColor = th.Color(theme.ColorNamePrimary, v)
	} else {
		r.background.FillColor = th.Color(theme.ColorNameSeparator, v)
	}
	r.background.Refresh()
	r.Layout(r.divider.Size())
}

func responsedividerTheme(d *responseDivider) fyne.Theme {
	if d == nil {
		return theme.Current()
	}

	return d.Theme()
}

func responsedividerThickness(d *responseDivider) float32 {
	if d.hovered {
		return 4
	}

	return 1
}

func responsedividerLength(d *responseDivider) float32 {
	th := responsedividerTheme(d)
	return th.Size(theme.SizeNamePadding) * 6
}
