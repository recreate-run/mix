package vision

import "image/color"

const (
	// BoxThickness is the pixel width of bounding box lines
	BoxThickness = 3

	// LabelOffsetX is the horizontal offset for element labels
	LabelOffsetX = 4

	// LabelOffsetY is the vertical offset for element labels
	LabelOffsetY = 14
)

var (
	// BoxColor is the orange color for bounding boxes (RGB 255, 100, 0)
	BoxColor = color.RGBA{R: 255, G: 100, B: 0, A: 255}

	// TextColor is the white color for element labels
	TextColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

// interactiveRoles defines which accessibility roles are considered interactive
var interactiveRoles = map[string]bool{
	"button":            true,
	"link":              true,
	"textbox":           true,
	"searchbox":         true,
	"combobox":          true,
	"listbox":           true,
	"menu":              true,
	"menuitem":          true,
	"menuitemcheckbox":  true,
	"menuitemradio":     true,
	"tab":               true,
	"checkbox":          true,
	"radio":             true,
	"slider":            true,
	"spinbutton":        true,
	"switch":            true,
}
