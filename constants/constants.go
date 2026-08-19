// Package constants holds centralized constants used across modules:
// error codes, default role & permission names, table names, and context keys.
package constants

// Internal response codes (sent on the "code" field of the response envelope).
const (
	CodeSuccess                 = "00"
	CodeValidationError         = "40"
	CodeUnauthorized            = "41"
	CodeForbidden               = "43"
	CodeNotFound                = "44"
	CodeConflict                = "49"
	CodeUnprocessable           = "42"
	CodeTooManyRequest          = "45"
	CodeUnknownError            = "99"
	CodeEmailNotVerified        = "43-EMAIL"
	CodeDuplicateSubmission     = "49-DUP"
	CodeInvalidStatusTransition = "42-STATUS"
)

// Default system roles.
const (
	RoleSuperAdmin           = "Super Admin"
	RoleEditor               = "Editor"
	RoleKontributor          = "Kontributor"
	RoleMember               = "Member" // pendaftar publik, tanpa akses CMS — lihat modules/comment
	RoleLDKAdmin             = "LDK Admin"
	RolePuskomdaVerifikator  = "Puskomda Verifikator"
	RolePuskomnasVerifikator = "Puskomnas Verifikator"
	RoleKader                = "Kader"
)

// Kode tipe organisasi (lk_organization_type), sekaligus nilai yang valid
// pada kolom ms_user.wildcardTierAccess.
const (
	OrgTypeLDK       = "LDK"
	OrgTypePuskomda  = "PUSKOMDA"
	OrgTypePuskomnas = "PUSKOMNAS"
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

	PermShortlinkView   = "shortlink.view"
	PermShortlinkCreate = "shortlink.create"
	PermShortlinkUpdate = "shortlink.update"
	PermShortlinkDelete = "shortlink.delete"

	PermOrganizationCreate          = "organization.create"
	PermOrganizationDeactivate      = "organization.deactivate"
	PermOrganizationProfileManage   = "organization.profile.manage"
	PermOrganizationLDKList         = "organization.ldk.list"
	PermOrganizationLDKListNational = "organization.ldk.list.national"
	PermOrganizationPuskomdaList    = "organization.puskomda.list"

	PermUserProvision = "user.provision"

	PermSubmissionFormView   = "submission_form.view"
	PermSubmissionFormManage = "submission_form.manage"

	PermSubmissionCreate = "submission.create"
	PermSubmissionUpdate = "submission.update"
	PermSubmissionCancel = "submission.cancel"
	PermSubmissionView   = "submission.view"

	PermSubmissionReviewLDK      = "submission.review.ldk"
	PermSubmissionReviewTier1    = "submission.review.tier1"
	PermSubmissionApproveTier1   = "submission.approve.tier1"
	PermSubmissionReviewTier2    = "submission.review.tier2"
	PermSubmissionLevelEstablish = "submission.level.establish"
	PermSubmissionPublish        = "submission.publish"
	PermSubmissionReopen         = "submission.reopen"
	PermSubmissionReassess       = "submission.reassess"
	PermKaderDeactivate          = "kader.deactivate"

	PermReportRegionView     = "report.region.view"
	PermReportRegionExport   = "report.region.export"
	PermReportNationalView   = "report.national.view"
	PermReportNationalExport = "report.national.export"

	PermEventView   = "event.view"
	PermEventCreate = "event.create"
	PermEventUpdate = "event.update"
	PermEventDelete = "event.delete"

	PermCommentView   = "comment.view"
	PermCommentUpdate = "comment.update"
	PermCommentDelete = "comment.delete"
)

// Kode form pendataan konkret bawaan (dibangun di atas submission form engine).
const (
	FormCodeLevelisasiLDK = "LEVELISASI_LDK"
	FormCodeSensusKader   = "SENSUS_KADER"
)

// Tipe subjek pengisi submission.
const (
	SubjectTypeOrganization = "ORGANIZATION"
	SubjectTypeKader        = "KADER"
)

// Status submission (state machine Levelisasi LDK & Sensus Kader).
const (
	SubmissionStatusDraft     = "DRAFT"
	SubmissionStatusSubmitted = "SUBMITTED"
	SubmissionStatusCancelled = "CANCELLED"
	SubmissionStatusRejected  = "REJECTED"

	// Levelisasi LDK
	SubmissionStatusPuskomdaReview             = "PUSKOMDA_REVIEW"
	SubmissionStatusRevisionRequestedPuskomda  = "REVISION_REQUESTED_PUSKOMDA"
	SubmissionStatusApprovedPuskomda           = "APPROVED_PUSKOMDA"
	SubmissionStatusPuskomnasReview            = "PUSKOMNAS_REVIEW"
	SubmissionStatusRevisionRequestedPuskomnas = "REVISION_REQUESTED_PUSKOMNAS"
	SubmissionStatusApprovedPuskomnas          = "APPROVED_PUSKOMNAS"
	SubmissionStatusLevelEstablished           = "LEVEL_ESTABLISHED"
	SubmissionStatusPublished                  = "PUBLISHED"

	// Sensus Kader
	SubmissionStatusLDKReview            = "LDK_REVIEW"
	SubmissionStatusRevisionRequestedLDK = "REVISION_REQUESTED_LDK"
	SubmissionStatusApprovedLDK          = "APPROVED_LDK"
	SubmissionStatusCodeIssued           = "CODE_ISSUED"
	SubmissionStatusActive               = "ACTIVE"
)

// Status kader (ms_kader.status).
const (
	KaderStatusPending  = "PENDING"
	KaderStatusActive   = "ACTIVE"
	KaderStatusRejected = "REJECTED"
	KaderStatusInactive = "INACTIVE"
)

// Tier reviewer (tr_submission_review.tierLevel).
const (
	ReviewTierLDK       = "LDK"
	ReviewTierPuskomda  = "PUSKOMDA"
	ReviewTierPuskomnas = "PUSKOMNAS"
)

// Keputusan review (tr_submission_review.decision).
const (
	ReviewDecisionApproved          = "APPROVED"
	ReviewDecisionRevisionRequested = "REVISION_REQUESTED"
	ReviewDecisionRejected          = "REJECTED"
)

// Status version form pendataan.
const (
	FormVersionDraft     = "DRAFT"
	FormVersionPublished = "PUBLISHED"
	FormVersionArchived  = "ARCHIVED"
)

// Tipe field form pendataan.
const (
	FieldTypeText        = "TEXT"
	FieldTypeTextarea    = "TEXTAREA"
	FieldTypeNumber      = "NUMBER"
	FieldTypeDate        = "DATE"
	FieldTypeSelect      = "SELECT"
	FieldTypeMultiselect = "MULTISELECT"
	FieldTypeRadio       = "RADIO"
	FieldTypeCheckbox    = "CHECKBOX"
	FieldTypeFileDoc     = "FILE_DOCUMENT"
	FieldTypeFileImage   = "FILE_IMAGE"
)

// Database table names (convention: prefix_PascalCase).
const (
	TableUser            = "ms_user"
	TableRole            = "ms_role"
	TablePermission      = "lk_permission"
	TableRolePermission  = "map_role_permission"
	TableNews            = "ms_news"
	TableNewsCategory    = "lk_news_category"
	TableArticle         = "ms_article"
	TableArticleCategory = "lk_article_category"
	TableShortlink       = "ms_shortlink"
	TableComment         = "ms_comment"
	TableCommentReaction = "tr_comment_reaction"
	TableUserLoginLog    = "tr_user_login_log"
	TableEmailToken      = "tr_email_token" // token verifikasi email & reset password
	TableEvent           = "ms_event"

	TableOrganization     = "ms_organization"
	TableOrganizationType = "lk_organization_type"

	TableSubmissionForm            = "ms_submission_form"
	TableSubmissionFormVersion     = "ms_submission_form_version"
	TableSubmissionFormSection     = "ms_submission_form_section"
	TableSubmissionFormField       = "ms_submission_form_field"
	TableSubmissionFormFieldOption = "ms_submission_form_field_option"

	TableSubmission              = "tr_submission"
	TableSubmissionAnswer        = "tr_submission_answer"
	TableSubmissionStatusHistory = "tr_submission_status_history"
	TableKader                   = "ms_kader"

	TableSubmissionReview = "tr_submission_review"
	TableLevel            = "lk_level"
	TableLevelisasiResult = "tr_levelisasi_result"

	TableFormAuditLog = "tr_form_audit_log"
	TableUserAuditLog = "tr_user_audit_log"
	TableExportLog    = "tr_export_log"
)

// Keys stored on gin.Context by the authentication middleware.
const (
	CtxUserID               = "ctxUserID"
	CtxUserEmail            = "ctxUserEmail"
	CtxRoleID               = "ctxRoleID"
	CtxRoleName             = "ctxRoleName"
	CtxEmailVerified        = "ctxEmailVerified"
	CtxPermissions          = "ctxPermissions"
	CtxOrganizationID       = "ctxOrganizationID"
	CtxOrganizationTypeCode = "ctxOrganizationTypeCode"
	CtxWildcardTierAccess   = "ctxWildcardTierAccess"
	CtxTargetOrganizationID = "ctxTargetOrganizationID"
)

// Email token types.
const (
	EmailTokenVerification  = "verification"
	EmailTokenPasswordReset = "password_reset"
)
