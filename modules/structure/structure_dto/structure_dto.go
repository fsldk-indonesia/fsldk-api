package structure_dto

// CreateRequest is the body for creating a structure.
type CreateRequest struct {
	Batch                string `json:"batch" validate:"required,max=50"`
	Period               string `json:"period" validate:"required,max=50"`
	StructureName        string `json:"structureName" validate:"required,max=255"`
	StructureDescription string `json:"structureDescription" validate:"required"`
	LogoImage            string `json:"logoImage" validate:"required,max=255"`
	StructureImage       string `json:"structureImage" validate:"required,max=255"`
}

// UpdateRequest is the body for updating a structure.
type UpdateRequest struct {
	Batch                string  `json:"batch" validate:"required,max=50"`
	Period               string  `json:"period" validate:"required,max=50"`
	StructureName        string  `json:"structureName" validate:"required,max=255"`
	StructureDescription string  `json:"structureDescription" validate:"required"`
	LogoImage            *string `json:"logoImage" validate:"omitempty,max=255"`
	StructureImage       *string `json:"structureImage" validate:"omitempty,max=255"`
}

// Filter holds query parameters for listing structures (used by repository & service).
type Filter struct {
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}
