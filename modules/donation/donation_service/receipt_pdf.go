package donation_service

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
)

// receiptPDFData menampung field yang dibutuhkan buildReceiptPDF, dilepas
// dari donation_dto.Response/donation_model.Donation supaya satu fungsi ini
// bisa dipakai baik oleh handler unduh publik (GetByPublicRef) maupun alur
// email otomatis (notifyDonationPaid) tanpa saling bergantung pada tipe
// spesifik masing-masing caller.
type receiptPDFData struct {
	PublicRef     string
	CampaignTitle string
	DonorName     string
	IsAnonymous   bool
	Amount        float64
	Message       string
	DateStr       string // sudah diformat pemanggil (mis. "3 Sep 2026, 22:33")
}

// buildReceiptPDF menghasilkan PDF "Bukti Donasi" satu halaman — layout
// sama persis dengan halaman web kantong-amal.donation-receipt (campaign,
// donatur, referensi, tanggal, status, pesan), dipakai baik untuk unduhan
// langsung maupun lampiran email donasi diterima (revision-prompt-4.md,
// item tambahan). Pure-Go (go-pdf/fpdf) — sengaja bukan HTML-to-PDF via
// binary eksternal (wkhtmltopdf/chromium) supaya tidak menambah dependensi
// deployment di server.
func buildReceiptPDF(d receiptPDFData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	const green = "00933b"
	r, g, b := hexToRGB(green)

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(r, g, b)
	pdf.CellFormat(0, 8, "FSLDK Indonesia", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 6, "Bukti Donasi", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetDrawColor(220, 220, 220)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetTextColor(r, g, b)
	pdf.CellFormat(0, 12, "Rp "+formatRupiah(d.Amount), "", 1, "C", false, 0, "")
	pdf.Ln(6)

	donor := d.DonorName
	if d.IsAnonymous {
		donor = "Anonim"
	}
	rows := [][2]string{
		{"Campaign", d.CampaignTitle},
		{"Donatur", donor},
		{"Referensi", d.PublicRef},
		{"Tanggal", d.DateStr},
		{"Status", "Lunas"},
	}
	if d.Message != "" {
		rows = append(rows, [2]string{"Pesan", "\"" + d.Message + "\""})
	}

	pdf.SetFont("Helvetica", "", 11)
	for _, row := range rows {
		y := pdf.GetY()
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(50, 8, row[0], "", 0, "L", false, 0, "")
		pdf.SetTextColor(30, 30, 30)
		pdf.SetXY(70, y)
		pdf.MultiCell(120, 8, row[1], "", "L", false)
		pdf.SetDrawColor(230, 230, 230)
		pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	}

	pdf.SetY(-30)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.MultiCell(0, 5, "Dokumen ini dibuat otomatis oleh sistem dan sah tanpa tanda tangan basah.", "", "C", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hexToRGB(hex string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
