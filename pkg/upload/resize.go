package upload

import (
	"bytes"
	"encoding/binary"
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
	// Phone cameras store portrait shots as landscape pixel data plus an
	// EXIF Orientation tag telling viewers how to rotate it for display.
	// jpeg.Decode ignores that tag, and the bytes we write below carry no
	// EXIF of their own, so the pixel data must be corrected here — the
	// only chance a viewer gets to see it upright.
	img = applyOrientation(img, parseJPEGOrientation(original))
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

// parseJPEGOrientation scans a JPEG byte stream for an embedded EXIF
// Orientation tag (0x0112) and returns its value, or 1 ("normal", a no-op
// for applyOrientation) if the tag or the whole EXIF segment is absent or
// malformed — callers can always apply the returned value safely.
func parseJPEGOrientation(data []byte) uint16 {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			break
		}
		marker := data[pos+1]
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9) {
			// no-payload markers (TEM, RSTn, SOI, EOI)
			pos += 2
			continue
		}
		length := int(data[pos+2])<<8 | int(data[pos+3])
		if length < 2 || pos+2+length > len(data) {
			return 1
		}
		segment := data[pos+4 : pos+2+length]
		if marker == 0xE1 && len(segment) >= 6 && string(segment[:6]) == "Exif\x00\x00" {
			if v, ok := parseTIFFOrientation(segment[6:]); ok {
				return v
			}
			return 1
		}
		if marker == 0xDA { // SOS: compressed scan data follows, nothing left to scan
			break
		}
		pos += 2 + length
	}
	return 1
}

// parseTIFFOrientation reads the Orientation tag (0x0112) out of a raw TIFF
// structure (the payload of an EXIF APP1 segment, after the "Exif\0\0"
// header) — just enough of the TIFF/IFD format to find that one tag.
func parseTIFFOrientation(tiff []byte) (uint16, bool) {
	if len(tiff) < 8 {
		return 0, false
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0, false
	}
	pos := int(bo.Uint32(tiff[4:8]))
	if pos+2 > len(tiff) {
		return 0, false
	}
	numEntries := int(bo.Uint16(tiff[pos : pos+2]))
	pos += 2
	for i := 0; i < numEntries; i++ {
		if pos+12 > len(tiff) {
			return 0, false
		}
		if bo.Uint16(tiff[pos:pos+2]) == 0x0112 && bo.Uint16(tiff[pos+2:pos+4]) == 3 {
			return bo.Uint16(tiff[pos+8 : pos+10]), true
		}
		pos += 12
	}
	return 0, false
}

// applyOrientation returns img corrected for the given EXIF Orientation
// value so the pixel data itself is stored upright. Needed because the
// resized/re-encoded bytes generateJPEGVariants writes carry no EXIF of
// their own — nothing is left for a viewer to rotate by afterwards.
// Orientation 1 (normal) or an unrecognized value is a no-op.
func applyOrientation(img image.Image, orientation uint16) image.Image {
	switch orientation {
	case 2:
		return flipHorizontal(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipVertical(img)
	case 5:
		return transpose(img)
	case 6:
		return rotate90CW(img)
	case 7:
		return transverse(img)
	case 8:
		return rotate270CW(img)
	default:
		return img
	}
}

// remap builds a new dstW x dstH RGBA image by placing each source pixel
// (x,y) at mapCoord(x,y) in the destination — the shared primitive behind
// every orientation transform below.
func remap(img image.Image, dstW, dstH int, mapCoord func(x, y int) (int, int)) *image.RGBA {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dx, dy := mapCoord(x, y)
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flipHorizontal(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, w, h, func(x, y int) (int, int) { return w - 1 - x, y })
}

func flipVertical(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, w, h, func(x, y int) (int, int) { return x, h - 1 - y })
}

func rotate180(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, w, h, func(x, y int) (int, int) { return w - 1 - x, h - 1 - y })
}

// rotate90CW rotates the image 90 degrees clockwise (EXIF orientation 6):
// the stored bottom-left corner becomes the displayed top-left.
func rotate90CW(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, h, w, func(x, y int) (int, int) { return h - 1 - y, x })
}

// rotate270CW rotates 90 degrees counter-clockwise (EXIF orientation 8).
func rotate270CW(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, h, w, func(x, y int) (int, int) { return y, w - 1 - x })
}

// transpose mirrors across the top-left/bottom-right diagonal (EXIF orientation 5).
func transpose(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, h, w, func(x, y int) (int, int) { return y, x })
}

// transverse mirrors across the top-right/bottom-left diagonal (EXIF orientation 7).
func transverse(img image.Image) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return remap(img, h, w, func(x, y int) (int, int) { return h - 1 - y, w - 1 - x })
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
