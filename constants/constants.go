// Package constants holds centralized constants used across modules:
// error codes, default role & permission names, table names, and context keys.
package constants

// Internal response codes (sent on the "code" field of the response envelope).
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

// Default system roles.
const (
	RoleSuperAdmin  = "Super Admin"
	RoleEditor      = "Editor"
	RoleKontributor = "Kontributor"
	RoleMember      = "Member" // pendaftar publik, tanpa akses CMS — lihat modules/comment
)

// Permission codes (format: module.action).
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

	PermShortlinkView    = "shortlink.view"
	PermShortlinkCreate  = "shortlink.create"
	PermShortlinkUpdate  = "shortlink.update"
	PermShortlinkDelete  = "shortlink.delete"
	PermShortlinkApprove = "shortlink.approve"

	PermEventView   = "event.view"
	PermEventCreate = "event.create"
	PermEventUpdate = "event.update"
	PermEventDelete = "event.delete"

	PermCommentView   = "comment.view"
	PermCommentUpdate = "comment.update"
	PermCommentDelete = "comment.delete"

	PermSettingView   = "setting.view"
	PermSettingUpdate = "setting.update"

	PermJobQueueView   = "jobqueue.view"
	PermJobQueueRetry  = "jobqueue.retry"
	PermJobQueueDelete = "jobqueue.delete"
)

// Database table names (convention: prefix_PascalCase).
const (
	TableUser               = "ms_user"
	TableRole               = "ms_role"
	TablePermission         = "lk_permission"
	TableRolePermission     = "map_role_permission"
	TableNews               = "ms_news"
	TableNewsCategory       = "lk_news_category"
	TableArticle            = "ms_article"
	TableArticleCategory    = "lk_article_category"
	TableShortlink          = "ms_shortlink"
	TableShortlinkRequest   = "ms_shortlink_request"
	TableComment            = "ms_comment"
	TableCommentReaction    = "tr_comment_reaction"
	TableUserLoginLog       = "tr_user_login_log"
	TableEmailToken         = "tr_email_token" // email verification & password reset token
	TableEvent              = "ms_event"
	TableSetting            = "ms_setting"
	TableJobQueue           = "tr_job_queue"
	TableWhatsAppMessageLog = "tr_whatsapp_message_log"
)

// Keys stored on gin.Context by the authentication middleware.
const (
	CtxUserID        = "ctxUserID"
	CtxUserEmail     = "ctxUserEmail"
	CtxRoleID        = "ctxRoleID"
	CtxRoleName      = "ctxRoleName"
	CtxEmailVerified = "ctxEmailVerified"
	CtxPermissions   = "ctxPermissions"
)

// Email token types.
const (
	EmailTokenVerification  = "verification"
	EmailTokenPasswordReset = "password_reset"
)
