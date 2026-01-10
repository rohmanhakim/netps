package common

const ColorWhite = "255"
const ColorDarkGray = "248"
const ColorBoulder = "243"
const ColorDavisGrey = "240"
const ColorDarkOlive = "237"
const ColorDarkCharcoal = "236"
const ColorGoldenRod = "178"
const ColorBrightRed = "124"
const ColorBlueJeans = "75"
const ColorElectricIndigo = "57"
const ColorArgent = "7"
const ColorOfficeGreen = "2"

type Theme struct {
	ColorForegroundBase      string
	ColorForegroundSecondary string
	ColorBackgroundSecondary string
	ColorForegroundSubtle    string
	ColorInactive            string
	ColorAccent              string
	ColorHighlight           string
	ColorHighlightSubtle     string
	ColorSuccess             string
	ColorNeutral             string
	ColorDanger              string
	ColorWarning             string
	SpacingSmall             int
	SpacingMedium            int
}
