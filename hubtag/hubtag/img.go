package hubtag

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/image/font"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

var fnt *truetype.Font

const fontPath = "/usr/share/fonts/TTF/DejaVuSerifCondensed-Bold.ttf"

func init() {
	b, err := ioutil.ReadFile(fontPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load font: %s: %s", fontPath, err)
		os.Exit(1)
	}
	fnt, err = freetype.ParseFont(b)
	if err != nil {
		panic(err)
	}
}

func renderCount(val int) (io.Reader, error) {
	rgba := image.NewRGBA(image.Rect(0, 0, 50, 20))
	draw.Draw(rgba, rgba.Bounds(), image.White, image.ZP, draw.Src)
	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetFont(fnt)
	ctx.SetFontSize(14)
	ctx.SetClip(rgba.Bounds())
	ctx.SetDst(rgba)
	ctx.SetSrc(image.Black)
	ctx.SetHinting(font.HintingFull)
	pt := freetype.Pt(2, int(ctx.PointToFixed(12)>>6)+2)

	text := fmt.Sprintf("+%d", val)
	if _, err := ctx.DrawString(text, pt); err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := png.Encode(&b, rgba); err != nil {
		return nil, err
	}
	return &b, nil
}
