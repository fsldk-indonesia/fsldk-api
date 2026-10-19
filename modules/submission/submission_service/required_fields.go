package submission_service

import (
	"fmt"
	"strconv"

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
