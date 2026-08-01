package user

// Response adalah representasi pengguna untuk API.
type Response struct {
	UserID        int64  `json:"userID"`
	FullName      string `json:"fullName"`
	Email         string `json:"email"`
	RoleID        int64  `json:"roleID"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"emailVerified"`
	IsActive      bool   `json:"isActive"`
	PhotoURL      string `json:"photoURL,omitempty"`
	HasGoogle     bool   `json:"hasGoogle"`
	HasPassword   bool   `json:"hasPassword"`
}

// ToResponse memetakan model User ke Response.
func ToResponse(u User) Response {
	return Response{
		UserID:        u.UserID,
		FullName:      u.FullName,
		Email:         u.Email,
		RoleID:        u.RoleID,
		Role:          u.RoleName,
		EmailVerified: u.EmailVerified(),
		IsActive:      u.IsActive,
		PhotoURL:      u.PhotoURL.String,
		HasGoogle:     u.GoogleID.Valid,
		HasPassword:   u.HasPassword(),
	}
}

// CreateRequest adalah body membuat pengguna baru (oleh admin).
type CreateRequest struct {
	FullName string `json:"fullName" validate:"required,min=3,max=150"`
	Email    string `json:"email" validate:"required,email,max=150"`
	RoleID   int64  `json:"roleID" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	IsActive *bool  `json:"isActive"`
}

// UpdateRequest adalah body memperbarui pengguna.
type UpdateRequest struct {
	FullName string `json:"fullName" validate:"required,min=3,max=150"`
	RoleID   int64  `json:"roleID" validate:"required"`
	IsActive bool   `json:"isActive"`
}

// StatusRequest adalah body mengubah status aktif pengguna.
type StatusRequest struct {
	IsActive bool `json:"isActive"`
}
