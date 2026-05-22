package styles

import (
	"image/color"

	"github.com/charmbracelet/x/exp/charmtone"
)

var (
	f4rgedOrange      = color.RGBA{R: 249, G: 115, B: 22, A: 255}
	f4rgedOrangeSoft  = color.RGBA{R: 251, G: 146, B: 60, A: 255}
	f4rgedOrangeMuted = color.RGBA{R: 154, G: 52, B: 18, A: 255}
	f4rgedBlack       = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	f4rgedBlackSoft   = color.RGBA{R: 8, G: 8, B: 8, A: 255}
	f4rgedBlackPanel  = color.RGBA{R: 14, G: 14, B: 14, A: 255}
	f4rgedWhite       = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	f4rgedWhiteSoft   = color.RGBA{R: 250, G: 250, B: 250, A: 255}
	f4rgedWhitePanel  = color.RGBA{R: 244, G: 244, B: 245, A: 255}
	f4rgedInk         = color.RGBA{R: 24, G: 24, B: 27, A: 255}
	f4rgedInkSoft     = color.RGBA{R: 63, G: 63, B: 70, A: 255}
	f4rgedInkMuted    = color.RGBA{R: 113, G: 113, B: 122, A: 255}
)

// ThemeForProvider returns the Styles associated with the given provider
// ID. Unknown or empty provider IDs yield the default 4RGED dark
// theme.
func ThemeForProvider(providerID string) Styles {
	switch providerID {
	case "hyper":
		return F4RGEDHyperDark()
	default:
		return F4RGEDDark()
	}
}

// ThemeForMode returns the 4RGED theme for a user-selected light/dark mode.
func ThemeForMode(mode string) Styles {
	if mode == "light" {
		return F4RGEDLight()
	}
	return F4RGEDDark()
}

// F4RGEDDark returns the 4RGED dark theme. It keeps the original Charmtone
// semantic palette and swaps only the primary brand accents to F4RGE orange.
func F4RGEDDark() Styles {
	return quickStyle(quickStyleOpts{
		primary:   f4rgedOrange,
		secondary: f4rgedOrangeSoft,
		accent:    charmtone.Bok,
		keyword:   charmtone.Blush,

		fgBase:       charmtone.Ash,
		fgMoreSubtle: charmtone.Squid,
		fgSubtle:     charmtone.Smoke,
		fgMostSubtle: charmtone.Oyster,

		onPrimary: charmtone.Butter,

		bgBase:         f4rgedBlack,
		bgLeastVisible: f4rgedBlackSoft,
		bgLessVisible:  f4rgedBlackSoft,
		bgMostVisible:  f4rgedBlackPanel,

		separator: f4rgedBlackPanel,

		destructive:       charmtone.Coral,
		error:             charmtone.Sriracha,
		warningSubtle:     charmtone.Zest,
		warning:           charmtone.Mustard,
		busy:              charmtone.Citron,
		info:              charmtone.Malibu,
		infoMoreSubtle:    charmtone.Sardine,
		infoMostSubtle:    charmtone.Damson,
		success:           charmtone.Julep,
		successMoreSubtle: charmtone.Bok,
		successMostSubtle: charmtone.Guac,
	})
}

// F4RGEDLight returns the 4RGED light theme. It mirrors the dark theme's
// orange brand accents while using white surfaces and dark text.
func F4RGEDLight() Styles {
	return quickStyle(quickStyleOpts{
		primary:   f4rgedOrange,
		secondary: f4rgedOrangeSoft,
		accent:    f4rgedOrangeMuted,
		keyword:   f4rgedOrangeMuted,

		fgBase:       f4rgedInk,
		fgMoreSubtle: f4rgedInkSoft,
		fgSubtle:     f4rgedInkMuted,
		fgMostSubtle: f4rgedInkMuted,

		onPrimary: f4rgedWhite,

		bgBase:         f4rgedWhite,
		bgLeastVisible: f4rgedWhiteSoft,
		bgLessVisible:  f4rgedWhiteSoft,
		bgMostVisible:  f4rgedWhitePanel,

		separator: f4rgedWhitePanel,

		destructive:       charmtone.Coral,
		error:             charmtone.Sriracha,
		warningSubtle:     f4rgedOrangeSoft,
		warning:           f4rgedOrange,
		busy:              charmtone.Zest,
		info:              f4rgedOrange,
		infoMoreSubtle:    f4rgedOrangeMuted,
		infoMostSubtle:    f4rgedWhitePanel,
		success:           charmtone.Julep,
		successMoreSubtle: charmtone.Bok,
		successMostSubtle: charmtone.Guac,
	})
}

// F4RGEDHyperDark returns the Hyper-flavored 4RGED dark theme.
func F4RGEDHyperDark() Styles {
	return quickStyle(quickStyleOpts{
		primary:   f4rgedOrange,
		secondary: f4rgedOrangeSoft,
		accent:    charmtone.Bok,

		fgBase:       charmtone.Ash,
		fgMoreSubtle: charmtone.Squid,
		fgSubtle:     charmtone.Smoke,
		fgMostSubtle: charmtone.Oyster,

		onPrimary: charmtone.Butter,

		bgBase:         f4rgedBlack,
		bgLeastVisible: f4rgedBlackSoft,
		bgLessVisible:  f4rgedBlackSoft,
		bgMostVisible:  f4rgedBlackPanel,

		separator: f4rgedBlackPanel,

		destructive:       charmtone.Coral,
		error:             charmtone.Sriracha,
		warningSubtle:     charmtone.Zest,
		warning:           charmtone.Mustard,
		busy:              charmtone.Citron,
		info:              charmtone.Malibu,
		infoMoreSubtle:    charmtone.Sardine,
		infoMostSubtle:    charmtone.Damson,
		success:           charmtone.Julep,
		successMoreSubtle: charmtone.Bok,
		successMostSubtle: charmtone.Guac,
	})
}
