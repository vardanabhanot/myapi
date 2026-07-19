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
	// Neutral greys + a single restrained blue accent, in the spirit of
	// native dev tools (VS Code / terminal), not tinted dashboard palettes.
	if variant == theme.VariantDark {
		switch name {
		case theme.ColorNamePrimary:
			return color.NRGBA{R: 0, G: 122, B: 204, A: 255} // dev-tool blue
		case theme.ColorNameBackground:
			return color.NRGBA{R: 24, G: 24, B: 24, A: 255}
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 37, G: 37, B: 38, A: 255}
		case theme.ColorNameButton:
			return color.NRGBA{R: 45, G: 45, B: 46, A: 255}
		case theme.ColorNameDisabledButton:
			return color.NRGBA{R: 32, G: 32, B: 33, A: 255}
		case theme.ColorNameHover:
			return color.NRGBA{R: 55, G: 55, B: 56, A: 200}
		case theme.ColorNamePressed:
			return color.NRGBA{R: 0, G: 122, B: 204, A: 50}
		case theme.ColorNameForeground:
			return color.NRGBA{R: 212, G: 212, B: 212, A: 255}
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 110, G: 110, B: 110, A: 255}
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 138, G: 138, B: 138, A: 255}
		case theme.ColorNameSeparator:
			return color.NRGBA{R: 51, G: 51, B: 51, A: 255}
		case theme.ColorNameInputBorder:
			return color.NRGBA{R: 60, G: 60, B: 60, A: 255}
		case theme.ColorNameMenuBackground:
			return color.NRGBA{R: 31, G: 31, B: 32, A: 255}
		case theme.ColorNameOverlayBackground:
			return color.NRGBA{R: 37, G: 37, B: 38, A: 255}
		case theme.ColorNameHeaderBackground:
			return color.NRGBA{R: 31, G: 31, B: 32, A: 255}
		case theme.ColorNameScrollBar:
			return color.NRGBA{R: 110, G: 110, B: 110, A: 180}
		case theme.ColorNameSelection:
			return color.NRGBA{R: 0, G: 122, B: 204, A: 90}
		case theme.ColorNameFocus:
			return color.NRGBA{R: 0, G: 122, B: 204, A: 255}
		case theme.ColorNameHyperlink:
			return color.NRGBA{R: 77, G: 166, B: 255, A: 255}
		case theme.ColorNameSuccess:
			return color.NRGBA{R: 46, G: 160, B: 67, A: 255}
		case theme.ColorNameError:
			return color.NRGBA{R: 229, G: 83, B: 75, A: 255}
		case theme.ColorNameWarning:
			return color.NRGBA{R: 210, G: 153, B: 34, A: 255}
		case theme.ColorNameShadow:
			return color.NRGBA{R: 0, G: 0, B: 0, A: 120}
		}
	} else {
		switch name {
		case theme.ColorNamePrimary:
			return color.NRGBA{R: 0, G: 102, B: 181, A: 255} // dev-tool blue
		case theme.ColorNameBackground:
			return color.NRGBA{R: 248, G: 248, B: 248, A: 255}
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		case theme.ColorNameHover:
			return color.NRGBA{R: 232, G: 232, B: 232, A: 255}
		case theme.ColorNameSeparator:
			return color.NRGBA{R: 218, G: 218, B: 218, A: 255}
		case theme.ColorNameInputBorder:
			return color.NRGBA{R: 206, G: 206, B: 206, A: 255}
		case theme.ColorNameDisabled:
			// Doubles as the muted-label color (sectionHeader, history
			// timestamps); Fyne's pale default washes out on light bg.
			return color.NRGBA{R: 105, G: 105, B: 105, A: 255}
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 140, G: 140, B: 140, A: 255}
		case theme.ColorNameMenuBackground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		case theme.ColorNameOverlayBackground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		case theme.ColorNameHeaderBackground:
			return color.NRGBA{R: 243, G: 243, B: 243, A: 255}
		case theme.ColorNameScrollBar:
			return color.NRGBA{R: 140, G: 140, B: 140, A: 200}
		case theme.ColorNameSelection:
			return color.NRGBA{R: 0, G: 102, B: 181, A: 60}
		case theme.ColorNameFocus:
			return color.NRGBA{R: 0, G: 102, B: 181, A: 255}
		case theme.ColorNameHyperlink:
			return color.NRGBA{R: 0, G: 102, B: 181, A: 255}
		case theme.ColorNameSuccess:
			return color.NRGBA{R: 26, G: 127, B: 55, A: 255}
		case theme.ColorNameError:
			return color.NRGBA{R: 207, G: 34, B: 46, A: 255}
		case theme.ColorNameWarning:
			return color.NRGBA{R: 154, G: 103, B: 0, A: 255}
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}

// Chevron replacements for Fyne's stemmed MoveDown/MoveUp arrows (material
// expand_more / expand_less). Overridden globally — Tree's branchIcon
// re-resolves via theme.Current() on every Refresh (hover/select), so a
// scoped ThemeOverride gets reverted. NavigateNext is already a ">" chevron.
var chevronDownIcon = theme.NewThemedResource(fyne.NewStaticResource("chevron-down.svg",
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M16.59 8.59 12 13.17 7.41 8.59 6 10l6 6 6-6z"/></svg>`)))

var chevronUpIcon = theme.NewThemedResource(fyne.NewStaticResource("chevron-up.svg",
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 8l-6 6 1.41 1.41L12 10.83l4.59 4.58L18 14z"/></svg>`)))

var chevronRightIcon = theme.NewThemedResource(fyne.NewStaticResource("chevron-right.svg",
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M10 6 8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z"/></svg>`)))

func (m BaseTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	switch name {
	case theme.IconNameMoveDown:
		return chevronDownIcon
	case theme.IconNameMoveUp:
		return chevronUpIcon
	case theme.IconNameNavigateNext:
		return chevronRightIcon
	}
	return theme.DefaultTheme().Icon(name)
}

func (m BaseTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m BaseTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 12
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNamePadding:
		return 6
	}
	return theme.DefaultTheme().Size(name)
}
