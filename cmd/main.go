package main

import (
	"errors"
	"fmt"
	"os"

	xbin "github.com/Crushless/go-ansiart"
	char_sets "github.com/Crushless/go-ansiart/char_sets"
	color_modes "github.com/Crushless/go-ansiart/color_modes"
	"github.com/jessevdk/go-flags"
)

var args struct {
	Format              string `short:"f" long:"format" choice:"ansi" choice:"bin" choice:"xbin" description:"input format (ansi, bin or xbin)" default:"ansi"`
	Output              string `short:"O" long:"output" choice:"ansi" choice:"kitty" choice:"sixel" description:"output format (ansi, kitty or sixel)" default:"ansi"`
	CharSet             string `short:"s" long:"charset" choice:"cp437" choice:"pet" choice:"amiga"  description:"character set to use for rendering (cp437, pet, amiga)"`
	Columns             uint   `short:"w" long:"width" description:"number of columns for ANSI stream wrapping (ignored for xbin input)"`
	ColorMode           string `short:"c" long:"color-mode" choice:"truecolor" choice:"monochrome" choice:"halfbright" description:"color mode to use for rendering (truecolor, monochrome, halfbright"`
	HalfbrightAlgorithm string `short:"a" long:"halfbright-algorithm" choice:"palette" choice:"luminance" description:"algorithm to use for halfbright color mode (palette, luminance)"`
	Opaque              bool   `short:"o" long:"opaque" description:"render with opaque background instead of transparent"`
	FillSolidBlocks     bool   `short:"b" long:"fill-solid-blocks" description:"render full-block characters as filled terminal cells to avoid glyph gaps"`
	DisableAutoWrap     bool   `long:"disable-autowrap" description:"disable terminal autowrap while rendering rectangular artwork"`
	StableColumns       bool   `long:"stable-columns" description:"reposition the cursor after each cell to keep block glyphs aligned"`
}

func main() {
	parser := flags.NewParser(&args, flags.HelpFlag|flags.PassDoubleDash)
	fileNames, err := parser.Parse()
	if err != nil {
		if !errors.Is(err, flags.ErrHelp) {
			fmt.Println(err)
		}
		return
	}

	charSet, err := decodeCharSet(args.CharSet)
	if err != nil {
		fmt.Println(err)
		return
	}
	colorMode, err := decodeColorMode(args.ColorMode, args.HalfbrightAlgorithm)
	if err != nil {
		fmt.Println(err)
		return
	}

	options := xbin.Options{
		ColorMode:       colorMode,
		Opaque:          args.Opaque,
		Charset:         charSet,
		Columns:         int(args.Columns),
		FillSolidBlocks: args.FillSolidBlocks,
		DisableAutoWrap: args.DisableAutoWrap,
		StableColumns:   args.StableColumns,
	}

	for _, fileName := range fileNames {
		if err := processFile(fileName, options); err != nil {
			fmt.Println(err)
			continue
		}
	}
}

func processFile(fileName string, options xbin.Options) error {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	img, err := parseImage(data, options)
	if err != nil {
		return err
	}
	rendered, err := renderImage(img, options)
	if err != nil {
		return err
	}
	fmt.Print(rendered)
	if args.Output == "ansi" {
		fmt.Println()
	}
	return nil
}

func parseImage(data []byte, options xbin.Options) (*xbin.Image, error) {
	switch args.Format {
	case "ansi":
		return xbin.ParseANSI(data, options)
	case "bin":
		return xbin.ParseBIN(data, options)
	case "xbin":
		return xbin.Parse(data, options)
	default:
		return nil, fmt.Errorf("invalid input format: %q", args.Format)
	}
}

func renderImage(img *xbin.Image, options xbin.Options) (string, error) {
	switch args.Output {
	case "ansi":
		return xbin.Render(img, options), nil
	case "kitty":
		return xbin.RenderKitty(img, options)
	case "sixel":
		return xbin.RenderSixel(img, options)
	default:
		return "", fmt.Errorf("invalid output format: %q", args.Output)
	}
}

func decodeColorMode(colorMode string, halfbrightAlgorithm string) (color_modes.ColorMode, error) {
	switch colorMode {
	case "truecolor":
		return color_modes.NewTrueColorMode(), nil
	case "monochrome":
		return color_modes.NewMonochromeMode(), nil
	case "halfbright":
		switch halfbrightAlgorithm {
		case "palette":
			return color_modes.NewHalfBrightMode(color_modes.HalfBrightByPaletteColumn), nil
		case "luminance":
			return color_modes.NewHalfBrightMode(color_modes.HalfBrightByLuminance), nil
		case "":
			return color_modes.NewHalfBrightMode(color_modes.HalfBrightByPaletteColumn), nil
		default:
			return nil, fmt.Errorf("invalid halfbright algorithm: %q", halfbrightAlgorithm)
		}
	case "":
		return color_modes.NewTrueColorMode(), nil
	default:
		return nil, fmt.Errorf("invalid color mode: %q", colorMode)
	}
}

func decodeCharSet(format string) (*char_sets.Charset, error) {
	switch format {
	case "cp437":
		return char_sets.CP437, nil
	case "pet":
		return char_sets.PETSCII, nil
	case "amiga":
		return char_sets.AmigaTopaz, nil
	case "":
		return char_sets.CP437, nil
	default:
		return nil, fmt.Errorf("invalid charset: %q", format)
	}
}
