package submission_service

import (
	"math"

	"fsldk-api/constants"
	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
)

// computeConsolidatedScore menghitung Final Consolidated Score satu submission
// dari konfigurasi scoring field (Form Builder) + jawaban tersimpan + skor
// manual (Puskomnas) — enhancement Flexible Scoring (design/development/
// enahnce-development-submission-dashboard/new-enhance-development.md).
//
// Field AUTOMATIC (Single Choice) mengambil raw score dari score opsi yang
// dijawab LDK — TIDAK PERNAH disimpan terpisah, selalu dihitung ulang di sini
// supaya otomatis konsisten dengan migrateToLatestVersion (jawaban berubah →
// skor otomatis ikut berubah tanpa migrasi data skor). Field MANUAL mengambil
// raw score dari manualScores (tr_submission_field_score, diberikan
// Puskomnas). Field yang belum punya nilai (belum dijawab/belum diberi skor)
// tetap muncul di breakdown (HasScore=false) tapi tidak ikut dijumlah ke
// FinalScore, dan menandai keseluruhan hasil sebagai IsComplete=false.
//
// Tidak ada pembulatan di sepanjang perhitungan (normalized/weighted tetap
// float64 penuh) — pembulatan HANYA dilakukan sekali di titik akhir pada
// FinalScore, sesuai instruksi dokumen ("jangan melakukan pembulatan terlalu
// awal").
func computeConsolidatedScore(
	fields []submission_form_model.Field,
	options []submission_form_model.Option,
	answers []submission_model.Answer,
	manualScores map[int64]float64,
) submission_dto.ConsolidatedScoreResponse {
	optionScoreByField := make(map[int64]map[int64]float64)
	for _, o := range options {
		if !o.Score.Valid {
			continue
		}
		if optionScoreByField[o.FieldID] == nil {
			optionScoreByField[o.FieldID] = make(map[int64]float64)
		}
		optionScoreByField[o.FieldID][o.OptionID] = o.Score.Float64
	}
	answerByField := make(map[int64]submission_model.Answer, len(answers))
	for _, a := range answers {
		answerByField[a.FieldID] = a
	}

	out := submission_dto.ConsolidatedScoreResponse{
		Fields:     []submission_dto.FieldScoreResponse{},
		IsComplete: true,
	}
	var finalScore float64

	for _, f := range fields {
		if !f.UseScoring || !f.MaxScore.Valid || !f.Weight.Valid {
			continue
		}
		fs := submission_dto.FieldScoreResponse{
			FieldID:    f.FieldID,
			FieldCode:  f.FieldCode,
			FieldLabel: f.FieldLabel,
			MaxScore:   f.MaxScore.Float64,
			Weight:     f.Weight.Float64,
			Source:     f.ScoringMethod.String,
		}

		var raw float64
		hasScore := false
		switch f.ScoringMethod.String {
		case constants.ScoringMethodAutomatic:
			if answer, ok := answerByField[f.FieldID]; ok && answer.ValueOptionID.Valid {
				if score, found := optionScoreByField[f.FieldID][answer.ValueOptionID.Int64]; found {
					raw = score
					hasScore = true
				}
			}
		case constants.ScoringMethodManual:
			if score, ok := manualScores[f.FieldID]; ok {
				raw = score
				hasScore = true
			}
		}

		fs.HasScore = hasScore
		if hasScore {
			fs.RawScore = raw
			// Normalized & WeightedScore dilaporkan sebagai persen (0-100,
			// sama seperti skala Weight yang tersimpan) supaya sejalan
			// dengan contoh dokumen ("Normalized: 80%", "Weighted: 24%") —
			// weightedPercent = (raw/max) * weight langsung setara
			// "Contribution" pada dokumen, tanpa langkah pembulatan
			// antara.
			normalizedPercent := (raw / f.MaxScore.Float64) * 100
			weightedPercent := (raw / f.MaxScore.Float64) * f.Weight.Float64
			fs.Normalized = normalizedPercent
			fs.WeightedScore = weightedPercent
			finalScore += weightedPercent
		} else {
			out.IsComplete = false
		}
		out.Fields = append(out.Fields, fs)
	}

	// finalScore terakumulasi langsung dalam skala persen (0-100) — dibulatkan
	// HANYA di titik akhir ini, tidak pernah di tahap normalized/weighted.
	out.FinalScore = roundTo2(finalScore)
	return out
}

// roundTo2 membulatkan ke 2 desimal — dipakai hanya di titik akhir (FinalScore
// dalam persen), tidak pernah di tahap normalized/weighted per-field.
func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}
