// Package qrgen membuat gambar PNG QR code yang bisa dikustomisasi: warna
// depan/belakang, ikon di tengah, dan teks caption di bawah kode. Pembungkus
// di atas github.com/skip2/go-qrcode (matriks QR) + golang.org/x/image
// (compositing ikon & render teks) — mengisolasi dependensi dari modul bisnis.
package qrgen

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	qrcode "github.com/skip2/go-qrcode"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// DefaultSize adalah sisi (piksel) area matriks QR bila tidak ditentukan.
const DefaultSize = 512

// Options adalah parameter pembuatan satu gambar QR.
type Options struct {
	Content    string      // isi yang di-encode (URL tujuan langsung)
	Size       int         // sisi area QR dalam piksel; <=0 => DefaultSize
	Foreground color.Color // warna modul QR; nil => hitam
	Background color.Color // warna latar; nil => putih
	CenterIcon image.Image // opsional; digambar ~20% di tengah dengan bantalan latar
	Caption    string      // opsional; digambar satu baris di bawah QR
}

// PNG merender Options menjadi byte PNG.
func PNG(o Options) ([]byte, error) {
	size := o.Size
	if size <= 0 {
		size = DefaultSize
	}
	fg := o.Foreground
	if fg == nil {
		fg = color.Black
	}
	bg := o.Background
	if bg == nil {
		bg = color.White
	}

	// Ikon di tengah "melubangi" sebagian modul — pakai level koreksi
	// tertinggi (~30%) supaya kode tetap terbaca.
	level := qrcode.Medium
	if o.CenterIcon != nil {
		level = qrcode.Highest
	}

	q, err := qrcode.New(o.Content, level)
	if err != nil {
		return nil, err
	}
	q.ForegroundColor = fg
	q.BackgroundColor = bg
	qrImg := q.Image(size)

	captionH := 0
	var face font.Face
	if o.Caption != "" {
		face, err = newFace(float64(size) * 0.052)
		if err != nil {
			return nil, err
		}
		defer face.Close()
		captionH = int(float64(size) * 0.16)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, size, size+captionH))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(0, 0, size, size), qrImg, image.Point{}, draw.Src)

	if o.CenterIcon != nil {
		drawCenterIcon(canvas, size, o.CenterIcon, bg)
	}
	if o.Caption != "" {
		drawCaption(canvas, size, captionH, o.Caption, face, fg)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawCenterIcon menggambar ikon di tengah QR menjaga rasio aspek asli
// (contain-fit ke kotak ~20% lebar QR — TIDAK dipaksa persegi sehingga tidak
// gepeng), di atas bantalan latar sedikit lebih besar supaya modul di
// belakangnya "bersih".
func drawCenterIcon(dst *image.RGBA, qrSize int, icon image.Image, bg color.Color) {
	box := qrSize * 20 / 100
	ib := icon.Bounds()
	iw, ih := ib.Dx(), ib.Dy()
	if iw <= 0 || ih <= 0 {
		return
	}
	// skala contain: sisi terpanjang ikon dipetakan ke `box`.
	dw, dh := box, box
	if iw >= ih {
		dh = box * ih / iw
	} else {
		dw = box * iw / ih
	}
	cx, cy := qrSize/2, qrSize/2

	// Bantalan latar tipis — ikon preset sudah membawa kotak putihnya sendiri;
	// untuk logo kustom pun cukup sedikit ruang napas.
	padW := dw*106/100 + 1
	padH := dh*106/100 + 1
	pad := image.Rect(cx-padW/2, cy-padH/2, cx-padW/2+padW, cy-padH/2+padH)
	draw.Draw(dst, pad, image.NewUniform(bg), image.Point{}, draw.Src)

	target := image.Rect(cx-dw/2, cy-dh/2, cx-dw/2+dw, cy-dh/2+dh)
	xdraw.CatmullRom.Scale(dst, target, icon, ib, xdraw.Over, nil)
}

// drawCaption menggambar satu baris teks di bawah area QR, rata tengah,
// dipotong dengan elipsis bila melebihi lebar kanvas.
func drawCaption(dst *image.RGBA, qrSize, captionH int, text string, face font.Face, fg color.Color) {
	maxW := fixed.I(qrSize - qrSize/12)
	trimmed := text
	for font.MeasureString(face, trimmed) > maxW && len(trimmed) > 1 {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if trimmed != text {
		for font.MeasureString(face, trimmed+"…") > maxW && len(trimmed) > 1 {
			trimmed = trimmed[:len(trimmed)-1]
		}
		trimmed += "…"
	}

	adv := font.MeasureString(face, trimmed)
	metrics := face.Metrics()
	x := (fixed.I(qrSize) - adv) / 2
	y := fixed.I(qrSize) + fixed.I(captionH)/2 + (metrics.Ascent-metrics.Descent)/2

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(fg),
		Face: face,
		Dot:  fixed.Point26_6{X: x, Y: y},
	}
	d.DrawString(trimmed)
}

// newFace memuat font Go Regular (tertanam di golang.org/x/image) pada ukuran
// piksel yang diberikan.
func newFace(px float64) (font.Face, error) {
	ttf, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(ttf, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
}
