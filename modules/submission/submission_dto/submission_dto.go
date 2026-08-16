// Package submission_dto memuat DTO request/response modul submission.
package submission_dto

import "time"

// ---------- Request ----------

// CreateRequest adalah body membuat submission baru.
// OrganizationID wajib diisi untuk form ber-subjek KADER (memilih LDK tujuan);
// diabaikan/dikunci ke organisasi pemanggil untuk form ber-subjek ORGANIZATION.
type CreateRequest struct {
	FormCode       string `json:"formCode" validate:"required"`
	OrganizationID *int64 `json:"organizationID"`
}

// AnswerInput adalah satu jawaban field pada body simpan jawaban.
// Hanya satu dari ValueText/ValueNumber/ValueDate/ValueOptionID/ValueOptionIDs/
// ValueFileURL yang relevan, ditentukan oleh fieldType field bersangkutan —
// nilai yang tidak sesuai tipe field diabaikan di lapisan service.
type AnswerInput struct {
	FieldID        int64    `json:"fieldID" validate:"required"`
	ValueText      *string  `json:"valueText,omitempty"`
	ValueNumber    *float64 `json:"valueNumber,omitempty"`
	ValueDate      *string  `json:"valueDate,omitempty"`
	ValueOptionID  *int64   `json:"valueOptionID,omitempty"`
	ValueOptionIDs []int64  `json:"valueOptionIDs,omitempty"`
	ValueFileURL   *string  `json:"valueFileURL,omitempty"`
	ValueFileName  *string  `json:"valueFileName,omitempty"`
}

// SaveAnswersRequest adalah body menyimpan draft jawaban (dapat dipanggil berkali-kali).
type SaveAnswersRequest struct {
	Answers []AnswerInput `json:"answers" validate:"required,min=1,dive"`
}

// ListFilter menampung parameter penyaringan daftar submission.
type ListFilter struct {
	OrganizationIDs   []int64
	SubmittedByUserID *int64
	Status            string
	Limit             int
	Offset            int
	OrderBy           string
}

// ---------- Response ----------

// AnswerResponse adalah representasi satu jawaban field untuk API.
type AnswerResponse struct {
	FieldID        int64    `json:"fieldID"`
	FieldCode      string   `json:"fieldCode"`
	ValueText      string   `json:"valueText,omitempty"`
	ValueNumber    *float64 `json:"valueNumber,omitempty"`
	ValueDate      string   `json:"valueDate,omitempty"`
	ValueOptionID  *int64   `json:"valueOptionID,omitempty"`
	ValueOptionIDs []int64  `json:"valueOptionIDs,omitempty"`
	ValueFileURL   string   `json:"valueFileURL,omitempty"`
	ValueFileName  string   `json:"valueFileName,omitempty"`
}

// StatusHistoryResponse adalah representasi satu entri riwayat status.
type StatusHistoryResponse struct {
	FromStatus  string    `json:"fromStatus,omitempty"`
	ToStatus    string    `json:"toStatus"`
	ActorUserID int64     `json:"actorUserID"`
	Note        string    `json:"note,omitempty"`
	CreatedDate time.Time `json:"createdDate"`
}

// Response adalah representasi ringkas submission untuk API.
type Response struct {
	SubmissionID   int64      `json:"submissionID"`
	FormID         int64      `json:"formID"`
	FormCode       string     `json:"formCode"`
	FormVersionID  int64      `json:"formVersionID"`
	OrganizationID int64      `json:"organizationID"`
	SubjectType    string     `json:"subjectType"`
	Status         string     `json:"status"`
	Version        int        `json:"version"`
	SubmittedDate  *time.Time `json:"submittedDate,omitempty"`
	CreatedDate    time.Time  `json:"createdDate"`
}

// DetailResponse adalah submission beserta jawaban & riwayat statusnya.
type DetailResponse struct {
	Response
	Answers       []AnswerResponse        `json:"answers"`
	StatusHistory []StatusHistoryResponse `json:"statusHistory"`
}
