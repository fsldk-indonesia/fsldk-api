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
