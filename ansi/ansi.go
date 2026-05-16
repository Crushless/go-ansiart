package ansi

import (
	"bytes"
	"fmt"
	"io"

	"org.kremm/go-ansiart/color_modes"
	"org.kremm/go-ansiart/shared"
)

const (
	ansiDefaultColumns = 80
	ansiEsc            = 0x1b
	ansiSub            = 0x1a
)

var ansiToXBinColorIndex = [16]byte{
	0, 4, 2, 6, 1, 5, 3, 7,
	8, 12, 10, 14, 9, 13, 11, 15,
}

// Parse decodes a DOS ANSI/CP437 byte stream into the common parsed image model.
func Parse(data []byte, options ...shared.Options) (*shared.Image, error) {
	usedOptions := shared.NormalizeOptions(options...)
	if usedOptions.Columns <= 0 {
		usedOptions.Columns = ansiDefaultColumns
	}

	screen := newANSIScreen(usedOptions.Columns)
	screen.parse(stripANSISauce(data))
	return screen.image(), nil
}

// DecodeANSI decodes a DOS ANSI/CP437 byte stream and renders it with the selected options.
func DecodeANSI(data []byte, options ...shared.Options) (string, error) {
	usedOptions := shared.NormalizeOptions(options...)
	image, err := Parse(data, usedOptions)
	if err != nil {
		return "", err
	}
	return shared.Render(image, usedOptions), nil
}

type ansiCell struct {
	char byte
	attr byte
	fg   color_modes.Override
	bg   color_modes.Override
}

type ansiScreen struct {
	columns        int
	cells          []ansiCell
	rowMaxX        []int
	x              int
	y              int
	maxY           int
	savedX         int
	savedY         int
	hasSaved       bool
	pendingWrap    bool
	lastChar       byte
	hasLastChar    bool
	fg             byte
	bg             byte
	fgRGB          color_modes.Override
	bgRGB          color_modes.Override
	bold           bool
	blink          bool
	inverse        bool
	highBackground bool
}

func newANSIScreen(columns int) *ansiScreen {
	screen := &ansiScreen{columns: columns}
	screen.resetAttributes()
	return screen
}

func (screen *ansiScreen) resetAttributes() {
	screen.fg = 7
	screen.bg = 0
	screen.fgRGB = color_modes.Override{}
	screen.bgRGB = color_modes.Override{}
	screen.bold = false
	screen.blink = false
	screen.inverse = false
}

func (screen *ansiScreen) parse(data []byte) {
	for i := 0; i < len(data); {
		switch data[i] {
		case '\r':
			screen.x = 0
			screen.pendingWrap = false
			i++
		case '\n':
			screen.newLine()
			i++
		case ansiEsc:
			next, ok := screen.parseEscape(data, i)
			if !ok {
				i++
				continue
			}
			i = next
		default:
			screen.put(data[i])
			i++
		}
	}
}

func (screen *ansiScreen) parseEscape(data []byte, start int) (int, bool) {
	if start+1 >= len(data) || data[start+1] != '[' {
		return start + 1, false
	}

	i := start + 2
	values := []int{}
	current := 0
	hasValue := false
	for i < len(data) {
		code := data[i]
		switch {
		case code >= '0' && code <= '9':
			current = current*10 + int(code-'0')
			hasValue = true
			i++
		case code == ';' || code == ':':
			if hasValue {
				values = append(values, current)
			} else {
				values = append(values, 0)
			}
			current = 0
			hasValue = false
			i++
		case code >= '@' && code <= '~':
			if hasValue || len(values) > 0 {
				values = append(values, current)
			}
			screen.applyEscape(code, values)
			return i + 1, true
		default:
			return i + 1, false
		}
	}
	return len(data), false
}

func (screen *ansiScreen) applyEscape(final byte, values []int) {
	switch final {
	case 'A':
		screen.y -= ansiValue(values, 0, 1)
		if screen.y < 0 {
			screen.y = 0
		}
		screen.pendingWrap = false
	case 'B':
		screen.y += ansiValue(values, 0, 1)
		screen.updateMaxY()
		screen.pendingWrap = false
	case 'C':
		screen.applyPendingWrap()
		screen.x += ansiValue(values, 0, 1)
		if screen.x >= screen.columns {
			screen.x = screen.columns - 1
		}
		screen.pendingWrap = false
	case 'D':
		screen.x -= ansiValue(values, 0, 1)
		if screen.x < 0 {
			screen.x = 0
		}
		screen.pendingWrap = false
	case 'E':
		screen.y += ansiValue(values, 0, 1)
		screen.x = 0
		screen.updateMaxY()
		screen.pendingWrap = false
	case 'F':
		screen.y -= ansiValue(values, 0, 1)
		if screen.y < 0 {
			screen.y = 0
		}
		screen.x = 0
		screen.pendingWrap = false
	case 'G', '`':
		col := ansiValue(values, 0, 1)
		screen.x = min(max(col-1, 0), screen.columns-1)
		screen.pendingWrap = false
	case 'H', 'f':
		row := ansiValue(values, 0, 1)
		col := ansiValue(values, 1, 1)
		screen.y = max(row-1, 0)
		screen.x = min(max(col-1, 0), screen.columns-1)
		screen.updateMaxY()
		screen.pendingWrap = false
	case 'J':
		if ansiValue(values, 0, 0) == 2 {
			screen.cells = nil
			screen.x = 0
			screen.y = 0
			screen.maxY = 0
			screen.pendingWrap = false
		}
	case 'K':
		screen.clearLine(ansiValue(values, 0, 0))
	case '@':
		screen.insertCharacters(ansiValue(values, 0, 1))
	case 'P':
		screen.deleteCharacters(ansiValue(values, 0, 1))
	case 'X':
		screen.eraseCharacters(ansiValue(values, 0, 1))
	case 'b':
		screen.repeatLastCharacter(ansiValue(values, 0, 1))
	case 'm':
		screen.applySGR(values)
	case 's':
		screen.savedX = screen.x
		screen.savedY = screen.y
		screen.hasSaved = true
	case 'u':
		if screen.hasSaved {
			screen.x = screen.savedX
			screen.y = screen.savedY
			screen.pendingWrap = false
		}
	}
}

func (screen *ansiScreen) applySGR(values []int) {
	if len(values) == 0 {
		values = []int{0}
	}
	for i := 0; i < len(values); i++ {
		value := values[i]
		switch {
		case value == 0:
			screen.resetAttributes()
		case value == 1:
			screen.bold = true
		case value == 5:
			screen.blink = true
			screen.highBackground = true
		case value == 7:
			screen.inverse = true
		case value == 22:
			screen.bold = false
		case value == 21 || value == 25:
			screen.blink = false
		case value == 27:
			screen.inverse = false
		case value == 39:
			screen.fg = 7
			screen.fgRGB = color_modes.Override{}
		case value == 49:
			screen.bg = 0
			screen.bgRGB = color_modes.Override{}
		case value >= 30 && value <= 37:
			screen.fg = ansiToXBinColorIndex[value-30]
			screen.fgRGB = color_modes.Override{}
		case value >= 40 && value <= 47:
			screen.bg = ansiToXBinColorIndex[value-40]
			screen.bgRGB = color_modes.Override{}
		case value >= 90 && value <= 97:
			screen.fg = ansiToXBinColorIndex[value-90] + 8
			screen.fgRGB = color_modes.Override{}
		case value >= 100 && value <= 107:
			screen.bg = ansiToXBinColorIndex[value-100] + 8
			screen.bgRGB = color_modes.Override{}
			screen.highBackground = true
		case (value == 38 || value == 48) && i+2 < len(values) && values[i+1] == 5:
			index := clampANSIColor(values[i+2])
			override := color_modes.Override{
				Index:   index,
				Set:     true,
				Indexed: true,
			}
			if index < 16 {
				color := ansiToXBinColorIndex[index]
				if value == 38 {
					screen.fg = color
				} else {
					screen.bg = color
					if color >= 8 {
						screen.highBackground = true
					}
				}
			}
			if value == 38 {
				screen.fgRGB = override
			} else {
				screen.bgRGB = override
			}
			i += 2
		case (value == 38 || value == 48) && i+4 < len(values) && values[i+1] == 2:
			color := color_modes.Override{
				Color: color_modes.Color{
					R: byte(clampANSIColor(values[i+2])),
					G: byte(clampANSIColor(values[i+3])),
					B: byte(clampANSIColor(values[i+4])),
				},
				Set: true,
			}
			if value == 38 {
				screen.fgRGB = color
			} else {
				screen.bgRGB = color
			}
			i += 4
		}
	}
}

func (screen *ansiScreen) put(char byte) {
	screen.applyPendingWrap()

	screen.set(screen.x, screen.y, screen.cell(char))
	screen.lastChar = char
	screen.hasLastChar = true
	screen.x++
	if screen.x == screen.columns {
		screen.pendingWrap = true
	}
	screen.updateMaxY()
}

func (screen *ansiScreen) applyPendingWrap() {
	if !screen.pendingWrap {
		return
	}
	screen.x = 0
	screen.y++
	screen.pendingWrap = false
	screen.updateMaxY()
}

func (screen *ansiScreen) attr() byte {
	attr, _, _ := screen.style()
	return attr
}

func (screen *ansiScreen) cell(char byte) ansiCell {
	attr, fg, bg := screen.style()
	return ansiCell{char: char, attr: attr, fg: fg, bg: bg}
}

func (screen *ansiScreen) style() (byte, color_modes.Override, color_modes.Override) {
	fg := screen.fg
	bg := screen.bg
	fgRGB := screen.fgRGB
	bgRGB := screen.bgRGB
	if screen.bold && fg < 8 {
		fg += 8
	}
	if screen.blink && bg < 8 {
		bg += 8
	}
	if screen.inverse {
		fg, bg = bg, fg
		fgRGB, bgRGB = bgRGB, fgRGB
	}
	return fg | (bg << 4), fgRGB, bgRGB
}

func (screen *ansiScreen) newLine() {
	screen.pendingWrap = false
	screen.x = 0
	screen.y++
	screen.updateMaxY()
}

func (screen *ansiScreen) clearLine(mode int) {
	switch mode {
	case 0:
		for x := screen.x; x < screen.columns; x++ {
			screen.set(x, screen.y, screen.cell(' '))
		}
	case 1:
		for x := 0; x <= screen.x; x++ {
			screen.set(x, screen.y, screen.cell(' '))
		}
	case 2:
		for x := 0; x < screen.columns; x++ {
			screen.set(x, screen.y, screen.cell(' '))
		}
	}
}

func (screen *ansiScreen) insertCharacters(count int) {
	screen.pendingWrap = false
	if count <= 0 || screen.x >= screen.columns {
		return
	}
	if count > screen.columns-screen.x {
		count = screen.columns - screen.x
	}
	for x := screen.columns - 1; x >= screen.x+count; x-- {
		screen.set(x, screen.y, screen.cellAt(x-count, screen.y))
	}
	for x := screen.x; x < screen.x+count; x++ {
		screen.set(x, screen.y, screen.cell(' '))
	}
}

func (screen *ansiScreen) deleteCharacters(count int) {
	screen.pendingWrap = false
	if count <= 0 || screen.x >= screen.columns {
		return
	}
	if count > screen.columns-screen.x {
		count = screen.columns - screen.x
	}
	for x := screen.x; x < screen.columns-count; x++ {
		screen.set(x, screen.y, screen.cellAt(x+count, screen.y))
	}
	for x := screen.columns - count; x < screen.columns; x++ {
		screen.set(x, screen.y, screen.cell(' '))
	}
}

func (screen *ansiScreen) eraseCharacters(count int) {
	screen.pendingWrap = false
	if count <= 0 || screen.x >= screen.columns {
		return
	}
	if count > screen.columns-screen.x {
		count = screen.columns - screen.x
	}
	for x := screen.x; x < screen.x+count; x++ {
		screen.set(x, screen.y, screen.cell(' '))
	}
}

func (screen *ansiScreen) repeatLastCharacter(count int) {
	if !screen.hasLastChar || count <= 0 {
		return
	}
	for range count {
		screen.put(screen.lastChar)
	}
}

func (screen *ansiScreen) set(x int, y int, cell ansiCell) {
	if x < 0 || x >= screen.columns || y < 0 {
		return
	}
	index := y*screen.columns + x
	for len(screen.cells) <= index {
		screen.cells = append(screen.cells, ansiCell{char: ' ', attr: 0x07})
	}
	screen.cells[index] = cell
	for len(screen.rowMaxX) <= y {
		screen.rowMaxX = append(screen.rowMaxX, -1)
	}
	if x > screen.rowMaxX[y] {
		screen.rowMaxX[y] = x
	}
	if y > screen.maxY {
		screen.maxY = y
	}
}

func (screen *ansiScreen) updateMaxY() {
	if screen.y > screen.maxY {
		screen.maxY = screen.y
	}
}

func (screen *ansiScreen) cellAt(x int, y int) ansiCell {
	index := y*screen.columns + x
	cell := ansiCell{char: ' ', attr: 0x07}
	if index < len(screen.cells) {
		cell = screen.cells[index]
		if cell.char == 0 {
			cell.char = ' '
		}
	}
	return cell
}

func (screen *ansiScreen) image() *shared.Image {
	height := screen.maxY + 1
	if len(screen.cells) == 0 {
		height = 0
	}
	width := screen.trimmedWidth()
	cells := make([]shared.Cell, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			outIndex := y*width + x
			inIndex := y*screen.columns + x
			cell := ansiCell{char: ' ', attr: 0x07}
			if inIndex < len(screen.cells) {
				cell = screen.cells[inIndex]
				if cell.char == 0 {
					cell.char = ' '
				}
			}
			cells[outIndex] = shared.Cell{
				Char: cell.char,
				Attr: cell.attr,
				FG:   cell.fg,
				BG:   cell.bg,
			}
		}
	}

	return &shared.Image{
		Width:          width,
		Height:         height,
		Cells:          cells,
		Palette:        shared.DefaultPalette,
		HighBackground: screen.highBackground,
	}
}

func (screen *ansiScreen) trimmedWidth() int {
	width := 0
	for _, maxX := range screen.rowMaxX {
		if maxX+1 > width {
			width = maxX + 1
		}
	}
	if width == 0 && len(screen.cells) > 0 {
		width = 1
	}
	return width
}

func ansiValue(values []int, index int, fallback int) int {
	if index >= len(values) || values[index] == 0 {
		return fallback
	}
	return values[index]
}

func clampANSIColor(value int) int {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return value
}

func stripANSISauce(data []byte) []byte {
	if len(data) >= 128 && bytes.Equal(data[len(data)-128:len(data)-123], []byte("SAUCE")) {
		data = data[:len(data)-128]
		if len(data) > 0 && data[len(data)-1] == ansiSub {
			data = data[:len(data)-1]
		}
	}
	return data
}

func DecodeANSIReader(r io.Reader, options ...shared.Options) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read ansi: %w", err)
	}
	return DecodeANSI(data, options...)
}
