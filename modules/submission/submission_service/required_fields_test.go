package submission_service

import (
	"database/sql"
	"testing"

	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
)

func TestCollectMissingRequiredFields_ReturnsEveryMissingRequiredField(t *testing.T) {
	fields := []submission_form_model.Field{
		{FieldID: 1, FieldLabel: "Nama", IsRequired: true},
		{FieldID: 2, FieldLabel: "Alamat", IsRequired: true},
		{FieldID: 3, FieldLabel: "Catatan", IsRequired: false},
	}
	answerByField := map[int64]submission_model.Answer{
		3: {FieldID: 3, ValueText: sql.NullString{String: "opsional", Valid: true}},
	}

	got := collectMissingRequiredFields(fields, answerByField, nil)

	if len(got) != 2 {
		t.Fatalf("expected 2 missing required fields, got %d: %+v", len(got), got)
	}
	if got[0].Attribute != "field_1" || got[0].Field != "Nama" {
		t.Errorf("got[0] = %+v, want attribute field_1 / field Nama", got[0])
	}
	if got[1].Attribute != "field_2" || got[1].Field != "Alamat" {
		t.Errorf("got[1] = %+v, want attribute field_2 / field Alamat", got[1])
	}
}

func TestCollectMissingRequiredFields_SkipsConditionalFieldNotTriggered(t *testing.T) {
	fields := []submission_form_model.Field{
		{FieldID: 1, FieldLabel: "Sudah menikah?", IsRequired: true},
		{
			FieldID: 2, FieldLabel: "Nama pasangan", IsRequired: true,
			ConditionalOnFieldID: sql.NullInt64{Int64: 1, Valid: true},
			ConditionalRuleJSON:  sql.NullString{String: `{"operator":"equals","value":"Ya"}`, Valid: true},
		},
	}
	answerByField := map[int64]submission_model.Answer{
		1: {FieldID: 1, ValueText: sql.NullString{String: "Tidak", Valid: true}},
	}

	got := collectMissingRequiredFields(fields, answerByField, nil)

	if len(got) != 0 {
		t.Fatalf("expected conditional field to be skipped since trigger rule not met, got %+v", got)
	}
}
