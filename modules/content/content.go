// Package content mengelola konten Landing Page (ms_cms_content) dan
// struktur organisasi (ms_organization_structure).
package content

import "database/sql"

// Content merepresentasikan satu baris ms_cms_content.
type Content struct {
	ContentID    int64          `db:"contentID" json:"contentID"`
	ContentKey   string         `db:"contentKey" json:"contentKey"`
	ContentTitle sql.NullString `db:"contentTitle" json:"contentTitle"`
	ContentBody  sql.NullString `db:"contentBody" json:"contentBody"`
	ContentType  string         `db:"contentType" json:"contentType"`
	SortOrder    sql.NullInt64  `db:"sortOrder" json:"sortOrder"`
	IsActive     bool           `db:"isActive" json:"isActive"`
}

// OrgMember merepresentasikan satu baris ms_organization_structure.
type OrgMember struct {
	StructureID int64          `db:"structureID" json:"structureID"`
	MemberName  string         `db:"memberName" json:"memberName"`
	Position    string         `db:"position" json:"position"`
	PhotoURL    sql.NullString `db:"photoURL" json:"photoURL"`
	Level       sql.NullString `db:"level" json:"level"`
	SortOrder   sql.NullInt64  `db:"sortOrder" json:"sortOrder"`
	IsActive    bool           `db:"isActive" json:"isActive"`
}

// ContentUpdateRequest adalah body memperbarui konten.
type ContentUpdateRequest struct {
	ContentTitle string `json:"contentTitle" validate:"max=255"`
	ContentBody  string `json:"contentBody"`
}

// OrgRequest adalah body membuat/memperbarui pengurus.
type OrgRequest struct {
	MemberName string `json:"memberName" validate:"required,max=150"`
	Position   string `json:"position" validate:"required,max=150"`
	PhotoURL   string `json:"photoURL" validate:"max=255"`
	Level      string `json:"level" validate:"max=50"`
	SortOrder  int    `json:"sortOrder"`
}
