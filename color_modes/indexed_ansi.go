package color_modes

import "fmt"

// NewIndexedANSIMode renders xBin images using terminal ANSI color indices.
func NewIndexedANSIMode() ColorMode {
	return indexedANSIMode{}
}

var xbinToANSIColorIndex = [16]int{
	0, 4, 2, 6, 1, 5, 3, 7,
	8, 12, 10, 14, 9, 13, 11, 15,
}

type indexedANSIMode struct{}

func (mode indexedANSIMode) ColorMapping(context Context) Mapping {
	mapping := Mapping{
		canFillSolidBlocks: true,
		styleWithOverrides: func(attr byte, foreground Override, background Override) string {
			fgIndex, bgIndex := attrColors(attr, context.HighBackground)
			foregroundIndex := xbinToANSIColorIndex[fgIndex]
			backgroundIndex := xbinToANSIColorIndex[bgIndex]
			if foreground.Set && foreground.Indexed {
				foregroundIndex = clampANSIIndex(foreground.Index)
			}
			if background.Set && background.Indexed {
				backgroundIndex = clampANSIIndex(background.Index)
				return fmt.Sprintf("\x1b[38;5;%dm\x1b[48;5;%dm", foregroundIndex, backgroundIndex)
			}
			if bgIndex == 0 && !context.Opaque {
				return fmt.Sprintf("\x1b[38;5;%dm\x1b[49m", foregroundIndex)
			}
			return fmt.Sprintf("\x1b[38;5;%dm\x1b[48;5;%dm", foregroundIndex, backgroundIndex)
		},
		hasBackground: func(attr byte, foreground Override, background Override) bool {
			if background.Set && background.Indexed {
				return true
			}
			_, backgroundIndex := attrColors(attr, context.HighBackground)
			return backgroundIndex != 0 || context.Opaque
		},
	}
	for attr := range mapping.styles {
		foreground, background := attrColors(byte(attr), context.HighBackground)
		foreground = xbinToANSIColorIndex[foreground]
		background = xbinToANSIColorIndex[background]
		if background == 0 && !context.Opaque {
			mapping.styles[attr] = fmt.Sprintf("\x1b[38;5;%dm\x1b[49m", foreground)
			continue
		}
		mapping.styles[attr] = fmt.Sprintf("\x1b[38;5;%dm\x1b[48;5;%dm", foreground, background)
		mapping.backgrounds[attr] = true
	}
	return mapping
}

func clampANSIIndex(index int) int {
	if index < 0 {
		return 0
	}
	if index > 255 {
		return 255
	}
	return index
}
