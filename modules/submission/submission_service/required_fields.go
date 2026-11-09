package submission_service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
)

// collectMissingRequiredFields mengumpulkan SEMUA field wajib yang belum
// terjawab (bukan cuma yang pertama ditemukan) agar Submit() bisa melapor
// sekali jalan field mana saja yang kurang, bukan memaksa pengirim mengulang
// submit satu per satu untuk tiap field yang kosong.
func collectMissingRequiredFields(
	fields []submission_form_model.Field,
	answerByField map[int64]submission_model.Answer,
	optionValueByID map[int64]string,
) []apperror.FieldError {
	var missing []apperror.FieldError
	for _, f := range fields {
		if !f.IsRequired {
			continue
		}
		if f.ConditionalOnFieldID.Valid {
			triggerAnswer, ok := answerByField[f.ConditionalOnFieldID.Int64]
			if !evaluateConditional(f.ConditionalRuleJSON, triggerAnswer, ok, optionValueByID) {
				continue
			}
		}
		a, ok := answerByField[f.FieldID]
		if ok && hasValue(a) {
			continue
		}
		missing = append(missing, apperror.FieldError{
			Attribute: "field_" + strconv.FormatInt(f.FieldID, 10),
			Field:     f.FieldLabel,
			Code:      constants.CodeValidationError,
			Message:   fmt.Sprintf("%s wajib diisi", f.FieldLabel),
		})
	}
	return missing
}

// collectMissingOtherText melaporkan field pilihan yang memilih opsi "Lainnya"
// tetapi isian manualnya masih kosong — diperiksa saat Submit (simpan draf
// boleh belum lengkap). Field kondisional yang sedang tidak aktif dilewati.
func collectMissingOtherText(
	fields []submission_form_model.Field,
	answerByField map[int64]submission_model.Answer,
	options []submission_form_model.Option,
	optionValueByID map[int64]string,
) []apperror.FieldError {
	var missing []apperror.FieldError
	for _, f := range fields {
		switch f.FieldType {
		case constants.FieldTypeSelect, constants.FieldTypeRadio, constants.FieldTypeMultiselect, constants.FieldTypeCheckbox:
		default:
			continue
		}
		a, ok := answerByField[f.FieldID]
		if !ok {
			continue
		}
		if f.ConditionalOnFieldID.Valid {
			triggerAnswer, has := answerByField[f.ConditionalOnFieldID.Int64]
			if !evaluateConditional(f.ConditionalRuleJSON, triggerAnswer, has, optionValueByID) {
				continue
			}
		}
		var chosen []int64
		if a.ValueOptionID.Valid {
			chosen = []int64{a.ValueOptionID.Int64}
		} else if a.ValueText.Valid && a.ValueText.String != "" {
			_ = json.Unmarshal([]byte(a.ValueText.String), &chosen)
		}
		fieldOptions := make([]submission_form_model.Option, 0)
		for _, o := range options {
			if o.FieldID == f.FieldID {
				fieldOptions = append(fieldOptions, o)
			}
		}
		if otherChosen(fieldOptions, chosen) && strings.TrimSpace(a.ValueOtherText.String) == "" {
			missing = append(missing, apperror.FieldError{
				Attribute: "field_" + strconv.FormatInt(f.FieldID, 10),
				Field:     f.FieldLabel,
				Code:      constants.CodeValidationError,
				Message:   fmt.Sprintf("%s: isian Lainnya wajib diisi", f.FieldLabel),
			})
		}
	}
	return missing
}
