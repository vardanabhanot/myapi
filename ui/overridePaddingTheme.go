package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type overridePaddingTheme struct {
	padding float32
}

var _ fyne.Theme = (*overridePaddingTheme)(nil)

func (m *overridePaddingTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (m *overridePaddingTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *overridePaddingTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *overridePaddingTheme) Size(name fyne.ThemeSizeName) float32 {
	if m.padding != 0 && name == theme.SizeNamePadding {
		return m.padding
	}

	if name == theme.SizeNameText {
		return 11
	}

	return theme.DefaultTheme().Size(name)
}
