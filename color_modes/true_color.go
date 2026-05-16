package color_modes

import "fmt"

// NewTrueColorMode renders images using exact RGB colors when present, embedded XBin palettes, or the default palette.
func NewTrueColorMode() ColorMode {
	return trueColorMode{}
}

type trueColorMode struct{}

func (mode trueColorMode) ColorMapping(context Context) Mapping {
	mapping := Mapping{
		canFillSolidBlocks: true,
		styleWithOverrides: func(attr byte, foreground Override, background Override) string {
			fgIndex, bgIndex := attrColors(attr, context.HighBackground)
			fg := context.Palette[fgIndex]
			if foreground.Set {
				fg = overrideColor(foreground)
			}
			if background.Set {
				bg := overrideColor(background)
				return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", fg.R, fg.G, fg.B, bg.R, bg.G, bg.B)
			}
			if bgIndex == 0 && !context.Opaque {
				return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[49m", fg.R, fg.G, fg.B)
			}

			bg := context.Palette[bgIndex]
			return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", fg.R, fg.G, fg.B, bg.R, bg.G, bg.B)
		},
		hasBackground: func(attr byte, foreground Override, background Override) bool {
			if background.Set {
				return true
			}
			_, bgIndex := attrColors(attr, context.HighBackground)
			return bgIndex != 0 || context.Opaque
		},
	}
	for attr := range mapping.styles {
		foreground, background := attrColors(byte(attr), context.HighBackground)
		fg := context.Palette[foreground]
		if background == 0 && !context.Opaque {
			mapping.styles[attr] = fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[49m", fg.R, fg.G, fg.B)
			continue
		}

		bg := context.Palette[background]
		mapping.styles[attr] = fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", fg.R, fg.G, fg.B, bg.R, bg.G, bg.B)
		mapping.backgrounds[attr] = true
	}
	return mapping
}

func overrideColor(override Override) Color {
	if override.Indexed {
		return XTerm256Color(override.Index)
	}
	return override.Color
}

// XTerm256Color returns the RGB color for an xterm 256-color palette index.
func XTerm256Color(index int) Color {
	if index < 0 {
		index = 0
	}
	if index > 255 {
		index = 255
	}
	if index < 16 {
		return xtermANSIColors[index]
	}
	if index < 232 {
		index -= 16
		return Color{
			R: xtermColorCubeComponent(index / 36),
			G: xtermColorCubeComponent((index / 6) % 6),
			B: xtermColorCubeComponent(index % 6),
		}
	}
	gray := byte(8 + (index-232)*10)
	return Color{R: gray, G: gray, B: gray}
}

var xtermANSIColors = [16]Color{
	{R: 0x00, G: 0x00, B: 0x00},
	{R: 0x80, G: 0x00, B: 0x00},
	{R: 0x00, G: 0x80, B: 0x00},
	{R: 0x80, G: 0x80, B: 0x00},
	{R: 0x00, G: 0x00, B: 0x80},
	{R: 0x80, G: 0x00, B: 0x80},
	{R: 0x00, G: 0x80, B: 0x80},
	{R: 0xc0, G: 0xc0, B: 0xc0},
	{R: 0x80, G: 0x80, B: 0x80},
	{R: 0xff, G: 0x00, B: 0x00},
	{R: 0x00, G: 0xff, B: 0x00},
	{R: 0xff, G: 0xff, B: 0x00},
	{R: 0x00, G: 0x00, B: 0xff},
	{R: 0xff, G: 0x00, B: 0xff},
	{R: 0x00, G: 0xff, B: 0xff},
	{R: 0xff, G: 0xff, B: 0xff},
}

func xtermColorCubeComponent(value int) byte {
	if value == 0 {
		return 0
	}
	return byte(55 + value*40)
}
