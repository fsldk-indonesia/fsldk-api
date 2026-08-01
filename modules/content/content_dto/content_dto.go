// Package content_dto memuat DTO request/response modul content.
package content_dto

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
