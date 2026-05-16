package color_modes

type HalfBrightAlgorithm int8

const (
	// HalfBrightByPaletteColumn treats xBin palette indices 0-7 as dark and 8-15 as bright.
	HalfBrightByPaletteColumn HalfBrightAlgorithm = iota
	// HalfBrightByLuminance treats colors with low perceived brightness as dark.
	HalfBrightByLuminance
)

// NewHalfBrightMode renders xBin images without color, using SGR 2 faint/half-bright for dark colors.
func NewHalfBrightMode(algorithm HalfBrightAlgorithm) ColorMode {
	return halfBrightMode{algorithm: algorithm}
}

type halfBrightMode struct {
	algorithm HalfBrightAlgorithm
}

func (mode halfBrightMode) ColorMapping(context Context) Mapping {
	mapping := Mapping{
		initialStyle: halfBrightStyleNormal,
		escape: func(style string) string {
			if style == halfBrightStyleFaint {
				return "\x1b[2m"
			}
			return "\x1b[22m"
		},
		reset: func(style string) string {
			if style == halfBrightStyleFaint {
				return "\x1b[0m"
			}
			return ""
		},
	}
	for attr := range mapping.styles {
		foreground := byte(attr & 0x0f)
		if halfBright(foreground, context.Palette, mode.algorithm) {
			mapping.styles[attr] = halfBrightStyleFaint
			continue
		}
		mapping.styles[attr] = halfBrightStyleNormal
	}
	return mapping
}

const (
	halfBrightStyleNormal = "normal"
	halfBrightStyleFaint  = "faint"
)

func halfBright(foreground byte, palette [16]Color, algorithm HalfBrightAlgorithm) bool {
	if algorithm == HalfBrightByLuminance {
		if foreground > 8 {
			return false
		}
		return luminance(palette[foreground]) < 128
	}
	return foreground < 8
}
