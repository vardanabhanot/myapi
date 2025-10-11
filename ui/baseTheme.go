package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type BaseTheme struct{}

var _ fyne.Theme = (*BaseTheme)(nil)

// The color package has to be imported from "image/color".

func (m BaseTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (m BaseTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m BaseTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m BaseTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == "text" {
		return 11
	}

	return theme.DefaultTheme().Size(name)
}
