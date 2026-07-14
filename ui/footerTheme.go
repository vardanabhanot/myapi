package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// footerTheme is BaseTheme with tighter footer sizing.
type footerTheme struct {
	BaseTheme
}

var _ fyne.Theme = (*footerTheme)(nil)

func (m *footerTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameText:
		return 11
	case theme.SizeNameInnerPadding:
		return 3
	default:
		return m.BaseTheme.Size(name)
	}
}
