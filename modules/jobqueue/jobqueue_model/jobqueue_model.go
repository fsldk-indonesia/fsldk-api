// Package jobqueue_model memuat entitas modul job queue (antrian pengiriman
// asinkron — WhatsApp/email — dengan retry, §1b techspec). Seluruhnya murni
// struct data (tanpa function/method).
package jobqueue_model

import "time"

// Job merepresentasikan satu baris tr_job_queue.
type Job struct {
	JobID           int64      `gorm:"column:jobID;primaryKey"`
	Queue           string     `gorm:"column:queue"`
	JobType         string     `gorm:"column:jobType"`
	Payload         string     `gorm:"column:payload"`
	Status          string     `gorm:"column:status"`
	Attempts        int        `gorm:"column:attempts"`
	MaxAttempts     int        `gorm:"column:maxAttempts"`
	AvailableDate   time.Time  `gorm:"column:availableDate"`
	ReservedDate    *time.Time `gorm:"column:reservedDate"`
	LastError       *string    `gorm:"column:lastError"`
	CorrelationType *string    `gorm:"column:correlationType"`
	CorrelationID   *int64     `gorm:"column:correlationID"`
	CreatedDate     time.Time  `gorm:"column:createdDate"`
	CompletedDate   *time.Time `gorm:"column:completedDate"`
	FailedDate      *time.Time `gorm:"column:failedDate"`
}

// Queue, jobType, status & correlationType konstan — 1 sumber kebenaran
// dipakai baik oleh producer (mis. shortlinkrequest_service) maupun worker.
const (
	QueueWhatsApp = "whatsapp"
	QueueEmail    = "email"
	// QueueDefault carries generic background jobs handled by callbacks other
	// modules register via jobqueue_service.RegisterHandler (e.g. the
	// dynamicform Google Sheets mirror). It has no rate limiter.
	QueueDefault = "default"

	JobTypeWhatsAppTemplate       = "whatsapp_template"
	JobTypeEmailShortlinkApproved = "email_shortlink_approved"
	JobTypeEmailShortlinkRejected = "email_shortlink_rejected"

	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"

	CorrelationTypeShortlinkRequest = "shortlink_request"

	// Kantong Amal (Phase 8) — notifikasi WhatsApp donasi/campaign/withdrawal.
	CorrelationTypeDonation   = "donation"
	CorrelationTypeCampaign   = "campaign"
	CorrelationTypeWithdrawal = "withdrawal"
)
