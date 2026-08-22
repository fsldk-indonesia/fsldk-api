// Package auditlog menulis baris audit trail generik (Section 29) ke salah
// satu dari tiga tabel log (form/user/export). Dipakai oleh service lain yang
// perlu mencatat perubahan signifikan — tidak menambah lapisan repository
// tersendiri karena penulisannya seragam & tidak memerlukan query baca balik.
package auditlog

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Entry menampung satu baris audit trail.
type Entry struct {
	ActorUserID         int64
	ActorOrganizationID *int64
	Action              string
	Entity              string
	EntityID            int64
	Before              interface{}
	After               interface{}
	IPAddress           string
	Metadata            interface{}
}

// Logger menulis Entry ke tabel audit tertentu.
type Logger struct{ db *gorm.DB }

// New membuat Logger.
func New(db *gorm.DB) *Logger { return &Logger{db: db} }

func toJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(b)
}

func (l *Logger) write(ctx context.Context, table string, e Entry) {
	_ = l.db.WithContext(ctx).Table(table).Create(map[string]interface{}{
		"actorUserID":         e.ActorUserID,
		"actorOrganizationID": e.ActorOrganizationID,
		"action":              e.Action,
		"entity":              e.Entity,
		"entityID":            e.EntityID,
		"beforeJSON":          toJSON(e.Before),
		"afterJSON":           toJSON(e.After),
		"ipAddress":           e.IPAddress,
		"metadata":            toJSON(e.Metadata),
		"createdDate":         time.Now(),
	}).Error
}

// LogForm mencatat perubahan struktur/publish form pendataan (tr_form_audit_log).
func (l *Logger) LogForm(ctx context.Context, e Entry) { l.write(ctx, "tr_form_audit_log", e) }

// LogUser mencatat perubahan organisasi/role/permission pada akun (tr_user_audit_log).
func (l *Logger) LogUser(ctx context.Context, e Entry) { l.write(ctx, "tr_user_audit_log", e) }

// LogExport mencatat aktivitas ekspor data (tr_export_log).
func (l *Logger) LogExport(ctx context.Context, e Entry) { l.write(ctx, "tr_export_log", e) }

// LogFinance mencatat aksi finansial Kantong Amal — campaign, donation,
// wallet, withdrawal (tr_finance_audit_log). Sama seperti LogForm/LogUser/
// LogExport, kegagalan insert tidak pernah menggagalkan aksi bisnis di
// atasnya (error diserap oleh write()).
func (l *Logger) LogFinance(ctx context.Context, e Entry) { l.write(ctx, "tr_finance_audit_log", e) }
