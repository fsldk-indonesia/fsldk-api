// Package validation membungkus go-playground/validator dan menerjemahkan
// error validasi ke format FieldError standar aplikasi.
package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

// shortlinkKeyPattern mengizinkan huruf, angka, dan tanda hubung — dipakai
// tag kustom `shortlinkkey` (bawaan `alphanum` terlalu ketat, tidak
// mengizinkan tanda hubung yang lazim dipakai sebagai kunci shortlink).
var shortlinkKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// codeIdentifierPattern mengizinkan huruf, angka, dan garis bawah — dipakai
// tag kustom `codeidentifier` untuk kode form/section/field (mis. LEVELISASI_LDK).
var codeIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func init() {
	validate = validator.New()
	// Gunakan tag `json` sebagai nama field pada pesan error.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
	_ = validate.RegisterValidation("shortlinkkey", func(fl validator.FieldLevel) bool {
		return shortlinkKeyPattern.MatchString(fl.Field().String())
	})
	_ = validate.RegisterValidation("codeidentifier", func(fl validator.FieldLevel) bool {
		return codeIdentifierPattern.MatchString(fl.Field().String())
	})
}

// Struct memvalidasi sebuah struct. Mengembalikan *apperror.AppError bila gagal.
func Struct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var verrs validator.ValidationErrors
	if !reflectAs(err, &verrs) {
		return apperror.BadRequest("Data tidak valid")
	}

	fields := make([]apperror.FieldError, 0, len(verrs))
	for _, fe := range verrs {
		fields = append(fields, apperror.FieldError{
			Attribute: fe.Tag(),
			Field:     fe.Field(),
			Code:      constants.CodeValidationError,
			Message:   humanMessage(fe),
		})
	}
	return apperror.Validation("Validation Error", fields)
}

func reflectAs(err error, target *validator.ValidationErrors) bool {
	if v, ok := err.(validator.ValidationErrors); ok {
		*target = v
		return true
	}
	return false
}

func humanMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", field)
	case "email":
		return fmt.Sprintf("%s harus berupa alamat email yang valid", field)
	case "min":
		return fmt.Sprintf("%s minimal %s karakter", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s maksimal %s karakter", field, fe.Param())
	case "eqfield":
		return fmt.Sprintf("%s tidak cocok", field)
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari: %s", field, fe.Param())
	case "url":
		return fmt.Sprintf("%s harus berupa URL yang valid", field)
	case "alphanum":
		return fmt.Sprintf("%s hanya boleh berisi huruf dan angka", field)
	case "shortlinkkey":
		return fmt.Sprintf("%s hanya boleh berisi huruf, angka, dan tanda hubung (-)", field)
	case "codeidentifier":
		return fmt.Sprintf("%s hanya boleh berisi huruf, angka, dan garis bawah (_)", field)
	default:
		return fmt.Sprintf("%s tidak valid", field)
	}
}
