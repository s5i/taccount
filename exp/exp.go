//go:build windows

package exp

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/deluan/lookup"

	_ "embed"
	_ "image/png"
)

func New() (*Reader, error) {
	ocr := lookup.NewOCR(0.6)
	if err := ocr.LoadFont(fontPath); err != nil {
		return nil, err
	}
	return &Reader{
		ocr: ocr,
	}, nil
}

type Reader struct {
	ocr *lookup.OCR
}

func (r *Reader) Read() (int, bool, error) {
	windowImg, err := captureWindow("Tibiantis")
	if err != nil {
		return 0, false, fmt.Errorf(`CaptureWindow("Tibiantis") failed: %v`, err)
	}

	expMatch, err := lookup.NewLookup(windowImg).FindAll(expKeyImg, 0.6)
	if err != nil {
		return 0, false, fmt.Errorf(`lookup.FindAll(expImg) failed: %v`, err)
	}

	if len(expMatch) == 0 {
		return 0, false, nil
	}

	x := expMatch[0].X + expKeyImg.Bounds().Dx()
	y := expMatch[0].Y
	dx := 80
	dy := 10
	expValueImg := windowImg.SubImage(image.Rect(x, y, x+dx, y+dy))

	expStr, err := r.ocr.Recognize(expValueImg)
	if err != nil {
		return 0, false, fmt.Errorf("ocr.Recognize() failed: %v", err)
	}

	exp, err := strconv.Atoi(expStr)
	if err != nil {
		return 0, false, err
	}

	return exp, true, nil
}

var (
	//go:embed assets/experience.png
	expKeyBytes []byte
	expKeyImg   image.Image

	//go:embed assets/font
	fontFS embed.FS

	basePath = filepath.Join(os.TempDir(), "tassist")
	fontPath = filepath.Join(basePath, "assets", "font")
)

func init() {
	os.RemoveAll(basePath)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		log.Fatalf("os.MkdirAll(%q) failed: %v", basePath, err)
	}
	if err := os.CopyFS(basePath, fontFS); err != nil {
		log.Fatalf("os.CopyFS(%q) failed: %v", basePath, err)
	}

	eImg, _, err := image.Decode(bytes.NewReader(expKeyBytes))
	if err != nil {
		log.Fatalf("image.Decode(expKeyBytes) failed: %v", err)
	}
	expKeyImg = eImg
}
