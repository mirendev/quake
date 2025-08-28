package color

import (
	"os"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Faint     = "\033[2m"
	Underline = "\033[4m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Purple    = "\033[35m"
	Cyan      = "\033[36m"
	Gray      = "\033[37m"
	White     = "\033[97m"
)

var (
	// NoColor disables color output
	NoColor = false
)

func init() {
	// Check if we're in a terminal environment
	if os.Getenv("NO_COLOR") != "" {
		NoColor = true
	}
}

// colorize applies color codes if colors are enabled
func colorize(color, text string) string {
	if NoColor {
		return text
	}
	return color + text + Reset
}

// Bold returns bold text
func BoldText(text string) string {
	return colorize(Bold, text)
}

// Red returns red text
func RedText(text string) string {
	return colorize(Red, text)
}

// Green returns green text
func GreenText(text string) string {
	return colorize(Green, text)
}

// Yellow returns yellow text
func YellowText(text string) string {
	return colorize(Yellow, text)
}

// Blue returns blue text
func BlueText(text string) string {
	return colorize(Blue, text)
}

// Cyan returns cyan text
func CyanText(text string) string {
	return colorize(Cyan, text)
}

// Purple returns purple text
func PurpleText(text string) string {
	return colorize(Purple, text)
}

// UnderlineText returns underlined text
func UnderlineText(text string) string {
	return colorize(Underline, text)
}

// FaintText returns faint/dim text
func FaintText(text string) string {
	return colorize(Faint, text)
}