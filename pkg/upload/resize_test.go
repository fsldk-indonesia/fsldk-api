package upload

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodeTestJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("failed to encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

func encodeTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	return buf.Bytes()
}

func decodedSize(t *testing.T, data []byte, decode func([]byte) (image.Config, error)) (int, int) {
	t.Helper()
	cfg, err := decode(data)
	if err != nil {
		t.Fatalf("failed to decode result image: %v", err)
	}
	return cfg.Width, cfg.Height
}

func jpegConfig(data []byte) (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(data)) }
func pngConfig(data []byte) (image.Config, error)  { return png.DecodeConfig(bytes.NewReader(data)) }

func TestGenerateVariants_DownscalesLargeJPEG(t *testing.T) {
	original := encodeTestJPEG(t, 3000, 2000)

	main, thumb := generateVariants(original, ".jpg")

	mainW, _ := decodedSize(t, main, jpegConfig)
	thumbW, _ := decodedSize(t, thumb, jpegConfig)

	if mainW != maxMainWidth {
		t.Errorf("expected main width %d, got %d", maxMainWidth, mainW)
	}
	if thumbW != maxThumbWidth {
		t.Errorf("expected thumb width %d, got %d", maxThumbWidth, thumbW)
	}
}

func TestGenerateVariants_DoesNotUpscaleSmallJPEG(t *testing.T) {
	original := encodeTestJPEG(t, 200, 150)

	main, thumb := generateVariants(original, ".jpg")

	mainW, mainH := decodedSize(t, main, jpegConfig)
	thumbW, thumbH := decodedSize(t, thumb, jpegConfig)

	if mainW != 200 || mainH != 150 {
		t.Errorf("expected main to stay 200x150 (no upscale), got %dx%d", mainW, mainH)
	}
	if thumbW != 200 || thumbH != 150 {
		t.Errorf("expected thumb to stay 200x150 (no upscale), got %dx%d", thumbW, thumbH)
	}
}

func TestGenerateVariants_KeepsPNGFormatWhenResizing(t *testing.T) {
	original := encodeTestPNG(t, 2400, 1600)

	main, thumb := generateVariants(original, ".png")

	mainW, _ := decodedSize(t, main, pngConfig)
	thumbW, _ := decodedSize(t, thumb, pngConfig)

	if mainW != maxMainWidth {
		t.Errorf("expected main width %d, got %d", maxMainWidth, mainW)
	}
	if thumbW != maxThumbWidth {
		t.Errorf("expected thumb width %d, got %d", maxThumbWidth, thumbW)
	}
}

func TestGenerateVariants_WebpIsNotResized(t *testing.T) {
	// Not a real webp decode target — generateVariants must not even attempt
	// to decode .webp (no Go-native encoder to re-encode it after resizing),
	// so arbitrary bytes must survive untouched for both variants.
	original := []byte("fake-webp-bytes-not-a-real-image")

	main, thumb := generateVariants(original, ".webp")

	if !bytes.Equal(main, original) {
		t.Errorf("expected main bytes to equal original for .webp, got different bytes")
	}
	if !bytes.Equal(thumb, original) {
		t.Errorf("expected thumb bytes to equal original for .webp, got different bytes")
	}
}

func TestGenerateVariants_GifIsNotResized(t *testing.T) {
	img := image.NewPaletted(image.Rect(0, 0, 10, 10), []color.Color{color.White, color.Black})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode test gif: %v", err)
	}
	original := buf.Bytes()

	main, thumb := generateVariants(original, ".gif")

	if !bytes.Equal(main, original) {
		t.Errorf("expected main bytes to equal original for .gif (animation must not be touched)")
	}
	if !bytes.Equal(thumb, original) {
		t.Errorf("expected thumb bytes to equal original for .gif (animation must not be touched)")
	}
}

// buildExifOrientationSegment builds a minimal JPEG APP1 (Exif) segment
// containing only the Orientation tag (little-endian TIFF), simulating what
// a phone camera writes for a portrait shot — without pulling in a real
// EXIF-writing dependency just for tests.
func buildExifOrientationSegment(orientation uint16) []byte {
	tiff := []byte{
		'I', 'I', 0x2A, 0x00, // TIFF header: little-endian byte order + magic 42
		0x08, 0x00, 0x00, 0x00, // offset to IFD0
		0x01, 0x00, // IFD0: 1 entry
		0x12, 0x01, // tag 0x0112 (Orientation)
		0x03, 0x00, // type 3 (SHORT)
		0x01, 0x00, 0x00, 0x00, // count 1
		byte(orientation), byte(orientation >> 8), 0x00, 0x00, // value + padding
		0x00, 0x00, 0x00, 0x00, // next IFD offset (none)
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	length := len(payload) + 2 // length field counts itself
	segment := []byte{0xFF, 0xE1, byte(length >> 8), byte(length)}
	return append(segment, payload...)
}

// encodeTestJPEGWithOrientation encodes a landscape width x height JPEG
// (bottom half red, top half blue) tagged with the given EXIF orientation,
// inserted as an APP1 segment right after SOI like a real camera JPEG.
func encodeTestJPEGWithOrientation(t *testing.T, width, height int, orientation uint16) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < height/2 {
				img.Set(x, y, color.RGBA{B: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("failed to encode base jpeg: %v", err)
	}
	base := buf.Bytes()
	out := make([]byte, 0, len(base)+64)
	out = append(out, base[:2]...) // SOI
	out = append(out, buildExifOrientationSegment(orientation)...)
	out = append(out, base[2:]...)
	return out
}

func TestGenerateVariants_CorrectsExifOrientation(t *testing.T) {
	// Stored pixel grid is landscape (32x16, bottom half red) tagged
	// orientation 6 — "rotate 90 CW to display correctly", the same tag
	// phones write for a portrait photo. Since the resized bytes we write
	// carry no EXIF for a viewer to rotate by, the pixel data itself must
	// already be corrected: portrait (16x32), with the stored bottom-left
	// corner (red) now at the displayed top-left.
	original := encodeTestJPEGWithOrientation(t, 32, 16, 6)

	main, thumb := generateVariants(original, ".jpg")

	for name, variant := range map[string][]byte{"main": main, "thumb": thumb} {
		img, err := jpeg.Decode(bytes.NewReader(variant))
		if err != nil {
			t.Fatalf("%s: failed to decode corrected image: %v", name, err)
		}
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		if w != h/2 {
			t.Fatalf("%s: expected corrected image to be portrait (width = height/2), got %dx%d", name, w, h)
		}
		r, _, b, _ := img.At(0, 0).RGBA()
		if r <= b {
			t.Errorf("%s: expected top-left of corrected image to be red (from stored bottom-left), got r=%d b=%d", name, r, b)
		}
	}
}

func TestGenerateVariants_FallsBackToOriginalWhenJPEGDecodeFails(t *testing.T) {
	corrupt := []byte("this is not a valid jpeg file at all")

	main, thumb := generateVariants(corrupt, ".jpg")

	if !bytes.Equal(main, corrupt) {
		t.Errorf("expected main to fall back to original bytes on decode failure")
	}
	if !bytes.Equal(thumb, corrupt) {
		t.Errorf("expected thumb to fall back to original bytes on decode failure")
	}
}
