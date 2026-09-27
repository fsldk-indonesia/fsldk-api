package upload

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// maxMainWidth caps the "display" variant (used for lightbox/full-view) so an
// unbounded phone-camera original (often 4000px+) never gets served whole.
const maxMainWidth = 1920

// maxThumbWidth caps the grid/card thumbnail variant.
const maxThumbWidth = 480

const mainJPEGQuality = 85
const thumbJPEGQuality = 75

// generateVariants produces the (main, thumb) byte pair stored for an
// uploaded image. Only .jpg/.jpeg/.png are actually resized: .gif is left
// untouched to avoid destroying animation, and .webp has no Go-native
// encoder to re-compress it after decoding, so both are passed through
// as-is for both variants. A decode failure on a supported format also
// falls back to the original bytes for both variants — a thumbnail that
// fails to generate must never fail the upload itself.
func generateVariants(original []byte, ext string) (main []byte, thumb []byte) {
	switch ext {
	case ".jpg", ".jpeg":
		return generateJPEGVariants(original)
	case ".png":
		return generatePNGVariants(original)
	default:
		return original, original
	}
}

func generateJPEGVariants(original []byte) ([]byte, []byte) {
	img, err := jpeg.Decode(bytes.NewReader(original))
	if err != nil {
		return original, original
	}
	main, mainOk := encodeJPEG(resizeToMaxWidth(img, maxMainWidth), mainJPEGQuality)
	thumb, thumbOk := encodeJPEG(resizeToMaxWidth(img, maxThumbWidth), thumbJPEGQuality)
	if !mainOk || !thumbOk {
		return original, original
	}
	return main, thumb
}

func generatePNGVariants(original []byte) ([]byte, []byte) {
	img, err := png.Decode(bytes.NewReader(original))
	if err != nil {
		return original, original
	}
	main, mainOk := encodePNG(resizeToMaxWidth(img, maxMainWidth))
	thumb, thumbOk := encodePNG(resizeToMaxWidth(img, maxThumbWidth))
	if !mainOk || !thumbOk {
		return original, original
	}
	return main, thumb
}

// resizeToMaxWidth scales img down so its width does not exceed maxWidth,
// preserving aspect ratio. Images already at or under maxWidth are returned
// unchanged — this never upscales.
func resizeToMaxWidth(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxWidth {
		return img
	}

	newHeight := int(float64(height) * float64(maxWidth) / float64(width))
	if newHeight < 1 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, maxWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image, quality int) ([]byte, bool) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

func encodePNG(img image.Image) ([]byte, bool) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

// thumbFileName derives the on-disk thumbnail filename from a main image
// filename by inserting a "_thumb" suffix before the extension — e.g.
// "abc123.jpg" -> "abc123_thumb.jpg". Kept as a plain filename transform
// (not a full path) so it composes with either a local path or a public URL.
func thumbFileName(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return base + "_thumb" + ext
}
