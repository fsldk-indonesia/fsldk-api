// Package submission_dto memuat DTO request/response modul submission.
package submission_dto

import (
	"encoding/json"
	"time"
)

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

// ReviewRequest adalah body memverifikasi submission (tier LDK/Puskomda/Puskomnas
// diresolusi otomatis dari organizationTypeCode pemanggil, bukan dari body).
type ReviewRequest struct {
	Decision  string          `json:"decision" validate:"required,oneof=APPROVED REVISION_REQUESTED REJECTED"`
	Note      string          `json:"note" validate:"max=1000"`
	Checklist json.RawMessage `json:"checklist"`
	Version   int             `json:"version" validate:"required"`
}

// EstablishLevelRequest adalah body penetapan level oleh Puskomnas.
type EstablishLevelRequest struct {
	LevelCode         string `json:"levelCode" validate:"required"`
	JustificationNote string `json:"justificationNote" validate:"max=2000"`
	Version           int    `json:"version" validate:"required"`
}

// VersionedRequest adalah body generik untuk aksi yang hanya butuh optimistic
// lock (publish, reassess).
type VersionedRequest struct {
	Version int `json:"version" validate:"required"`
}

// ReopenRequest adalah body membuka kembali submission PUBLISHED untuk koreksi.
type ReopenRequest struct {
	Reason  string `json:"reason" validate:"required,max=1000"`
	Version int    `json:"version" validate:"required"`
}

// FieldScoreInput adalah satu skor manual (Puskomnas) untuk satu field pada
// body PUT /submissions/:id/scores. RawScore sengaja tidak diberi tag
// `required` — 0 adalah nilai skor yang sah pada skala tertentu (mis. skala
// custom 0/3/7/10); rentang valid divalidasi di service terhadap
// minScore/maxScore field, bukan lewat validator generik.
type FieldScoreInput struct {
	FieldID  int64   `json:"fieldID" validate:"required"`
	RawScore float64 `json:"rawScore"`
}

// SaveFieldScoresRequest adalah body menyimpan skor manual (dapat dipanggil
// berkali-kali, hanya untuk field UseScoring bertipe MANUAL — lihat
// submission_service.SaveFieldScores).
type SaveFieldScoresRequest struct {
	Scores []FieldScoreInput `json:"scores" validate:"required,min=1,dive"`
}

// LevelResultResponse adalah representasi hasil levelisasi untuk API.
type LevelResultResponse struct {
	ResultID          int64      `json:"resultID"`
	LevelCode         string     `json:"levelCode"`
	LevelLabel        string     `json:"levelLabel,omitempty"`
	JustificationNote string     `json:"justificationNote,omitempty"`
	EstablishedDate   time.Time  `json:"establishedDate"`
	IsPublished       bool       `json:"isPublished"`
	PublishedDate     *time.Time `json:"publishedDate,omitempty"`
}

// KaderResponse adalah representasi data kader untuk API. OrganizationName/
// ParentOrganizationName hanya diisi saat status ACTIVE (disetujui) — dipakai
// Profil Saya Portal Kader menampilkan Nama LDK & Nama Puskomda pembina.
type KaderResponse struct {
	KaderID                int64      `json:"kaderID"`
	SubmissionID           int64      `json:"submissionID"`
	OrganizationID         int64      `json:"organizationID"`
	OrganizationName       string     `json:"organizationName,omitempty"`
	ParentOrganizationName string     `json:"parentOrganizationName,omitempty"`
	UniqueCode             string     `json:"uniqueCode,omitempty"`
	FullName               string     `json:"fullName"`
	Status                 string     `json:"status"`
	IssuedDate             *time.Time `json:"issuedDate,omitempty"`
}

// ListFilter menampung parameter penyaringan daftar submission.
type ListFilter struct {
	OrganizationIDs   []int64
	SubmittedByUserID *int64
	FormID            int64
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

// FieldScoreResponse adalah breakdown skor satu field UseScoring — dipakai
// untuk audit/troubleshooting (enhancement Flexible Scoring §11: sistem
// harus bisa membedakan Raw/Maximum/Normalized/Weight/Weighted Score/Source).
type FieldScoreResponse struct {
	FieldID       int64   `json:"fieldID"`
	FieldCode     string  `json:"fieldCode"`
	FieldLabel    string  `json:"fieldLabel"`
	HasScore      bool    `json:"hasScore"`
	RawScore      float64 `json:"rawScore,omitempty"`
	MaxScore      float64 `json:"maxScore"`
	Normalized    float64 `json:"normalized,omitempty"`
	Weight        float64 `json:"weight"`
	WeightedScore float64 `json:"weightedScore,omitempty"`
	Source        string  `json:"source"` // "AUTOMATIC" | "MANUAL"
}

// ConsolidatedScoreResponse adalah hasil konsolidasi seluruh field UseScoring
// pada satu submission — murni informatif (tidak memengaruhi EstablishLevel),
// dikirim ke Puskomnas & LDK pemilik submission saja, TIDAK ke Puskomda
// (lihat submission_service.Get).
type ConsolidatedScoreResponse struct {
	Fields     []FieldScoreResponse `json:"fields"`
	FinalScore float64              `json:"finalScore"`
	// IsComplete true hanya bila SELURUH field UseScoring pada version ini
	// sudah punya raw score (otomatis terjawab atau manual diberikan) — FE
	// harus menganggap FinalScore belum final selama false.
	IsComplete bool `json:"isComplete"`
}

// DetailResponse adalah submission beserta jawaban & riwayat statusnya.
type DetailResponse struct {
	Response
	Answers          []AnswerResponse           `json:"answers"`
	StatusHistory    []StatusHistoryResponse    `json:"statusHistory"`
	LevelResult      *LevelResultResponse       `json:"levelResult,omitempty"`
	Kader            *KaderResponse             `json:"kader,omitempty"`
	ConsolidatedScore *ConsolidatedScoreResponse `json:"consolidatedScore,omitempty"`
}
