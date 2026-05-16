package color_modes

// Color is one resolved 8-bit RGB palette entry.
type Color struct {
	R byte
	G byte
	B byte
}

// Override carries an exact color parsed from a source format.
type Override struct {
	Color   Color
	Index   int
	Set     bool
	Indexed bool
}

// Context contains the image-specific color state needed by a color mode.
type Context struct {
	Palette        [16]Color
	HighBackground bool
	Opaque         bool
}

// ColorMode describes how xBin attributes become terminal style escapes.
type ColorMode interface {
	ColorMapping(context Context) Mapping
}

// Mapping is a prepared, image-specific lookup table for terminal style changes.
type Mapping struct {
	styles             [256]string
	backgrounds        [256]bool
	initialStyle       string
	escape             func(string) string
	reset              func(string) string
	styleWithOverrides func(byte, Override, Override) string
	hasBackground      func(byte, Override, Override) bool
	canFillSolidBlocks bool
}

func (mapping Mapping) Style(attr byte) string {
	return mapping.styles[attr]
}

func (mapping Mapping) HasBackground(attr byte) bool {
	return mapping.backgrounds[attr]
}

func (mapping Mapping) StyleWithOverrides(attr byte, foreground Override, background Override) string {
	if mapping.styleWithOverrides != nil {
		return mapping.styleWithOverrides(attr, foreground, background)
	}
	return mapping.Style(attr)
}

func (mapping Mapping) HasBackgroundWithOverride(attr byte, foreground Override, background Override) bool {
	if mapping.hasBackground != nil {
		return mapping.hasBackground(attr, foreground, background)
	}
	return mapping.HasBackground(attr)
}

func (mapping Mapping) CanFillSolidBlocks() bool {
	return mapping.canFillSolidBlocks
}

func (mapping Mapping) Escape(style string) string {
	if mapping.escape == nil {
		return style
	}
	return mapping.escape(style)
}

func (mapping Mapping) InitialStyle() string {
	return mapping.initialStyle
}

func (mapping Mapping) Reset(style string) string {
	if mapping.reset != nil {
		return mapping.reset(style)
	}
	return "\x1b[0m"
}

func attrColors(attr byte, highBackground bool) (int, int) {
	foreground := int(attr & 0x0f)
	background := int((attr >> 4) & 0x07)
	if highBackground {
		background = int((attr >> 4) & 0x0f)
	}
	return foreground, background
}

func luminance(color Color) int {
	return (299*int(color.R) + 587*int(color.G) + 114*int(color.B)) / 1000
}
