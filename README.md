# go-ansiart

Render DOS ANSI art formats as terminal text from Go.

`go-ansiart` decodes ANSI (`.ans`), BIN (`.bin`), and XBin (`.xb`) byte streams and returns a string containing Unicode text plus terminal escape sequences. It supports ANSI SGR colors, optional XBin palettes, compressed XBin image data, raw textmode BIN cells, multiple character sets, several terminal color modes, and bitmap rendering with embedded or fallback fonts.

The text renderer maps character bytes through Unicode charset tables. The bitmap renderers rasterize embedded XBin fonts when present, otherwise they use the bundled 8x16 fallback font, and can emit Kitty graphics protocol or Sixel output.

## Install

```sh
go get github.com/Crushless/go-ansiart
```

## Basic Usage

```go
package main

import (
	"fmt"
	"log"
	"os"

	ansiart "github.com/Crushless/go-ansiart"
	"github.com/Crushless/go-ansiart/char_sets"
	"github.com/Crushless/go-ansiart/color_modes"
)

func main() {
	data, err := os.ReadFile("art.xb")
	if err != nil {
		log.Fatal(err)
	}

	out, err := ansiart.Decode(data, ansiart.Options{
		ColorMode: color_modes.NewTrueColorMode(),
		Charset:   char_sets.CP437,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(out)
}
```

Use the decoder that matches the source format:

```go
ansiText, err := ansiart.DecodeANSI(ansBytes)
binText, err := ansiart.DecodeBIN(binBytes, ansiart.Options{Columns: 80})
xbinText, err := ansiart.Decode(xbBytes)
```

Parsing and rendering are separate. Each input format has its own parser, and all parsers produce the same `ansiart.Image` structure for the shared renderer:

```go
ansiImage, err := ansiart.ParseANSI(ansBytes)
binImage, err := ansiart.ParseBIN(binBytes, ansiart.Options{Columns: 80})
xbinImage, err := ansiart.Parse(xbBytes)

out := ansiart.Render(xbinImage, ansiart.Options{
	ColorMode: color_modes.NewTrueColorMode(),
	Charset:   char_sets.CP437,
})
```

Images can also be rasterized instead of rendered as Unicode text. XBin embedded fonts are used when present; otherwise the bundled 8x16 fallback font is used:

```go
xbinImage, err := ansiart.Parse(xbBytes)
if err != nil {
	log.Fatal(err)
}

kitty, err := ansiart.RenderKitty(xbinImage, ansiart.Options{Opaque: true})
if err != nil {
	log.Fatal(err)
}

sixel, err := ansiart.RenderSixel(xbinImage, ansiart.Options{Opaque: true})
if err != nil {
	log.Fatal(err)
}

fmt.Print(kitty)
fmt.Print(sixel)
```

Reader helpers are also available:

```go
ansiText, err := ansiart.DecodeANSIReader(r)
binText, err := ansiart.DecodeBINReader(r)
xbinText, err := ansiart.DecodeReader(r)
```

## CLI

The repository includes a small renderer command:

```sh
go run ./cmd -f ansi art.ans
go run ./cmd -f bin -w 80 art.bin
go run ./cmd -f xbin art.xb
```

Useful flags include:

```text
-f, --format              input format: ansi, bin, or xbin
-s, --charset             character set: cp437, pet, or amiga
-w, --width               ANSI/BIN column width, ignored for XBin
-c, --color-mode          truecolor, monochrome, or halfbright
-a, --halfbright-algorithm palette or luminance
-o, --opaque              render palette index 0 as an opaque background
    --disable-autowrap    disable terminal autowrap while rendering
    --stable-columns      reposition after each cell for stricter alignment
```

## Options

`Options` values are optional. Defaults are true color, CP437, transparent background index 0, and 80 columns for ANSI/BIN streams.

```go
ansiart.Options{
	ColorMode:       color_modes.NewTrueColorMode(),
	Charset:         char_sets.CP437,
	Columns:         80,
	Opaque:          false,
	FillSolidBlocks: false,
	DisableAutoWrap: false,
	StableColumns:   false,
}
```

## Color Modes

True color preserves ANSI 24-bit SGR colors, converts ANSI 256-color SGR colors to RGB, and uses the embedded XBin palette when present. XBin files without a palette use the default DOS palette.

```go
ansiart.Options{ColorMode: color_modes.NewTrueColorMode()}
```

Indexed ANSI preserves ANSI 256-color SGR colors and maps XBin/DOS palette indices to terminal ANSI color indices.

```go
ansiart.Options{ColorMode: color_modes.NewIndexedANSIMode()}
```

Monochrome emits only text and line breaks.

```go
ansiart.Options{ColorMode: color_modes.NewMonochromeMode()}
```

Half-bright monochrome emits SGR 2 faint/half-bright for dark colors. Palette-column mode follows the classic low/high DOS color split:

```go
ansiart.Options{
	ColorMode: color_modes.NewHalfBrightMode(color_modes.HalfBrightByPaletteColumn),
}
```

For visual downgrading to monochrome terminals, luminance mode is often better:

```go
ansiart.Options{
	ColorMode: color_modes.NewHalfBrightMode(color_modes.HalfBrightByLuminance),
}
```

## Character Sets

```go
ansiart.Options{Charset: char_sets.CP437}
ansiart.Options{Charset: char_sets.PETSCII}
ansiart.Options{Charset: char_sets.AmigaTopaz}
```

`CP437` is the default when no charset is specified.

## Terminal Rendering

By default, background palette index 0 is treated as transparent and rendered as the terminal default background. Set `Opaque` to render it as a color:

```go
ansiart.Options{Opaque: true}
```

Some terminals or fonts draw the Unicode full-block glyph with tiny side bearings, which can show thin gaps between solid color cells. Set `FillSolidBlocks` to render those cells as filled backgrounds instead:

```go
ansiart.Options{FillSolidBlocks: true}
```

For terminals that handle autowrap differently at the rightmost column, set `DisableAutoWrap` while rendering rectangular artwork:

```go
ansiart.Options{DisableAutoWrap: true}
```

For terminals that apply unexpected widths to block-element glyphs, set `StableColumns` to keep each rendered cell aligned without replacing the artist's chosen characters:

```go
ansiart.Options{StableColumns: true}
```

## Supported Features

| Feature                    | State | Notes                                           |
| :------------------------- | :---: | :---------------------------------------------- |
| ANSI byte streams          |  yes  | CP437 text plus common cursor/SGR CSI           |
| ANSI 24-bit true color SGR |  yes  | foreground and background                       |
| ANSI 256-color SGR         |  yes  | preserved or converted by color mode            |
| BIN raw textmode cells     |  yes  | configurable width, defaults to 80              |
| XBin image data            |  yes  | uncompressed and compressed streams             |
| XBin embedded palette      |  yes  | used when present                               |
| XBin embedded fonts        |  yes  | preferred by bitmap/Kitty/Sixel renderers       |
| Bitmap fallback font       |  yes  | bundled 8x16 PSFU font                          |
| Kitty graphics protocol    |  yes  | PNG payload generated from bitmap rendering     |
| Sixel graphics protocol    |  yes  | generated from bitmap rendering, up to 256 colors |

## Links

- [Moebius](https://github.com/christiansacks/moebius/), an ANSI and ASCII art editor
- [MoebiusXBIN](https://blog.glyphdrawing.club/moebiusxbin-ascii-and-text-mode-art-editor-with-custom-font-support)
- [XBIN format specification](https://www.acid.org/images/0896/XBIN.TXT)
