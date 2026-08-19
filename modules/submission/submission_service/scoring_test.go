package submission_service

import (
	"database/sql"
	"math"
	"testing"

	"fsldk-api/constants"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
)

func f64(v float64) sql.NullFloat64 { return sql.NullFloat64{Float64: v, Valid: true} }

func automaticField(id int64, code string, min, max, weight float64) submission_form_model.Field {
	return submission_form_model.Field{
		FieldID: id, FieldCode: code, FieldLabel: code, FieldType: constants.FieldTypeRadio,
		UseScoring: true, ScoringMethod: sql.NullString{String: constants.ScoringMethodAutomatic, Valid: true},
		MinScore: f64(min), MaxScore: f64(max), Weight: f64(weight),
	}
}

func manualField(id int64, code string, min, max, weight float64) submission_form_model.Field {
	return submission_form_model.Field{
		FieldID: id, FieldCode: code, FieldLabel: code, FieldType: constants.FieldTypeTextarea,
		UseScoring: true, ScoringMethod: sql.NullString{String: constants.ScoringMethodManual, Valid: true},
		MinScore: f64(min), MaxScore: f64(max), Weight: f64(weight),
	}
}

func option(id, fieldID int64, score float64) submission_form_model.Option {
	return submission_form_model.Option{OptionID: id, FieldID: fieldID, IsActive: true, Score: f64(score)}
}

func selectedAnswer(fieldID, optionID int64) submission_model.Answer {
	return submission_model.Answer{FieldID: fieldID, ValueOptionID: sql.NullInt64{Int64: optionID, Valid: true}}
}

func approxEqual(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// Skala 1-4 automatic penuh — persis contoh dokumen §3: Sangat Baik=4/4=100%,
// Baik=3/4=75%, Kurang=2/4=50%, Sangat Kurang=1/4=25%.
func TestComputeConsolidatedScore_Automatic1to4(t *testing.T) {
	field := automaticField(1, "ADMIN", 1, 4, 100)
	opts := []submission_form_model.Option{
		option(10, 1, 1), // Sangat Kurang
		option(11, 1, 2), // Kurang
		option(12, 1, 3), // Baik
		option(13, 1, 4), // Sangat Baik
	}
	cases := []struct {
		optionID int64
		want     float64
	}{
		{13, 100}, {12, 75}, {11, 50}, {10, 25},
	}
	for _, c := range cases {
		answers := []submission_model.Answer{selectedAnswer(1, c.optionID)}
		out := computeConsolidatedScore([]submission_form_model.Field{field}, opts, answers, nil)
		if !out.IsComplete {
			t.Fatalf("option %d: expected IsComplete=true", c.optionID)
		}
		approxEqual(t, "FinalScore", out.FinalScore, c.want)
		approxEqual(t, "Fields[0].Normalized", out.Fields[0].Normalized, c.want)
		if out.Fields[0].Source != constants.ScoringMethodAutomatic {
			t.Errorf("Source = %s, want AUTOMATIC", out.Fields[0].Source)
		}
	}
}

// Kombinasi automatic + manual, skala campuran, weight 20/30/30/20 — persis
// contoh dokumen §8: Field A 3/4 (20%) + Field B 5/5 (20%) + Field C 4/5
// (30%, manual) + Field D 7/10 (30%, manual) = 15+20+24+21 = 80%.
// (Angka field berbeda dari §7 dokumen — ini memverifikasi kombinasi
// automatic+manual §8, bukan mengulang §7.)
func TestComputeConsolidatedScore_MixedAutomaticAndManual(t *testing.T) {
	fields := []submission_form_model.Field{
		automaticField(1, "A", 0, 4, 20),
		automaticField(2, "B", 0, 5, 20),
		manualField(3, "C", 0, 5, 30),
		manualField(4, "D", 0, 10, 30),
	}
	opts := []submission_form_model.Option{
		option(10, 1, 3), // A: raw 3/4
		option(20, 2, 5), // B: raw 5/5
	}
	answers := []submission_model.Answer{
		selectedAnswer(1, 10),
		selectedAnswer(2, 20),
	}
	manual := map[int64]float64{3: 4, 4: 7} // C: 4/5, D: 7/10

	out := computeConsolidatedScore(fields, opts, answers, manual)
	if !out.IsComplete {
		t.Fatalf("expected IsComplete=true, fields=%+v", out.Fields)
	}
	// A: 3/4*20=15, B: 5/5*20=20, C: 4/5*30=24, D: 7/10*30=21 => 80
	approxEqual(t, "FinalScore", out.FinalScore, 80)
}

// Field belum dijawab (automatic) / belum diberi skor (manual) → IsComplete
// false, field itu tidak ikut dijumlah ke FinalScore tapi tetap muncul di
// breakdown dengan HasScore=false.
func TestComputeConsolidatedScore_Incomplete(t *testing.T) {
	fields := []submission_form_model.Field{
		automaticField(1, "A", 0, 4, 50),
		manualField(2, "B", 0, 5, 50),
	}
	opts := []submission_form_model.Option{option(10, 1, 4)}
	answers := []submission_model.Answer{selectedAnswer(1, 10)} // A dijawab, B belum diberi skor manual

	out := computeConsolidatedScore(fields, opts, answers, nil)
	if out.IsComplete {
		t.Fatalf("expected IsComplete=false karena field B belum diberi skor")
	}
	// Hanya A yang ikut terhitung: 4/4*50 = 50 (B dilewati, bukan dianggap 0)
	approxEqual(t, "FinalScore", out.FinalScore, 50)

	found := false
	for _, fs := range out.Fields {
		if fs.FieldID == 2 {
			found = true
			if fs.HasScore {
				t.Errorf("field B HasScore = true, want false")
			}
		}
	}
	if !found {
		t.Fatalf("field B tidak muncul di breakdown — seharusnya tetap muncul walau belum ada skor")
	}
}

// Skala custom non-berurutan (skor opsi 0, 3, 7, 10) — memverifikasi sistem
// tidak mengasumsikan skala selalu 1-4/berurutan (Important Development
// Instruction pada dokumen enhancement).
func TestComputeConsolidatedScore_CustomNonSequentialScale(t *testing.T) {
	field := automaticField(1, "CUSTOM", 0, 10, 100)
	opts := []submission_form_model.Option{
		option(10, 1, 0),  // Tidak Sesuai
		option(11, 1, 3),  // Kurang Sesuai
		option(12, 1, 7),  // Sesuai
		option(13, 1, 10), // Sangat Sesuai
	}
	answers := []submission_model.Answer{selectedAnswer(1, 12)} // Sesuai = 7/10 = 70%
	out := computeConsolidatedScore([]submission_form_model.Field{field}, opts, answers, nil)
	if !out.IsComplete {
		t.Fatalf("expected IsComplete=true")
	}
	approxEqual(t, "FinalScore", out.FinalScore, 70)
}

// Field tanpa UseScoring atau tanpa konfigurasi lengkap (maxScore/weight
// belum diisi) diabaikan sepenuhnya — memverifikasi backward-compat form
// lama yang belum pernah pakai scoring sama sekali.
func TestComputeConsolidatedScore_IgnoresNonScoringFields(t *testing.T) {
	plain := submission_form_model.Field{FieldID: 99, FieldCode: "PLAIN", FieldType: constants.FieldTypeText, UseScoring: false}
	out := computeConsolidatedScore([]submission_form_model.Field{plain}, nil, nil, nil)
	if len(out.Fields) != 0 {
		t.Fatalf("expected 0 fields in breakdown, got %d", len(out.Fields))
	}
	if !out.IsComplete {
		t.Fatalf("expected IsComplete=true when there are no scoring fields at all")
	}
	approxEqual(t, "FinalScore", out.FinalScore, 0)
}
