// Package submission_model memuat entitas modul submission. Murni struct data.
package submission_model

import (
	"database/sql"
	"time"
)

// Submission merepresentasikan satu baris tr_submission.
type Submission struct {
	SubmissionID       int64         `gorm:"column:submissionID;primaryKey"`
	FormID             int64         `gorm:"column:formID"`
	FormVersionID      int64         `gorm:"column:formVersionID"`
	PeriodID           sql.NullInt64 `gorm:"column:periodID"`
	OrganizationID     int64         `gorm:"column:organizationID"`
	SubjectType        string        `gorm:"column:subjectType"`
	SubjectReferenceID sql.NullInt64 `gorm:"column:subjectReferenceID"`
	SubmittedByUserID  int64         `gorm:"column:submittedByUserID"`
	Status             string        `gorm:"column:status"`
	Version            int           `gorm:"column:version"`
	SubmittedDate      sql.NullTime  `gorm:"column:submittedDate"`
	LastUpdatedDate    sql.NullTime  `gorm:"column:lastUpdatedDate"`
	CreatedDate        time.Time     `gorm:"column:createdDate"`
}

// Answer merepresentasikan satu baris tr_submission_answer.
type Answer struct {
	AnswerID      int64           `gorm:"column:answerID;primaryKey"`
	SubmissionID  int64           `gorm:"column:submissionID"`
	FieldID       int64           `gorm:"column:fieldID"`
	ValueText     sql.NullString  `gorm:"column:valueText"`
	ValueNumber   sql.NullFloat64 `gorm:"column:valueNumber"`
	ValueDate     sql.NullTime    `gorm:"column:valueDate"`
	ValueOptionID sql.NullInt64   `gorm:"column:valueOptionID"`
	ValueFileURL  sql.NullString  `gorm:"column:valueFileURL"`
	ValueFileName sql.NullString  `gorm:"column:valueFileName"`
}

// AnswerParams menampung nilai jawaban satu field untuk disimpan (create/update).
type AnswerParams struct {
	FieldID       int64
	ValueText     sql.NullString
	ValueNumber   sql.NullFloat64
	ValueDate     sql.NullTime
	ValueOptionID sql.NullInt64
	ValueFileURL  sql.NullString
	ValueFileName sql.NullString
}

// StatusHistory merepresentasikan satu baris tr_submission_status_history.
type StatusHistory struct {
	HistoryID           int64          `gorm:"column:historyID;primaryKey"`
	SubmissionID        int64          `gorm:"column:submissionID"`
	FromStatus          sql.NullString `gorm:"column:fromStatus"`
	ToStatus            string         `gorm:"column:toStatus"`
	ActorUserID         int64          `gorm:"column:actorUserID"`
	ActorOrganizationID sql.NullInt64  `gorm:"column:actorOrganizationID"`
	Note                sql.NullString `gorm:"column:note"`
	CreatedDate         time.Time      `gorm:"column:createdDate"`
}

// Kader merepresentasikan satu baris ms_kader.
type Kader struct {
	KaderID        int64          `gorm:"column:kaderID;primaryKey"`
	SubmissionID   int64          `gorm:"column:submissionID"`
	OrganizationID int64          `gorm:"column:organizationID"`
	UserID         int64          `gorm:"column:userID"`
	UniqueCode     sql.NullString `gorm:"column:uniqueCode"`
	FullName       string         `gorm:"column:fullName"`
	Status         string         `gorm:"column:status"`
	IssuedDate     sql.NullTime   `gorm:"column:issuedDate"`
	CreatedDate    time.Time      `gorm:"column:createdDate"`
}

// CreateParams menampung data untuk membuat submission baru.
type CreateParams struct {
	FormID            int64
	FormVersionID     int64
	OrganizationID    int64
	SubjectType       string
	SubmittedByUserID int64
	CreatedBy         sql.NullInt64
}

// KaderParams menampung data untuk membuat baris ms_kader.
type KaderParams struct {
	SubmissionID   int64
	OrganizationID int64
	UserID         int64
	FullName       string
}
