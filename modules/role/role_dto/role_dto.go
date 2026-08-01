// Package role_dto memuat DTO request/response modul role.
package role_dto

// Response adalah representasi role untuk API.
type Response struct {
	RoleID          int64    `json:"roleID"`
	RoleName        string   `json:"roleName"`
	RoleDescription string   `json:"roleDescription"`
	IsSystemRole    bool     `json:"isSystemRole"`
	IsActive        bool     `json:"isActive"`
	UserCount       int      `json:"userCount"`
	Permissions     []string `json:"permissions"`
	PermissionIDs   []int64  `json:"permissionIDs"`
}

// CreateRequest adalah body membuat role.
type CreateRequest struct {
	RoleName        string `json:"roleName" validate:"required,min=2,max=100"`
	RoleDescription string `json:"roleDescription" validate:"max=255"`
}

// UpdateRequest adalah body memperbarui role.
type UpdateRequest struct {
	RoleName        string `json:"roleName" validate:"required,min=2,max=100"`
	RoleDescription string `json:"roleDescription" validate:"max=255"`
	IsActive        bool   `json:"isActive"`
}

// SetPermissionsRequest adalah body menetapkan permission ke role.
type SetPermissionsRequest struct {
	PermissionIDs []int64 `json:"permissionIDs"`
}
