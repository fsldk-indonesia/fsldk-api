// Package jobqueue_dto memuat DTO request/response modul job queue.
// Seluruhnya murni struct data (tanpa function/method).
package jobqueue_dto

// EnqueueInput adalah parameter memasukkan job baru — dipakai modul lain
// (mis. shortlinkrequest_service) lewat jobqueue_service.Enqueue.
type EnqueueInput struct {
	Queue           string      // jobqueue_model.QueueWhatsApp | QueueEmail
	JobType         string      // jobqueue_model.JobType*
	Payload         interface{} // di-marshal ke JSON oleh service
	DelaySeconds    int         // 0 = secepatnya (queue whatsapp tetap dapat jitter 0-8s tambahan)
	MaxAttempts     int         // 0 = pakai default config
	CorrelationType string      // opsional, mis. "shortlink_request"
	CorrelationID   int64
}

// Response adalah representasi job untuk API dashboard CMS.
type Response struct {
	JobID           int64  `json:"jobID"`
	Queue           string `json:"queue"`
	JobType         string `json:"jobType"`
	Status          string `json:"status"`
	Attempts        int    `json:"attempts"`
	MaxAttempts     int    `json:"maxAttempts"`
	LastError       string `json:"lastError"`
	CorrelationType string `json:"correlationType"`
	CorrelationID   int64  `json:"correlationID"`
	AvailableDate   string `json:"availableDate"`
	CreatedDate     string `json:"createdDate"`
	CompletedDate   string `json:"completedDate"`
	FailedDate      string `json:"failedDate"`
}

// StatsResponse adalah ringkasan jumlah job per bucket untuk dashboard.
type StatsResponse struct {
	Pending    int64 `json:"pending"`
	Delayed    int64 `json:"delayed"` // pending tapi availableDate masih di masa depan
	Processing int64 `json:"processing"`
	Stuck      int64 `json:"stuck"` // processing melewati ambang waktu (belum sempat di-sweep)
	Failed     int64 `json:"failed"`
	Completed  int64 `json:"completed"`
}

// ListFilter menampung parameter penyaringan daftar job.
type ListFilter struct {
	Status  string
	Queue   string
	Search  string
	Limit   int
	Offset  int
	OrderBy string
}

// ShortlinkApprovedEmailPayload adalah payload jobType=email_shortlink_approved.
type ShortlinkApprovedEmailPayload struct {
	ToEmail  string `json:"toEmail"`
	ToName   string `json:"toName"`
	ShortURL string `json:"shortURL"`
}

// ShortlinkRejectedEmailPayload adalah payload jobType=email_shortlink_rejected.
type ShortlinkRejectedEmailPayload struct {
	ToEmail string `json:"toEmail"`
	ToName  string `json:"toName"`
	Reason  string `json:"reason"`
}
