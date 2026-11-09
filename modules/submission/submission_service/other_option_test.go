package submission_service

import (
	"database/sql"
	"strings"
	"testing"

	"fsldk-api/constants"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
)

func TestNormalizeOtherText(t *testing.T) {
	blank := "   "
	got, err := normalizeOtherText(&blank)
	if err != nil || got.Valid {
		t.Fatalf("blank must be NULL, got %+v err=%v", got, err)
	}
	v := "  Kuliah di luar negeri  "
	got, err = normalizeOtherText(&v)
	if err != nil || got.String != "Kuliah di luar negeri" {
		t.Fatalf("trim failed: %+v err=%v", got, err)
	}
	long := strings.Repeat("a", otherTextMaxLen+1)
	if _, err := normalizeOtherText(&long); err == nil {
		t.Fatal("over-length text must be rejected")
	}
}

func TestOtherChosen(t *testing.T) {
	opts := []submission_form_model.Option{{OptionID: 1}, {OptionID: 2, IsOther: true}}
	if otherChosen(opts, []int64{1}) {
		t.Error("regular option is not Lainnya")
	}
	if !otherChosen(opts, []int64{1, 2}) {
		t.Error("Lainnya must be detected among multiple choices")
	}
}

func TestCollectMissingOtherText(t *testing.T) {
	fields := []submission_form_model.Field{{FieldID: 10, FieldLabel: "Pekerjaan", FieldType: constants.FieldTypeSelect}}
	opts := []submission_form_model.Option{{OptionID: 5, FieldID: 10, IsOther: true}}
	empty := map[int64]submission_model.Answer{10: {FieldID: 10, ValueOptionID: sql.NullInt64{Int64: 5, Valid: true}}}
	if got := collectMissingOtherText(fields, empty, opts, nil); len(got) != 1 {
		t.Fatalf("expected 1 missing, got %d", len(got))
	}
	filled := map[int64]submission_model.Answer{10: {FieldID: 10, ValueOptionID: sql.NullInt64{Int64: 5, Valid: true}, ValueOtherText: sql.NullString{String: "Wirausaha", Valid: true}}}
	if got := collectMissingOtherText(fields, filled, opts, nil); len(got) != 0 {
		t.Fatalf("expected none missing, got %d", len(got))
	}
}
