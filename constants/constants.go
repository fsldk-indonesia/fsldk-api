// Package constants memuat konstanta terpusat yang digunakan lintas modul:
// kode error, nama role & permission bawaan, nama tabel, dan kunci context.
package constants

// Kode response internal (dikirim pada field "code" pada envelope response).
const (
	CodeSuccess          = "00"
	CodeValidationError  = "40"
	CodeUnauthorized     = "41"
	CodeForbidden        = "43"
	CodeNotFound         = "44"
	CodeConflict         = "49"
	CodeUnprocessable    = "42"
	CodeTooManyRequest   = "45"
	CodeUnknownError     = "99"
	CodeEmailNotVerified = "43-EMAIL"
)

// Role bawaan sistem.
const (
	RoleSuperAdmin  = "Super Admin"
	RoleEditor      = "Editor"
	RoleKontributor = "Kontributor"
)

// Kode permission (format modul.aksi).
const (
	PermNewsView    = "news.view"
	PermNewsCreate  = "news.create"
	PermNewsUpdate  = "news.update"
	PermNewsDelete  = "news.delete"
	PermNewsPublish = "news.publish"

	PermArticleView    = "article.view"
	PermArticleCreate  = "article.create"
	PermArticleUpdate  = "article.update"
	PermArticleDelete  = "article.delete"
	PermArticlePublish = "article.publish"

	PermUserView   = "user.view"
	PermUserCreate = "user.create"
	PermUserUpdate = "user.update"
	PermUserDelete = "user.delete"

	PermRoleView   = "role.view"
	PermRoleCreate = "role.create"
	PermRoleUpdate = "role.update"
	PermRoleDelete = "role.delete"

	PermContentView   = "content.view"
	PermContentUpdate = "content.update"
)

// Nama tabel database (konvensi prefix_snake_case).
const (
	TableUser                = "ms_user"
	TableRole                = "ms_role"
	TablePermission          = "lk_permission"
	TableRolePermission      = "map_role_permission"
	TableNews                = "ms_news"
	TableNewsCategory        = "lk_news_category"
	TableArticle             = "ms_article"
	TableArticleCategory     = "lk_article_category"
	TableCMSContent          = "ms_cms_content"
	TableOrganizationStruct  = "ms_organization_structure"
	TableUserLoginLog        = "tr_user_login_log"
	TableEmailToken          = "tr_email_token" // token verifikasi email & reset password
)

// Kunci yang disimpan pada gin.Context oleh middleware autentikasi.
const (
	CtxUserID        = "ctxUserID"
	CtxUserEmail     = "ctxUserEmail"
	CtxRoleID        = "ctxRoleID"
	CtxRoleName      = "ctxRoleName"
	CtxEmailVerified = "ctxEmailVerified"
	CtxPermissions   = "ctxPermissions"
)

// Jenis token email.
const (
	EmailTokenVerification = "verification"
	EmailTokenPasswordReset = "password_reset"
)
