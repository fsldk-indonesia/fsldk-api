package qrcode_service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"image"
	"image/color"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	// dukungan decode format gambar ikon yang lazim
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/modules/qrcode/qrcode_dto"
	"fsldk-api/modules/qrcode/qrcode_model"
	"fsldk-api/modules/qrcode/qrcode_repository"
	"fsldk-api/pkg/qrgen"
)

const (
	// Selaras tema sistem: #16211C = --color-text FSLDK (hitam kehijauan,
	// lebih lembut dari hitam murni), latar putih.
	defaultForeground = "#16211C"
	defaultBackground = "#FFFFFF"
)

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"label":       "q.label",
	"createdDate": "q.createdDate",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo       qrcode_repository.Repository
	apiBaseURL string
	uploadDir  string // folder fisik berkas unggahan (mis. "assets/uploads")
}

// NewService membuat Service QR Code. apiBaseURL dipakai membentuk ImageURL
// absolut; uploadDir adalah folder tempat berkas unggahan (ikon tengah)
// tersimpan di disk, dipakai membaca ikon saat merender gambar.
func NewService(repo qrcode_repository.Repository, apiBaseURL, uploadDir string) Service {
	return &ServiceImpl{
		repo:       repo,
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		uploadDir:  uploadDir,
	}
}

func (s *ServiceImpl) imageURL(id int64) string {
	return s.apiBaseURL + "/public/qrcodes/" + strconv.FormatInt(id, 10) + "/image"
}

func nz(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func (s *ServiceImpl) toResponse(q qrcode_model.QRCode) qrcode_dto.Response {
	return qrcode_dto.Response{
		QRCodeID:        q.QRCodeID,
		Label:           nz(q.Label),
		DestinationURL:  q.DestinationURL,
		ForegroundColor: q.ForegroundColor,
		BackgroundColor: q.BackgroundColor,
		CenterIconURL:   nz(q.CenterIconURL),
		CenterIconKey:   nz(q.CenterIconKey),
		CaptionText:     nz(q.CaptionText),
		ImageURL:        s.imageURL(q.QRCodeID),
		AuthorName:      q.AuthorName,
		CreatedDate:     q.CreatedDate.Format("2006-01-02 15:04:05"),
	}
}

func (s *ServiceImpl) List(ctx context.Context, q dto.ListQuery) ([]qrcode_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, qrcode_dto.ListFilter{
		Search:  q.Search,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "q.createdDate DESC"),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]qrcode_dto.Response, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponse(r))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (qrcode_dto.Response, error) {
	q, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return qrcode_dto.Response{}, apperror.NotFound("QR Code tidak ditemukan")
	}
	return s.toResponse(q), nil
}

// buildModel menormalkan input DTO ke nilai model siap-simpan (warna default,
// trim string).
func buildModel(destinationURL, label, fg, bg, icon, iconKey, caption string) qrcode_model.QRCode {
	fg = strings.TrimSpace(fg)
	bg = strings.TrimSpace(bg)
	if fg == "" {
		fg = defaultForeground
	}
	if bg == "" {
		bg = defaultBackground
	}
	return qrcode_model.QRCode{
		Label:           optString(label),
		DestinationURL:  strings.TrimSpace(destinationURL),
		ForegroundColor: strings.ToUpper(fg),
		BackgroundColor: strings.ToUpper(bg),
		CenterIconURL:   optString(icon),
		CenterIconKey:   optString(iconKey),
		CaptionText:     optString(caption),
	}
}

func optString(v string) sql.NullString {
	v = strings.TrimSpace(v)
	return sql.NullString{String: v, Valid: v != ""}
}

func (s *ServiceImpl) Create(ctx context.Context, req qrcode_dto.CreateRequest, actorID int64) (qrcode_dto.Response, error) {
	m := buildModel(req.DestinationURL, req.Label, req.ForegroundColor, req.BackgroundColor, req.CenterIconURL, req.CenterIconKey, req.CaptionText)
	m.CreatedBy = sql.NullInt64{Int64: actorID, Valid: actorID > 0}
	id, err := s.repo.Create(ctx, m)
	if err != nil {
		return qrcode_dto.Response{}, apperror.Internal("Gagal membuat QR Code")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req qrcode_dto.UpdateRequest, actorID int64) (qrcode_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return qrcode_dto.Response{}, apperror.NotFound("QR Code tidak ditemukan")
	}
	m := buildModel(req.DestinationURL, req.Label, req.ForegroundColor, req.BackgroundColor, req.CenterIconURL, req.CenterIconKey, req.CaptionText)
	m.UpdatedBy = sql.NullInt64{Int64: actorID, Valid: actorID > 0}
	if err := s.repo.Update(ctx, id, m); err != nil {
		return qrcode_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("QR Code tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Image(ctx context.Context, id int64, size int) ([]byte, error) {
	q, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("QR Code tidak ditemukan")
	}
	opts := qrgen.Options{
		Content:    q.DestinationURL,
		Size:       size,
		Foreground: parseHexColor(q.ForegroundColor, color.Black),
		Background: parseHexColor(q.BackgroundColor, color.White),
		Caption:    nz(q.CaptionText),
	}
	if icon := s.loadIcon(nz(q.CenterIconURL)); icon != nil {
		opts.CenterIcon = icon
	}
	png, gerr := qrgen.PNG(opts)
	if gerr != nil {
		return nil, apperror.Internal("Gagal membuat gambar QR")
	}
	return png, nil
}

// loadIcon memuat gambar ikon tengah dari nilai centerIconURL yang tersimpan.
// Dua bentuk didukung:
//   - data URI PNG/JPEG (ikon preset yang dikomposisi di frontend — warna sudah
//     dibakar sesuai warna QR): "data:image/png;base64,...."
//   - URL unggahan lokal (kustom, hanya dari CMS): "http://host/uploads/<token>.png"
//
// Bentuk lain / berkas hilang / gagal decode diabaikan — QR tetap dirender
// tanpa ikon.
func (s *ServiceImpl) loadIcon(centerIconURL string) image.Image {
	if centerIconURL == "" {
		return nil
	}
	if strings.HasPrefix(centerIconURL, "data:image/") {
		i := strings.IndexByte(centerIconURL, ',')
		if i < 0 {
			return nil
		}
		raw, err := base64.StdEncoding.DecodeString(centerIconURL[i+1:])
		if err != nil {
			return nil
		}
		img, _, derr := image.Decode(bytes.NewReader(raw))
		if derr != nil {
			return nil
		}
		return img
	}

	if s.uploadDir == "" || !strings.Contains(centerIconURL, "/uploads/") {
		return nil
	}
	name := path.Base(centerIconURL)
	if name == "" || name == "." || name == "/" || strings.Contains(name, "..") {
		return nil
	}
	f, err := os.Open(filepath.Join(s.uploadDir, name))
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

// parseHexColor menerjemahkan "#RRGGBB" (case-insensitive, "#" opsional) ke
// color.RGBA. Nilai tidak valid mengembalikan def.
func parseHexColor(s string, def color.Color) color.Color {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return def
	}
	var rgb [3]uint8
	for i := 0; i < 3; i++ {
		hi, ok1 := hexVal(s[i*2])
		lo, ok2 := hexVal(s[i*2+1])
		if !ok1 || !ok2 {
			return def
		}
		rgb[i] = hi<<4 | lo
	}
	return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 0xFF}
}

func hexVal(c byte) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
