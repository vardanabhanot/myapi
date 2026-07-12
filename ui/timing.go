package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

// timingLabel is the response-time label; hovering it toggles the timing
// waterfall panel. The panel lives in the response Stack rather than a
// widget.PopUp: a PopUp adds a full-canvas overlay that steals hover from
// this label, which made the popup flicker open/closed.
type timingLabel struct {
	widget.Label
	onHover func(hovering bool)
}

func newTimingLabel(text binding.String, onHover func(bool)) *timingLabel {
	l := &timingLabel{onHover: onHover}
	l.ExtendBaseWidget(l)
	l.Bind(text)
	return l
}

var _ desktop.Hoverable = (*timingLabel)(nil)

func (l *timingLabel) MouseIn(*desktop.MouseEvent)    { l.onHover(true) }
func (l *timingLabel) MouseMoved(*desktop.MouseEvent) {}
func (l *timingLabel) MouseOut()                      { l.onHover(false) }

// timingColor grades a phase against its own thresholds: green up to
// good, amber up to ok, red beyond — same palette as the status pill.
func timingColor(d, good, ok time.Duration) color.Color {
	switch {
	case d <= good:
		return theme.Color(theme.ColorNameSuccess)
	case d <= ok:
		return theme.Color(theme.ColorNameWarning)
	default:
		return theme.Color(theme.ColorNameError)
	}
}

// timingPanel wraps the waterfall in an opaque rounded card.
func timingPanel(t core.Timings) fyne.CanvasObject {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
	bg.CornerRadius = 6
	bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(waterfall(t)))
}

// waterfall renders each phase as an offset bar scaled to the total time.
func waterfall(t core.Timings) fyne.CanvasObject {
	type phase struct {
		name     string
		start    time.Duration
		dur      time.Duration
		good, ok time.Duration // per-phase grading thresholds
	}

	// Wait = server think time between request sent and first byte
	// (TTFB minus setup). TTFB = cumulative start→first byte, so its bar
	// spans everything above it — that's the timeline, not double counting.
	const ms = time.Millisecond
	setup := t.DNS + t.Connect + t.TLS
	phases := []phase{
		{"DNS", 0, t.DNS, 150 * ms, 400 * ms},
		{"Connect", t.DNS, t.Connect, 100 * ms, 300 * ms},
		{"TLS", t.DNS + t.Connect, t.TLS, 200 * ms, 500 * ms},
		{"Wait", setup, t.TTFB - setup, 300 * ms, 1000 * ms},
		{"TTFB", 0, t.TTFB, 500 * ms, 1200 * ms},
		{"Download", t.TTFB, t.Download, 300 * ms, 1000 * ms},
	}

	const barArea float32 = 220
	scale := barArea / float32(t.Total)

	rows := []fyne.CanvasObject{}
	for _, p := range phases {
		if p.dur < 0 {
			p.dur = 0
		}

		name := widget.NewLabel(p.name)

		offset := canvas.NewRectangle(color.Transparent)
		offset.SetMinSize(fyne.NewSize(float32(p.start)*scale, 8))
		bar := canvas.NewRectangle(timingColor(p.dur, p.good, p.ok))
		bar.CornerRadius = 2
		w := float32(p.dur) * scale
		if p.dur > 0 && w < 2 {
			w = 2 // keep sub-pixel phases visible
		}
		bar.SetMinSize(fyne.NewSize(w, 8))

		dur := widget.NewLabel(p.dur.Round(time.Millisecond / 10).String())
		dur.Alignment = fyne.TextAlignTrailing

		rows = append(rows,
			name,
			container.NewCenter(container.NewHBox(offset, bar, hSpacer(barArea-float32(p.start)*scale-w))),
			dur,
		)
	}

	total := widget.NewLabel("Total")
	total.TextStyle.Bold = true
	totalDur := widget.NewLabel(t.Total.Round(time.Millisecond / 10).String())
	totalDur.TextStyle.Bold = true
	totalDur.Alignment = fyne.TextAlignTrailing
	rows = append(rows, total, widget.NewLabel(""), totalDur)

	return container.New(&threeCol{}, rows...)
}

// hSpacer returns an invisible rect of the given width so every HBox bar
// row adds up to the same width and the bars align as a waterfall.
func hSpacer(w float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	if w < 0 {
		w = 0
	}
	r.SetMinSize(fyne.NewSize(w, 8))
	return r
}

// topRight pins its child at its natural size to the top-right corner and
// reports a zero MinSize, so showing the waterfall panel never grows the
// response container (which moved the split divider — hover jitter).
type topRight struct{}

func (topRight) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (topRight) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		m := o.MinSize()
		o.Resize(m)
		o.Move(fyne.NewPos(size.Width-m.Width, 0))
	}
}

// threeCol lays out rows of [name, bar, duration] with natural column widths.
type threeCol struct{}

func (l *threeCol) colWidths(objects []fyne.CanvasObject) [3]float32 {
	var w [3]float32
	for i, o := range objects {
		if m := o.MinSize().Width; m > w[i%3] {
			w[i%3] = m
		}
	}
	return w
}

func (l *threeCol) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w := l.colWidths(objects)
	var rowH float32
	for _, o := range objects {
		if h := o.MinSize().Height; h > rowH {
			rowH = h
		}
	}
	rowCount := float32((len(objects) + 2) / 3)
	return fyne.NewSize(w[0]+w[1]+w[2]+2*theme.Padding(), rowCount*rowH+(rowCount-1)*theme.Padding())
}

func (l *threeCol) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	w := l.colWidths(objects)
	var rowH float32
	for _, o := range objects {
		if h := o.MinSize().Height; h > rowH {
			rowH = h
		}
	}

	y := float32(0)
	for i := 0; i < len(objects); i += 3 {
		x := float32(0)
		for c := 0; c < 3 && i+c < len(objects); c++ {
			o := objects[i+c]
			o.Resize(fyne.NewSize(w[c], rowH))
			o.Move(fyne.NewPos(x, y))
			x += w[c] + theme.Padding()
		}
		y += rowH + theme.Padding()
	}
}
