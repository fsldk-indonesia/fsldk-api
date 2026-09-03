package dynamicform_service

import (
	"encoding/json"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_model"
)

// conditionalLogic mirrors conditionalLogicJSON.
type conditionalLogic struct {
	Action     string `json:"action"` // show | hide
	Match      string `json:"match"`  // all | any
	Conditions []struct {
		FieldID  int64  `json:"fieldID"`
		Operator string `json:"operator"` // eq|neq|contains|gt|lt|filled|empty
		Value    string `json:"value"`
	} `json:"conditions"`
}

// fieldValidation mirrors validationJSON.
type fieldValidation struct {
	Min           *int     `json:"min"`
	Max           *int     `json:"max"`
	Pattern       *string  `json:"pattern"`
	AcceptedTypes []string `json:"acceptedTypes"`
	MaxSizeKB     *int     `json:"maxSizeKB"`
}

func isDisplayType(t string) bool {
	for _, d := range constants.DynamicFormDisplayFieldTypes {
		if d == t {
			return true
		}
	}
	return false
}

func firstVal(values map[int64][]string, id int64) string {
	if v, ok := values[id]; ok && len(v) > 0 {
		return strings.TrimSpace(v[0])
	}
	return ""
}

func allVals(values map[int64][]string, id int64) []string {
	out := make([]string, 0)
	for _, v := range values[id] {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// optionValues returns the allowed values of a choice field.
func optionValues(f dynamicform_model.Field) []string {
	if f.OptionsJSON == nil || *f.OptionsJSON == "" {
		return nil
	}
	var opts []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}
	if json.Unmarshal([]byte(*f.OptionsJSON), &opts) != nil {
		return nil
	}
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.Value)
	}
	return out
}

// fieldVisible re-evaluates a field's conditionalLogic against the submitted
// values so a conditionally-hidden field is never held to its `required` rule.
func fieldVisible(f dynamicform_model.Field, values map[int64][]string) bool {
	if f.ConditionalLogicJSON == nil || strings.TrimSpace(*f.ConditionalLogicJSON) == "" {
		return true
	}
	var cl conditionalLogic
	if json.Unmarshal([]byte(*f.ConditionalLogicJSON), &cl) != nil || len(cl.Conditions) == 0 {
		return true
	}
	matchAll := !strings.EqualFold(cl.Match, "any")
	matched := matchAll
	for _, c := range cl.Conditions {
		got := firstVal(values, c.FieldID)
		var ok bool
		switch c.Operator {
		case "eq":
			ok = got == c.Value
		case "neq":
			ok = got != c.Value
		case "contains":
			ok = strings.Contains(got, c.Value)
			if !ok {
				for _, v := range allVals(values, c.FieldID) {
					if v == c.Value {
						ok = true
						break
					}
				}
			}
		case "gt":
			ok = numLess(c.Value, got)
		case "lt":
			ok = numLess(got, c.Value)
		case "filled":
			ok = got != ""
		case "empty":
			ok = got == ""
		}
		if matchAll {
			matched = matched && ok
		} else {
			matched = matched || ok
		}
	}
	if strings.EqualFold(cl.Action, "hide") {
		return !matched
	}
	return matched
}

func numLess(a, b string) bool {
	af, aerr := strconv.ParseFloat(strings.TrimSpace(a), 64)
	bf, berr := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if aerr != nil || berr != nil {
		return false
	}
	return af < bf
}

var timeRe = regexp.MustCompile(`^\d{2}:\d{2}(:\d{2})?$`)
var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var datetimeRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}`)

// validateAnswers runs the dynamic per-field validation. fileSatisfied reports,
// per file fieldID, whether a live upload or a staged draft file covers it.
func validateAnswers(fields []dynamicform_model.Field, values map[int64][]string, fileSatisfied map[int64]bool) []apperror.FieldError {
	var errs []apperror.FieldError
	add := func(f dynamicform_model.Field, msg string) {
		errs = append(errs, apperror.FieldError{
			Attribute: "field_" + strconv.FormatInt(f.FieldID, 10),
			Field:     f.Label,
			Code:      constants.CodeValidationError,
			Message:   msg,
		})
	}

	for _, f := range fields {
		if !f.IsActive || isDisplayType(f.FieldType) {
			continue
		}
		if !fieldVisible(f, values) {
			continue
		}

		if f.FieldType == "file" {
			if f.IsRequired && !fileSatisfied[f.FieldID] {
				add(f, f.Label+" wajib diunggah")
			}
			continue
		}

		single := firstVal(values, f.FieldID)
		multi := allVals(values, f.FieldID)
		empty := single == "" && len(multi) == 0

		if f.IsRequired && empty {
			add(f, f.Label+" wajib diisi")
			continue
		}
		if empty {
			continue
		}

		var val fieldValidation
		if f.ValidationJSON != nil && *f.ValidationJSON != "" {
			_ = json.Unmarshal([]byte(*f.ValidationJSON), &val)
		}

		switch f.FieldType {
		case "email":
			if _, err := mail.ParseAddress(single); err != nil {
				add(f, f.Label+" harus berupa alamat email yang valid")
			}
		case "url":
			if u, err := url.ParseRequestURI(single); err != nil || u.Scheme == "" {
				add(f, f.Label+" harus berupa URL yang valid")
			}
		case "number", "linear_scale", "rating":
			n, err := strconv.ParseFloat(single, 64)
			if err != nil {
				add(f, f.Label+" harus berupa angka")
			} else {
				if val.Min != nil && n < float64(*val.Min) {
					add(f, f.Label+" minimal "+strconv.Itoa(*val.Min))
				}
				if val.Max != nil && n > float64(*val.Max) {
					add(f, f.Label+" maksimal "+strconv.Itoa(*val.Max))
				}
			}
		case "date":
			if !dateRe.MatchString(single) {
				add(f, f.Label+" harus berupa tanggal (YYYY-MM-DD)")
			}
		case "time":
			if !timeRe.MatchString(single) {
				add(f, f.Label+" harus berupa waktu (HH:MM)")
			}
		case "datetime":
			if !datetimeRe.MatchString(single) {
				add(f, f.Label+" harus berupa tanggal & waktu")
			}
		case "short_text", "long_text", "phone":
			if val.Min != nil && len(single) < *val.Min {
				add(f, f.Label+" minimal "+strconv.Itoa(*val.Min)+" karakter")
			}
			if val.Max != nil && len(single) > *val.Max {
				add(f, f.Label+" maksimal "+strconv.Itoa(*val.Max)+" karakter")
			}
		}

		if f.FieldType == "dropdown" || f.FieldType == "radio" || f.FieldType == "checkbox" {
			allowed := optionValues(f)
			if len(allowed) > 0 {
				check := multi
				if f.FieldType != "checkbox" {
					check = []string{single}
				}
				for _, c := range check {
					if !contains(allowed, c) {
						add(f, "Pilihan pada "+f.Label+" tidak valid")
						break
					}
				}
			}
		}

		if val.Pattern != nil && *val.Pattern != "" && f.FieldType != "checkbox" {
			if re, err := regexp.Compile(*val.Pattern); err == nil && !re.MatchString(single) {
				add(f, f.Label+" tidak sesuai format yang diminta")
			}
		}
	}
	return errs
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
