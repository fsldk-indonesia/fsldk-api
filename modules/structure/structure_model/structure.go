package structure_model

import "time"

// Structure represents a single row in ms_structure.
type Structure struct {
	StructureID          int64      `gorm:"column:structureID;primaryKey;autoIncrement" json:"structureID"`
	Batch                string     `gorm:"column:batch" json:"batch"`
	Period               string     `gorm:"column:period" json:"period"`
	StructureName        string     `gorm:"column:structureName" json:"structureName"`
	StructureDescription string     `gorm:"column:structureDescription" json:"structureDescription"`
	LogoImage            *string    `gorm:"column:logoImage" json:"logoImage"`
	StructureImage       *string    `gorm:"column:structureImage" json:"structureImage"`
	CreatedDate          time.Time  `gorm:"column:createdDate;autoCreateTime" json:"createdDate"`
	CreatedBy            *int64     `gorm:"column:createdBy" json:"createdBy,omitempty"`
	UpdatedDate          *time.Time `gorm:"column:updatedDate;autoUpdateTime" json:"updatedDate,omitempty"`
	UpdatedBy            *int64     `gorm:"column:updatedBy" json:"updatedBy,omitempty"`
}

func (Structure) TableName() string { return "ms_structure" }
