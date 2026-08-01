// Package content_model memuat entitas modul content. Murni struct data.
package content_model

// Content merepresentasikan satu baris ms_cms_content.
type Content struct {
	ContentID    int64   `gorm:"column:contentID;primaryKey" json:"contentID"`
	ContentKey   string  `gorm:"column:contentKey" json:"contentKey"`
	ContentTitle *string `gorm:"column:contentTitle" json:"contentTitle"`
	ContentBody  *string `gorm:"column:contentBody" json:"contentBody"`
	ContentType  string  `gorm:"column:contentType" json:"contentType"`
	SortOrder    *int    `gorm:"column:sortOrder" json:"sortOrder"`
	IsActive     bool    `gorm:"column:isActive" json:"isActive"`
}

// OrgMember merepresentasikan satu baris ms_organization_structure.
type OrgMember struct {
	StructureID int64   `gorm:"column:structureID;primaryKey" json:"structureID"`
	MemberName  string  `gorm:"column:memberName" json:"memberName"`
	Position    string  `gorm:"column:position" json:"position"`
	PhotoURL    *string `gorm:"column:photoURL" json:"photoURL"`
	Level       *string `gorm:"column:level" json:"level"`
	SortOrder   *int    `gorm:"column:sortOrder" json:"sortOrder"`
	IsActive    bool    `gorm:"column:isActive" json:"isActive"`
}
