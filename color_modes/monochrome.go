package color_modes

// NewMonochromeMode renders xBin images without ANSI color or intensity escape sequences.
func NewMonochromeMode() ColorMode {
	return monochromeMode{}
}

type monochromeMode struct{}

func (mode monochromeMode) ColorMapping(context Context) Mapping {
	return Mapping{
		escape: func(string) string {
			return ""
		},
		reset: func(string) string {
			return ""
		},
	}
}
