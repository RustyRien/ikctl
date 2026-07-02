package ui

import "github.com/derailed/tcell/v2"

var (
	colorHeader    = tcell.ColorCadetBlue
	colorInfo      = tcell.ColorCadetBlue
	colorSuccess   = tcell.ColorLightSkyBlue
	colorWarning   = tcell.ColorDarkOrange
	colorError     = tcell.ColorRed
	colorRunning   = tcell.ColorDodgerBlue
	colorTitle     = tcell.ColorWhite
	colorTableText = tcell.ColorLightSteelBlue
	colorBg        = tcell.ColorBlack
	colorLogo      = tcell.ColorOrange
	colorKey       = tcell.ColorDodgerBlue
	colorNumKey    = tcell.ColorFuchsia
	colorTitleInfo = tcell.ColorAqua
)

func applyColors(enabled bool) {
	if enabled {
		colorHeader = tcell.ColorCadetBlue
		colorInfo = tcell.ColorCadetBlue
		colorSuccess = tcell.ColorLightSkyBlue
		colorWarning = tcell.ColorDarkOrange
		colorError = tcell.ColorRed
		colorRunning = tcell.ColorDodgerBlue
		colorTitle = tcell.ColorWhite
		colorTableText = tcell.ColorLightSteelBlue
		colorBg = tcell.ColorBlack
		colorLogo = tcell.ColorOrange
		colorKey = tcell.ColorDodgerBlue
		colorNumKey = tcell.ColorFuchsia
		colorTitleInfo = tcell.ColorAqua
		return
	}
	colorHeader = tcell.ColorWhite
	colorInfo = tcell.ColorWhite
	colorSuccess = tcell.ColorWhite
	colorWarning = tcell.ColorWhite
	colorError = tcell.ColorWhite
	colorRunning = tcell.ColorWhite
	colorTitle = tcell.ColorWhite
	colorTableText = tcell.ColorWhite
	colorBg = tcell.ColorBlack
	colorLogo = tcell.ColorWhite
	colorKey = tcell.ColorWhite
	colorNumKey = tcell.ColorWhite
	colorTitleInfo = tcell.ColorWhite
}
