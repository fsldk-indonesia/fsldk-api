// Package content_model memuat entitas modul content.
package content_model

// Content merepresentasikan satu baris ms_cms_content.
type Content struct {
	ContentID    int64   `db:"contentID" json:"contentID"`
	ContentKey   string  `db:"contentKey" json:"contentKey"`
	ContentTitle *string `db:"contentTitle" json:"contentTitle"`
	ContentBody  *string `db:"contentBody" json:"contentBody"`
	ContentType  string  `db:"contentType" json:"contentType"`
	SortOrder    *int    `db:"sortOrder" json:"sortOrder"`
	IsActive     bool    `db:"isActive" json:"isActive"`
}

// OrgMember merepresentasikan satu baris ms_organization_structure.
type OrgMember struct {
	StructureID int64   `db:"structureID" json:"structureID"`
	MemberName  string  `db:"memberName" json:"memberName"`
	Position    string  `db:"position" json:"position"`
	PhotoURL    *string `db:"photoURL" json:"photoURL"`
	Level       *string `db:"level" json:"level"`
	SortOrder   *int    `db:"sortOrder" json:"sortOrder"`
	IsActive    bool    `db:"isActive" json:"isActive"`
}
