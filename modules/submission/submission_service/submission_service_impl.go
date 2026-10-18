package submission_service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/modules/jobqueue/jobqueue_service"
	"fsldk-api/modules/organization/organization_repository"
	"fsldk-api/modules/role/role_repository"
	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission/submission_repository"
	"fsldk-api/modules/submission_form/submission_form_model"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/modules/submission_form/submission_form_service"
	"fsldk-api/modules/user/user_repository"
)

// roleNamePengunjung/roleNameKader adalah nama role sistem (bukan kode) —
// dipakai untuk transisi otomatis akun pendaftar Sensus Kader begitu
// disetujui LDK (lihat promoteToKaderRole), sama seperti nilai literal yang
// sudah dipakai migrations/0002_seed.up.sql & 0005_organization_access.up.sql.
const (
	roleNamePengunjung = "Pengunjung"
	roleNameKader      = "Kader"
)

// pengurusRolesByTier memetakan organizationTypeCode caller ke daftar role
// yang boleh ditetapkan lewat fitur Pengurus (promosi/demosi kader aktif
// dalam scope organisasi caller sendiri) — index menaik = rank lebih
// tinggi. Dipakai pengurusRoleAllowed (validasi UpdatePengurusRole) dan
// TIDAK dipakai arah sebaliknya oleh DeactivateKader (yang selalu
// menurunkan ke Kader tanpa pengecekan allow-list, karena itu aksi sistem
// otomatis, bukan permintaan caller).
var pengurusRolesByTier = map[string][]string{
	constants.OrgTypeLDK:       {roleNameKader, constants.RoleLDKAdmin},
	constants.OrgTypePuskomda:  {roleNameKader, constants.RoleLDKAdmin, constants.RolePuskomdaVerifikator},
	constants.OrgTypePuskomnas: {roleNameKader, constants.RoleLDKAdmin, constants.RolePuskomdaVerifikator, constants.RolePuskomnasVerifikator},
}

func pengurusRoleAllowed(callerOrgType, roleName string) bool {
	for _, r := range pengurusRolesByTier[callerOrgType] {
		if r == roleName {
			return true
		}
	}
	return false
}

var sortColumns = map[string]string{
	"createdDate":      "s.createdDate",
	"submittedDate":    "s.submittedDate",
	"status":           "s.status",
	"organizationName": "o.organizationName",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo        submission_repository.Repository
	formRepo    submission_form_repository.Repository
	orgRepo     organization_repository.Repository
	userRepo    user_repository.Repository
	roleRepo    role_repository.Repository
	orgScope    OrgScopeResolver
	jobqueueSvc jobqueue_service.Service
}

// NewService membuat Service submission. jobqueueSvc dipakai EnqueueCascade
// memasukkan job reset/reassessment massal (lihat HandlePublishCascadeJob)
// saat submission_form_service mem-publish version baru.
func NewService(
	repo submission_repository.Repository,
	formRepo submission_form_repository.Repository,
	orgRepo organization_repository.Repository,
	userRepo user_repository.Repository,
	roleRepo role_repository.Repository,
	orgScope OrgScopeResolver,
	jobqueueSvc jobqueue_service.Service,
) Service {
	return &ServiceImpl{repo: repo, formRepo: formRepo, orgRepo: orgRepo, userRepo: userRepo, roleRepo: roleRepo, orgScope: orgScope, jobqueueSvc: jobqueueSvc}
}

// promoteToKaderRole mengubah role akun pendaftar Sensus Kader dari
// Pengunjung menjadi Kader begitu pendaftarannya disetujui LDK (dipanggil
// dari issueKaderCode). Hanya akun yang masih ber-role Pengunjung yang
// disentuh — staf LDK/Puskomda/Puskomnas yang kebetulan juga mendaftar
// sebagai kader TIDAK didowngrade rolenya (Section 19.1: role menjawab
// "boleh ngapain", tidak seharusnya berubah diam-diam akibat aksi pihak lain).
// Kegagalan di sini tidak membatalkan persetujuan kader itu sendiri (kode
// kader sudah terlanjur terbit) — dicatat, bukan menggagalkan transaksi.
func (s *ServiceImpl) promoteToKaderRole(ctx context.Context, userID int64) error {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.RoleName != roleNamePengunjung {
		return nil
	}
	kaderRoleID, err := s.roleRepo.IDByName(ctx, roleNameKader)
	if err != nil {
		return err
	}
	return s.userRepo.Update(ctx, u.UserID, u.FullName, u.Email, kaderRoleID, u.IsActive, u.OrganizationID, u.WildcardTierAccess, userID)
}

// ---------- validation rule & conditional rule ----------

type fieldValidationRule struct {
	Regex     *string  `json:"regex"`
	Min       *float64 `json:"min"`
	Max       *float64 `json:"max"`
	MinLength *int     `json:"minLength"`
	MaxLength *int     `json:"maxLength"`
}

func parseValidationRule(raw sql.NullString) fieldValidationRule {
	var r fieldValidationRule
	if !raw.Valid || raw.String == "" {
		return r
	}
	_ = json.Unmarshal([]byte(raw.String), &r)
	return r
}

func (r fieldValidationRule) validateText(v string) error {
	if v == "" {
		return nil
	}
	if r.MinLength != nil && len(v) < *r.MinLength {
		return fmt.Errorf("panjang minimal %d karakter", *r.MinLength)
	}
	if r.MaxLength != nil && len(v) > *r.MaxLength {
		return fmt.Errorf("panjang maksimal %d karakter", *r.MaxLength)
	}
	if r.Regex != nil && *r.Regex != "" {
		if re, err := regexp.Compile(*r.Regex); err == nil && !re.MatchString(v) {
			return fmt.Errorf("format tidak sesuai")
		}
	}
	return nil
}

func (r fieldValidationRule) validateNumber(v float64) error {
	if r.Min != nil && v < *r.Min {
		return fmt.Errorf("nilai minimal %v", *r.Min)
	}
	if r.Max != nil && v > *r.Max {
		return fmt.Errorf("nilai maksimal %v", *r.Max)
	}
	return nil
}

type conditionalRule struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// evaluateConditional menentukan apakah field kondisional "aktif" (wajib
// dipertimbangkan) berdasarkan jawaban field acuan. Tanpa aturan tersimpan
// dianggap selalu aktif (tidak ada mekanisme show/hide).
func evaluateConditional(ruleJSON sql.NullString, triggerAnswer submission_model.Answer, hasTriggerAnswer bool, optionValueByID map[int64]string) bool {
	if !ruleJSON.Valid || ruleJSON.String == "" {
		return true
	}
	var rule conditionalRule
	if err := json.Unmarshal([]byte(ruleJSON.String), &rule); err != nil {
		return true
	}
	if !hasTriggerAnswer {
		return false
	}
	actual := answerAsString(triggerAnswer, optionValueByID)
	switch rule.Operator {
	case "equals":
		return actual == rule.Value
	case "notEquals":
		return actual != rule.Value
	default:
		return true
	}
}

func answerAsString(a submission_model.Answer, optionValueByID map[int64]string) string {
	switch {
	case a.ValueOptionID.Valid:
		return optionValueByID[a.ValueOptionID.Int64]
	case a.ValueText.Valid:
		return a.ValueText.String
	case a.ValueNumber.Valid:
		return strconv.FormatFloat(a.ValueNumber.Float64, 'f', -1, 64)
	case a.ValueDate.Valid:
		return a.ValueDate.Time.Format("2006-01-02")
	default:
		return ""
	}
}

func hasValue(a submission_model.Answer) bool {
	return (a.ValueText.Valid && a.ValueText.String != "") ||
		a.ValueNumber.Valid ||
		a.ValueDate.Valid ||
		a.ValueOptionID.Valid ||
		(a.ValueFileURL.Valid && a.ValueFileURL.String != "")
}

func optionExists(options []submission_form_model.Option, id int64) bool {
	for _, o := range options {
		if o.OptionID == id {
			return true
		}
	}
	return false
}

func subjectTypeForForm(formCode string) string {
	if formCode == constants.FormCodeSensusKader {
		return constants.SubjectTypeKader
	}
	return constants.SubjectTypeOrganization
}

func containsTier(wildcardTierAccess, tier string) bool {
	for _, t := range strings.Split(wildcardTierAccess, ",") {
		if t == tier {
			return true
		}
	}
	return false
}

// editableStatuses adalah status yang mengizinkan pemilik data mengisi/mengubah
// jawaban dan mengirim ulang (DRAFT awal, atau salah satu status revisi).
var editableStatuses = map[string]bool{
	constants.SubmissionStatusDraft:                      true,
	constants.SubmissionStatusRevisionRequestedPuskomda:  true,
	constants.SubmissionStatusRevisionRequestedPuskomnas: true,
	constants.SubmissionStatusRevisionRequestedLDK:       true,
}

// cascadeEligibleStatuses adalah seluruh status yang ikut diproses saat
// publish cascade (reset/reassessment massal, lihat HandlePublishCascadeJob)
// — DRAFT/CANCELLED/REJECTED sengaja TIDAK disertakan (TechSpec DL-2).
// Diekspresikan sebagai daftar INCLUDE, bukan exclude, supaya status baru
// yang ditambah nanti tidak diam-diam ikut tercakup tanpa sengaja.
var cascadeEligibleStatuses = []string{
	constants.SubmissionStatusSubmitted,
	constants.SubmissionStatusPuskomdaReview, constants.SubmissionStatusRevisionRequestedPuskomda, constants.SubmissionStatusApprovedPuskomda,
	constants.SubmissionStatusPuskomnasReview, constants.SubmissionStatusRevisionRequestedPuskomnas, constants.SubmissionStatusApprovedPuskomnas,
	constants.SubmissionStatusLevelEstablished, constants.SubmissionStatusPublished,
	constants.SubmissionStatusLDKReview, constants.SubmissionStatusRevisionRequestedLDK, constants.SubmissionStatusApprovedLDK,
	constants.SubmissionStatusCodeIssued, constants.SubmissionStatusActive,
}

// callerHasTier menentukan apakah pemanggil dapat bertindak sebagai tier
// tertentu — baik lewat organizationTypeCode home org, maupun wildcardTierAccess
// (mis. Super Admin dapat bertindak sebagai tier manapun sesuai kebutuhan
// submission yang sedang diproses).
//
// Poin tambahan (dikonfirmasi 2026-08-18, membalik DL-11 TechSpec): Puskomda/
// Puskomnas Verifikator yang masuk ke konteks sebuah LDK di bawahnya (org-
// switcher shell cms-ldk) sekarang boleh bertindak selengkap LDK Admin,
// termasuk Persetujuan Kader — bukan cuma tier sendiri. Caller SUDAH lolos
// checkOrgAccess(ctx, caller, targetOrganizationID) sebelum method ini
// dipanggil (lihat Review()), jadi delegasi ini aman: cakupan organisasi
// (LDK MANA yang boleh disentuh) tetap dijaga penuh oleh checkOrgAccess,
// method ini hanya melonggarkan syarat IDENTITAS (organizationTypeCode)
// untuk tier LDK spesifik — submission selalu milik organisasi ber-tipe LDK,
// jadi pengecekan ini tidak pernah memberi jalan ke tier PUSKOMDA/PUSKOMNAS.
func (s *ServiceImpl) callerHasTier(ctx context.Context, caller CallerScope, targetOrganizationID int64, tier string) bool {
	if containsTier(caller.WildcardTierAccess, tier) {
		return true
	}
	if caller.OrganizationTypeCode == tier {
		return true
	}
	if tier != constants.ReviewTierLDK {
		return false
	}
	targetType, err := s.orgRepo.TypeCodeByID(ctx, targetOrganizationID)
	return err == nil && targetType == constants.OrgTypeLDK
}

// requiredTierForStatus menentukan tier reviewer yang berwenang bertindak atas
// submission pada status saat ini.
func requiredTierForStatus(subjectType, status string) (string, bool) {
	if subjectType == constants.SubjectTypeKader {
		if status == constants.SubmissionStatusSubmitted || status == constants.SubmissionStatusLDKReview {
			return constants.ReviewTierLDK, true
		}
		return "", false
	}
	switch status {
	case constants.SubmissionStatusSubmitted, constants.SubmissionStatusPuskomdaReview:
		return constants.ReviewTierPuskomda, true
	case constants.SubmissionStatusApprovedPuskomda, constants.SubmissionStatusPuskomnasReview:
		return constants.ReviewTierPuskomnas, true
	default:
		return "", false
	}
}

func nullStringFrom(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func jsonToNullString(raw json.RawMessage) (sql.NullString, error) {
	if len(raw) == 0 {
		return sql.NullString{}, nil
	}
	if !json.Valid(raw) {
		return sql.NullString{}, apperror.BadRequest("Format checklist tidak valid")
	}
	return sql.NullString{String: string(raw), Valid: true}, nil
}

func checkVersion(sub submission_model.Submission, clientVersion int) error {
	if sub.Version != clientVersion {
		return apperror.Conflict("Data telah diubah pihak lain, silakan muat ulang")
	}
	return nil
}

func (s *ServiceImpl) toLevelResultResponse(ctx context.Context, res submission_model.LevelisasiResult) submission_dto.LevelResultResponse {
	label, _ := s.repo.LevelLabel(ctx, res.LevelCode)
	lr := submission_dto.LevelResultResponse{
		ResultID:          res.ResultID,
		LevelCode:         res.LevelCode,
		LevelLabel:        label,
		JustificationNote: res.JustificationNote.String,
		EstablishedDate:   res.EstablishedDate,
		IsPublished:       res.IsPublished,
	}
	if res.PublishedDate.Valid {
		lr.PublishedDate = &res.PublishedDate.Time
	}
	return lr
}

func toKaderResponse(k submission_model.Kader) submission_dto.KaderResponse {
	out := submission_dto.KaderResponse{
		KaderID:        k.KaderID,
		SubmissionID:   k.SubmissionID,
		OrganizationID: k.OrganizationID,
		UniqueCode:     k.UniqueCode.String,
		FullName:       k.FullName,
		Status:         k.Status,
	}
	if k.IssuedDate.Valid {
		out.IssuedDate = &k.IssuedDate.Time
	}
	if k.UpdatedDate.Valid {
		out.StatusChangedDate = &k.UpdatedDate.Time
	}
	return out
}

// ---------- mapping ----------

func (s *ServiceImpl) toResponse(sub submission_model.Submission, formCode string) submission_dto.Response {
	out := submission_dto.Response{
		SubmissionID:   sub.SubmissionID,
		FormID:         sub.FormID,
		FormCode:       formCode,
		FormVersionID:  sub.FormVersionID,
		OrganizationID:   sub.OrganizationID,
		OrganizationName: sub.OrganizationName,
		SubjectType:      sub.SubjectType,
		Status:           sub.Status,
		Version:          sub.Version,
		CreatedDate:      sub.CreatedDate,
	}
	if sub.SubmittedDate.Valid {
		out.SubmittedDate = &sub.SubmittedDate.Time
	}
	return out
}

func (s *ServiceImpl) toResponses(ctx context.Context, subs []submission_model.Submission) []submission_dto.Response {
	formCodeByID := map[int64]string{}
	out := make([]submission_dto.Response, 0, len(subs))
	for _, sub := range subs {
		code, ok := formCodeByID[sub.FormID]
		if !ok {
			if f, err := s.formRepo.FindFormByID(ctx, sub.FormID); err == nil {
				code = f.FormCode
			}
			formCodeByID[sub.FormID] = code
		}
		out = append(out, s.toResponse(sub, code))
	}
	return out
}

func (s *ServiceImpl) recordHistory(ctx context.Context, submissionID int64, fromStatus, toStatus string, caller CallerScope, note string) {
	var from sql.NullString
	if fromStatus != "" {
		from = sql.NullString{String: fromStatus, Valid: true}
	}
	var actorOrg sql.NullInt64
	if caller.OrganizationID != nil {
		actorOrg = sql.NullInt64{Int64: *caller.OrganizationID, Valid: true}
	}
	var n sql.NullString
	if note != "" {
		n = sql.NullString{String: note, Valid: true}
	}
	_ = s.repo.InsertStatusHistory(ctx, submissionID, from, toStatus, caller.UserID, actorOrg, n)
}

// ---------- authorization ----------

// canEdit menentukan apakah caller adalah pemilik data (bukan reviewer yang
// sekadar mengakses lewat cascade organization scope) — Puskomda/Puskomnas
// yang "masuk" ke dashboard LDK lewat switcher tidak boleh mengubah data LDK
// tersebut secara langsung.
// canEdit menentukan apakah caller boleh mengubah jawaban/status submission
// ini. Selain kepemilikan langsung (LDK atas submission miliknya sendiri,
// Kader atas submission miliknya sendiri), Puskomda/Puskomnas yang punya
// org-scope access ke LDK pemilik submission ini JUGA boleh mengedit atas
// nama LDK tersebut (dikonfirmasi 2026-08-18, sejalan dengan callerHasTier
// — Puskomda/Puskomnas boleh bertindak selengkap LDK Admin di LDK bawahannya).
func (s *ServiceImpl) canEdit(ctx context.Context, caller CallerScope, sub submission_model.Submission) bool {
	if caller.WildcardTierAccess != "" {
		return true
	}
	if sub.SubjectType == constants.SubjectTypeKader {
		return caller.UserID == sub.SubmittedByUserID
	}
	if caller.OrganizationID != nil && *caller.OrganizationID == sub.OrganizationID {
		return true
	}
	ok, _ := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, sub.OrganizationID)
	return ok
}

func (s *ServiceImpl) canView(ctx context.Context, caller CallerScope, sub submission_model.Submission) (bool, error) {
	if sub.SubjectType == constants.SubjectTypeKader && caller.UserID == sub.SubmittedByUserID {
		return true, nil
	}
	return s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, sub.OrganizationID)
}

// ---------- Create ----------

func (s *ServiceImpl) findOwnerSubmission(ctx context.Context, subjectType string, organizationID, formID, userID int64) (submission_model.Submission, error) {
	if subjectType == constants.SubjectTypeKader {
		return s.repo.FindByUserAndForm(ctx, userID, formID)
	}
	return s.repo.FindByOwnerOrganization(ctx, organizationID, formID)
}

func (s *ServiceImpl) Create(ctx context.Context, caller CallerScope, req submission_dto.CreateRequest) (submission_dto.Response, error) {
	form, err := s.formRepo.FindFormByCode(ctx, strings.TrimSpace(req.FormCode))
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Form tidak ditemukan")
	}
	version, err := s.formRepo.FindPublishedVersionByForm(ctx, form.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Unprocessable("Form belum memiliki version yang dipublish")
	}

	subjectType := subjectTypeForForm(form.FormCode)

	var organizationID int64
	switch subjectType {
	case constants.SubjectTypeKader:
		// "LDK Tujuan" Kader kini SATU kolom dengan field LDK role lain
		// (ms_user.organizationID) — terkunci seumur hidup begitu terisi
		// (lihat SetOrganizationID), supaya kalau akun ini kelak dipromosikan
		// jadi LDK Admin, LDK yang sama otomatis nempel tanpa admin perlu
		// pilih ulang (resolveEffectiveOrg yang menentukan nilai ini TIDAK
		// memengaruhi cakupan akses sebelum promosi — lihat auth_service_impl.go).
		callerUser, err := s.userRepo.FindByID(ctx, caller.UserID)
		if err != nil {
			return submission_dto.Response{}, apperror.Internal("")
		}
		if callerUser.OrganizationID.Valid {
			organizationID = callerUser.OrganizationID.Int64
		} else {
			if req.OrganizationID == nil {
				return submission_dto.Response{}, apperror.BadRequest("organizationID (LDK tujuan) wajib diisi")
			}
			org, err := s.orgRepo.FindByID(ctx, *req.OrganizationID)
			if err != nil {
				return submission_dto.Response{}, apperror.BadRequest("Organisasi tujuan tidak ditemukan")
			}
			if org.OrganizationTypeCode != constants.OrgTypeLDK {
				return submission_dto.Response{}, apperror.BadRequest("Organisasi tujuan harus berupa LDK")
			}
			if !org.IsActive {
				return submission_dto.Response{}, apperror.Unprocessable("LDK tujuan sedang tidak aktif")
			}
			organizationID = org.OrganizationID
			if err := s.userRepo.SetOrganizationID(ctx, caller.UserID, organizationID); err != nil {
				return submission_dto.Response{}, apperror.Internal("")
			}
		}
	default:
		switch {
		case caller.RequestedOrganizationID != nil:
			// Delegasi: Puskomda/Puskomnas mengisi Pendataan atas nama LDK
			// di bawahnya (org-switcher shell cms-ldk) — dikonfirmasi
			// 2026-08-18, sejalan pembalikan DL-11 (lihat callerHasTier).
			ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *caller.RequestedOrganizationID)
			if err != nil {
				return submission_dto.Response{}, apperror.Internal("")
			}
			if !ok {
				return submission_dto.Response{}, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini")
			}
			targetType, err := s.orgRepo.TypeCodeByID(ctx, *caller.RequestedOrganizationID)
			if err != nil || targetType != constants.OrgTypeLDK {
				return submission_dto.Response{}, apperror.BadRequest("Organisasi tujuan harus berupa LDK")
			}
			organizationID = *caller.RequestedOrganizationID
		case caller.OrganizationTypeCode == constants.OrgTypeLDK && caller.OrganizationID != nil:
			organizationID = *caller.OrganizationID
		default:
			return submission_dto.Response{}, apperror.Forbidden("Hanya LDK yang dapat mengisi form ini")
		}
	}

	existing, err := s.findOwnerSubmission(ctx, subjectType, organizationID, form.FormID, caller.UserID)
	if err == nil {
		switch existing.Status {
		case constants.SubmissionStatusDraft:
			return s.toResponse(existing, form.FormCode), nil
		case constants.SubmissionStatusCancelled:
			if err := s.repo.UpdateStatus(ctx, existing.SubmissionID, constants.SubmissionStatusDraft, sql.NullTime{}, caller.UserID); err != nil {
				return submission_dto.Response{}, apperror.Internal("")
			}
			s.recordHistory(ctx, existing.SubmissionID, existing.Status, constants.SubmissionStatusDraft, caller, "Dibuat ulang setelah dibatalkan")
			existing.Status = constants.SubmissionStatusDraft
			return s.toResponse(existing, form.FormCode), nil
		default:
			return submission_dto.Response{}, apperror.DuplicateSubmission("")
		}
	} else if !errors.Is(err, submission_repository.ErrNotFound) {
		return submission_dto.Response{}, apperror.Internal("")
	}

	id, err := s.repo.Create(ctx, submission_model.CreateParams{
		FormID:            form.FormID,
		FormVersionID:     version.VersionID,
		OrganizationID:    organizationID,
		SubjectType:       subjectType,
		SubmittedByUserID: caller.UserID,
		CreatedBy:         sql.NullInt64{Int64: caller.UserID, Valid: caller.UserID > 0},
	})
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("Gagal membuat submission")
	}
	s.recordHistory(ctx, id, "", constants.SubmissionStatusDraft, caller, "")
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

// ---------- SaveAnswers ----------

func (s *ServiceImpl) buildAnswerParams(ctx context.Context, field submission_form_model.Field, input submission_dto.AnswerInput) (submission_model.AnswerParams, error) {
	p := submission_model.AnswerParams{FieldID: field.FieldID}
	rule := parseValidationRule(field.ValidationRuleJSON)

	switch field.FieldType {
	case constants.FieldTypeText, constants.FieldTypeTextarea:
		if input.ValueText == nil {
			return p, nil
		}
		v := strings.TrimSpace(*input.ValueText)
		if err := rule.validateText(v); err != nil {
			return p, apperror.BadRequest(err.Error())
		}
		p.ValueText = sql.NullString{String: v, Valid: v != ""}
	case constants.FieldTypeNumber:
		if input.ValueNumber == nil {
			return p, nil
		}
		if err := rule.validateNumber(*input.ValueNumber); err != nil {
			return p, apperror.BadRequest(err.Error())
		}
		p.ValueNumber = sql.NullFloat64{Float64: *input.ValueNumber, Valid: true}
	case constants.FieldTypeDate:
		if input.ValueDate == nil {
			return p, nil
		}
		t, err := time.Parse("2006-01-02", *input.ValueDate)
		if err != nil {
			return p, apperror.BadRequest("Format tanggal tidak valid (YYYY-MM-DD)")
		}
		p.ValueDate = sql.NullTime{Time: t, Valid: true}
	case constants.FieldTypeSelect, constants.FieldTypeRadio:
		if input.ValueOptionID == nil {
			return p, nil
		}
		options, err := s.formRepo.ListOptionsByField(ctx, field.FieldID)
		if err != nil {
			return p, apperror.Internal("")
		}
		if !optionExists(options, *input.ValueOptionID) {
			return p, apperror.BadRequest("Pilihan tidak valid untuk field ini")
		}
		p.ValueOptionID = sql.NullInt64{Int64: *input.ValueOptionID, Valid: true}
	case constants.FieldTypeMultiselect, constants.FieldTypeCheckbox:
		if len(input.ValueOptionIDs) == 0 {
			return p, nil
		}
		options, err := s.formRepo.ListOptionsByField(ctx, field.FieldID)
		if err != nil {
			return p, apperror.Internal("")
		}
		for _, id := range input.ValueOptionIDs {
			if !optionExists(options, id) {
				return p, apperror.BadRequest("Pilihan tidak valid untuk field ini")
			}
		}
		encoded, _ := json.Marshal(input.ValueOptionIDs)
		p.ValueText = sql.NullString{String: string(encoded), Valid: true}
	case constants.FieldTypeFileDoc, constants.FieldTypeFileImage:
		if input.ValueFileURL == nil {
			return p, nil
		}
		p.ValueFileURL = sql.NullString{String: *input.ValueFileURL, Valid: true}
		if input.ValueFileName != nil {
			p.ValueFileName = sql.NullString{String: *input.ValueFileName, Valid: true}
		}
	}
	return p, nil
}

// migrateToLatestVersion menyamakan formVersionID submission yang sedang
// diedit ke version PUBLISHED terbaru form-nya, bila ternyata masih mengacu
// ke version lama — akar masalah "Field X bukan bagian dari form ini" saat
// LDK mengisi ulang submission REVISION_REQUESTED_* setelah Puskomnas
// mempublikasikan version baru lewat Form Builder (fieldID field baru tidak
// pernah ada di version lama yang jadi acuan validasi SaveAnswers). Jawaban
// lama direlokasi ke fieldID versi baru lewat kecocokan fieldCode (fieldCode
// dijamin sama persis oleh CloneVersionStructure saat version baru dibuat)
// supaya tidak hilang dari tampilan hanya karena field-nya di-clone ulang
// dengan ID baru; field yang benar-benar baru otomatis tampil kosong.
func (s *ServiceImpl) migrateToLatestVersion(ctx context.Context, sub submission_model.Submission) submission_model.Submission {
	latest, err := s.formRepo.FindPublishedVersionByForm(ctx, sub.FormID)
	if err != nil || latest.VersionID == sub.FormVersionID {
		return sub
	}
	oldFields, err := s.formRepo.ListFieldsByVersion(ctx, sub.FormVersionID)
	if err != nil {
		return sub
	}
	newFields, err := s.formRepo.ListFieldsByVersion(ctx, latest.VersionID)
	if err != nil {
		return sub
	}
	newFieldIDByCode := make(map[string]int64, len(newFields))
	for _, f := range newFields {
		newFieldIDByCode[f.FieldCode] = f.FieldID
	}
	for _, old := range oldFields {
		if newID, ok := newFieldIDByCode[old.FieldCode]; ok && newID != old.FieldID {
			_ = s.repo.RemapAnswerField(ctx, sub.SubmissionID, old.FieldID, newID)
		}
	}
	if err := s.repo.UpdateFormVersion(ctx, sub.SubmissionID, latest.VersionID); err != nil {
		return sub
	}
	sub.FormVersionID = latest.VersionID
	return sub
}

func (s *ServiceImpl) SaveAnswers(ctx context.Context, id int64, caller CallerScope, req submission_dto.SaveAnswersRequest) (submission_dto.DetailResponse, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if !s.canEdit(ctx, caller, sub) {
		return submission_dto.DetailResponse{}, apperror.Forbidden("Anda tidak dapat mengubah pendataan ini")
	}
	if !editableStatuses[sub.Status] {
		return submission_dto.DetailResponse{}, apperror.InvalidStatusTransition("Jawaban hanya dapat diubah selama status DRAFT atau revisi")
	}
	sub = s.migrateToLatestVersion(ctx, sub)

	fields, err := s.formRepo.ListFieldsByVersion(ctx, sub.FormVersionID)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	fieldByID := make(map[int64]submission_form_model.Field, len(fields))
	for _, f := range fields {
		fieldByID[f.FieldID] = f
	}

	for _, input := range req.Answers {
		field, ok := fieldByID[input.FieldID]
		if !ok {
			return submission_dto.DetailResponse{}, apperror.BadRequest(fmt.Sprintf("Field %d bukan bagian dari form ini", input.FieldID))
		}
		params, err := s.buildAnswerParams(ctx, field, input)
		if err != nil {
			return submission_dto.DetailResponse{}, err
		}
		if err := s.repo.UpsertAnswer(ctx, sub.SubmissionID, params); err != nil {
			return submission_dto.DetailResponse{}, apperror.Internal("")
		}
	}
	if err := s.repo.TouchVersion(ctx, sub.SubmissionID, caller.UserID); err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	return s.Get(ctx, id, caller)
}

// ---------- Submit ----------

func (s *ServiceImpl) Submit(ctx context.Context, id int64, caller CallerScope) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if !s.canEdit(ctx, caller, sub) {
		return submission_dto.Response{}, apperror.Forbidden("Anda tidak dapat mengirim pendataan ini")
	}
	if sub.Status == constants.SubmissionStatusCancelled || sub.Status == constants.SubmissionStatusRejected {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Pendataan sudah tidak dapat dikirim lagi")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	if !editableStatuses[sub.Status] {
		// Idempotent: request submit kedua pada submission yang sudah lewat
		// tahap DRAFT/revisi direspons sebagai no-op, bukan error (mencegah
		// double-submit akibat retry/refresh jaringan).
		return s.toResponse(sub, form.FormCode), nil
	}

	org, err := s.orgRepo.FindByID(ctx, sub.OrganizationID)
	if err != nil || !org.IsActive {
		return submission_dto.Response{}, apperror.Unprocessable("Organisasi tujuan sedang tidak aktif")
	}

	fields, err := s.formRepo.ListFieldsByVersion(ctx, sub.FormVersionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	options, err := s.formRepo.ListOptionsByVersion(ctx, sub.FormVersionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	optionValueByID := make(map[int64]string, len(options))
	for _, o := range options {
		optionValueByID[o.OptionID] = o.OptionValue
	}
	answers, err := s.repo.ListAnswersBySubmission(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	answerByField := make(map[int64]submission_model.Answer, len(answers))
	for _, a := range answers {
		answerByField[a.FieldID] = a
	}

	if missing := collectMissingRequiredFields(fields, answerByField, optionValueByID); len(missing) > 0 {
		return submission_dto.Response{}, apperror.Validation("Beberapa field wajib diisi belum lengkap", missing)
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusSubmitted, sql.NullTime{Time: now, Valid: true}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, constants.SubmissionStatusSubmitted, caller, "")

	if sub.SubjectType == constants.SubjectTypeKader {
		if _, err := s.repo.FindKaderBySubmission(ctx, sub.SubmissionID); errors.Is(err, submission_repository.ErrNotFound) {
			fullName := ""
			if u, uerr := s.userRepo.FindByID(ctx, caller.UserID); uerr == nil {
				fullName = u.FullName
			}
			kaderID, err := s.repo.CreateKader(ctx, submission_model.KaderParams{
				SubmissionID:   sub.SubmissionID,
				OrganizationID: sub.OrganizationID,
				UserID:         caller.UserID,
				FullName:       fullName,
			})
			if err != nil {
				return submission_dto.Response{}, apperror.Internal("Gagal membuat data kader")
			}
			if err := s.repo.SetSubjectReference(ctx, sub.SubmissionID, kaderID); err != nil {
				return submission_dto.Response{}, apperror.Internal("")
			}
		}
	}

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

// ---------- Cancel ----------

func (s *ServiceImpl) Cancel(ctx context.Context, id int64, caller CallerScope) error {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Submission tidak ditemukan")
	}
	if !s.canEdit(ctx, caller, sub) {
		return apperror.Forbidden("Anda tidak dapat membatalkan pendataan ini")
	}
	if sub.Status != constants.SubmissionStatusDraft {
		return apperror.InvalidStatusTransition("Hanya pendataan berstatus DRAFT yang dapat dibatalkan")
	}
	if err := s.repo.UpdateStatus(ctx, id, constants.SubmissionStatusCancelled, sql.NullTime{}, caller.UserID); err != nil {
		return apperror.Internal("")
	}
	s.recordHistory(ctx, id, sub.Status, constants.SubmissionStatusCancelled, caller, "")
	return nil
}

// ---------- List & Get ----------

// resolveScopedOrganizationIDs menghitung cakupan organisasi yang berlaku
// untuk sebuah permintaan list (submission/kader): default ke seluruh
// accessible set milik caller, tapi bila klien mengirim `organizationID`
// eksplisit (org-switcher di shell cms-ldk/cms-puskomda), dipersempit ke
// cascade yang berakar pada organisasi itu SETELAH divalidasi accessible
// terhadap caller — mencegah IDOR (TechSpec Section 30) sekaligus jadi
// perbaikan akar untuk bug "ganti organisasi di switcher, data tidak berubah".
func (s *ServiceImpl) resolveScopedOrganizationIDs(ctx context.Context, caller CallerScope) ([]int64, error) {
	if caller.RequestedOrganizationID != nil {
		ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *caller.RequestedOrganizationID)
		if err != nil {
			return nil, apperror.Internal("")
		}
		if !ok {
			return nil, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini")
		}
		ids, err := s.orgScope.AccessibleOrganizationIDsForTarget(ctx, *caller.RequestedOrganizationID)
		if err != nil {
			return nil, apperror.Internal("")
		}
		return ids, nil
	}
	ids, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return nil, apperror.Internal("")
	}
	return ids, nil
}

func (s *ServiceImpl) List(ctx context.Context, caller CallerScope, q dto.ListQuery, statuses []string, formCode string) ([]submission_dto.Response, int, error) {
	filter := submission_dto.ListFilter{
		Statuses: statuses,
		Search:   q.Search,
		Limit:    q.Limit,
		Offset:   q.Offset(),
		OrderBy:  q.OrderBy(sortColumns, "s.createdDate DESC"),
	}
	if formCode != "" {
		form, err := s.formRepo.FindFormByCode(ctx, formCode)
		if err != nil {
			return []submission_dto.Response{}, 0, nil
		}
		filter.FormID = form.FormID
	}
	if caller.OrganizationID == nil && caller.WildcardTierAccess == "" {
		uid := caller.UserID
		filter.SubmittedByUserID = &uid
	} else {
		ids, err := s.resolveScopedOrganizationIDs(ctx, caller)
		if err != nil {
			return nil, 0, err
		}
		filter.OrganizationIDs = ids
	}
	subs, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return s.toResponses(ctx, subs), int(total), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64, caller CallerScope) (submission_dto.DetailResponse, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.NotFound("Submission tidak ditemukan")
	}
	ok, err := s.canView(ctx, caller, sub)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	if editableStatuses[sub.Status] {
		// Migrasi ditampilkan lebih dini di sini (bukan hanya saat SaveAnswers)
		// supaya form yang dimuat pemilik data untuk direvisi langsung
		// menampilkan jawaban lama yang sudah direlokasi, bukan tampak kosong
		// sampai simpan pertama. Reviewer/pihak lain tidak pernah melihat
		// status editable (mereka melihat status review/publish), jadi migrasi
		// ini tidak pernah mengubah data yang sedang direview/sudah terbit.
		sub = s.migrateToLatestVersion(ctx, sub)
	}
	if !ok {
		return submission_dto.DetailResponse{}, apperror.Forbidden("Anda tidak memiliki akses ke pendataan ini")
	}

	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	fields, err := s.formRepo.ListFieldsByVersion(ctx, sub.FormVersionID)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	fieldByID := make(map[int64]submission_form_model.Field, len(fields))
	for _, f := range fields {
		fieldByID[f.FieldID] = f
	}

	answers, err := s.repo.ListAnswersBySubmission(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	answerResponses := make([]submission_dto.AnswerResponse, 0, len(answers))
	for _, a := range answers {
		f := fieldByID[a.FieldID]
		ar := submission_dto.AnswerResponse{FieldID: a.FieldID, FieldCode: f.FieldCode}
		switch {
		case a.ValueOptionID.Valid:
			v := a.ValueOptionID.Int64
			ar.ValueOptionID = &v
		case a.ValueFileURL.Valid:
			ar.ValueFileURL = a.ValueFileURL.String
			ar.ValueFileName = a.ValueFileName.String
		case a.ValueNumber.Valid:
			v := a.ValueNumber.Float64
			ar.ValueNumber = &v
		case a.ValueDate.Valid:
			ar.ValueDate = a.ValueDate.Time.Format("2006-01-02")
		case a.ValueText.Valid:
			if f.FieldType == constants.FieldTypeMultiselect || f.FieldType == constants.FieldTypeCheckbox {
				var ids []int64
				if json.Unmarshal([]byte(a.ValueText.String), &ids) == nil {
					ar.ValueOptionIDs = ids
				}
			} else {
				ar.ValueText = a.ValueText.String
			}
		}
		answerResponses = append(answerResponses, ar)
	}

	history, err := s.repo.ListStatusHistory(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	historyResponses := make([]submission_dto.StatusHistoryResponse, 0, len(history))
	for _, h := range history {
		historyResponses = append(historyResponses, submission_dto.StatusHistoryResponse{
			FromStatus:  h.FromStatus.String,
			ToStatus:    h.ToStatus,
			ActorUserID: h.ActorUserID,
			Note:        h.Note.String,
			CreatedDate: h.CreatedDate,
		})
	}

	var levelResult *submission_dto.LevelResultResponse
	if res, err := s.repo.FindLevelResultBySubmission(ctx, sub.SubmissionID); err == nil {
		lr := s.toLevelResultResponse(ctx, res)
		levelResult = &lr
	}

	// Dibatasi subjek Organisasi (Levelisasi) — subjek Kader tidak relevan
	// dengan tangga level LDK, sama seperti consolidatedScore di bawah.
	var levelHistory []submission_dto.LevelResultResponse
	if sub.SubjectType == constants.SubjectTypeOrganization {
		if rows, err := s.repo.FindLevelResultHistory(ctx, sub.OrganizationID); err == nil {
			levelHistory = make([]submission_dto.LevelResultResponse, 0, len(rows))
			for _, res := range rows {
				levelHistory = append(levelHistory, s.toLevelResultResponse(ctx, res))
			}
		}
	}

	var kaderResp *submission_dto.KaderResponse
	if k, err := s.repo.FindKaderBySubmission(ctx, sub.SubmissionID); err == nil {
		kr := toKaderResponse(k)
		// Nama LDK/Puskomda dibutuhkan kartu "Info Kekaderan" di Profil Saya,
		// yang tampil untuk ACTIVE maupun INACTIVE (lihat
		// user.my-profile.page.html) — kader yang dinonaktifkan tetap pernah
		// disetujui & terikat ke LDK itu, bukan cuma ACTIVE yang relevan.
		if k.Status == constants.KaderStatusActive || k.Status == constants.KaderStatusInactive {
			if org, err := s.orgRepo.FindByID(ctx, k.OrganizationID); err == nil {
				kr.OrganizationName = org.OrganizationName
				kr.ParentOrganizationName = org.ParentOrganizationName.String
			}
		}
		kaderResp = &kr
	}

	// Skor konsolidasi (enhancement Flexible Scoring) dihitung & dikirim untuk
	// Puskomnas (pemberi skor manual) DAN LDK pemilik submission sendiri
	// (dikonfirmasi 2026-08-19: LDK perlu melihat detail & ringkasan skornya
	// sendiri sebagai umpan balik — sejalan dengan levelResult yang juga sudah
	// ditampilkan ke LDK). Puskomda TETAP tidak pernah melihatnya, termasuk
	// saat berdelegasi ke konteks LDK (organizationTypeCode asli caller yang
	// dicek di sini, bukan callerHasTier(...,LDK) yang sengaja longgar untuk
	// delegasi — lihat komentar callerHasTier). Skor murni informatif, tidak
	// memengaruhi EstablishLevel.
	var consolidatedScore *submission_dto.ConsolidatedScoreResponse
	canSeeScore := s.callerHasTier(ctx, caller, sub.OrganizationID, constants.ReviewTierPuskomnas) || caller.OrganizationTypeCode == constants.OrgTypeLDK
	if sub.SubjectType == constants.SubjectTypeOrganization && canSeeScore {
		options, err := s.formRepo.ListOptionsByVersion(ctx, sub.FormVersionID)
		if err == nil {
			manualRows, err := s.repo.ListFieldScoresBySubmission(ctx, sub.SubmissionID)
			if err == nil {
				manualByField := make(map[int64]float64, len(manualRows))
				for _, m := range manualRows {
					manualByField[m.FieldID] = m.RawScore
				}
				cs := computeConsolidatedScore(fields, options, answers, manualByField)
				if len(cs.Fields) > 0 {
					consolidatedScore = &cs
				}
			}
		}
	}

	return submission_dto.DetailResponse{
		Response:          s.toResponse(sub, form.FormCode),
		Answers:           answerResponses,
		StatusHistory:     historyResponses,
		LevelResult:       levelResult,
		LevelHistory:      levelHistory,
		Kader:             kaderResp,
		ConsolidatedScore: consolidatedScore,
	}, nil
}

// ---------- Review ----------

func (s *ServiceImpl) checkOrgAccess(ctx context.Context, caller CallerScope, organizationID int64) error {
	ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, organizationID)
	if err != nil {
		return apperror.Internal("")
	}
	if !ok {
		return apperror.Forbidden("Organisasi di luar wilayah Anda")
	}
	return nil
}

func (s *ServiceImpl) issueKaderCode(ctx context.Context, sub submission_model.Submission, caller CallerScope) error {
	kader, err := s.repo.FindKaderBySubmission(ctx, sub.SubmissionID)
	if err != nil {
		return apperror.Internal("Data kader tidak ditemukan")
	}
	if kader.UniqueCode.Valid {
		// Re-approval setelah "Isi Ulang Pendataan" (ReassessKader) — kode
		// kader sudah pernah terbit sebelumnya, cukup aktifkan kembali
		// submission tanpa menerbitkan kode baru (dikonfirmasi 2026-08-19).
		if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusActive, sql.NullTime{}, caller.UserID); err != nil {
			return apperror.Internal("")
		}
		s.recordHistory(ctx, sub.SubmissionID, constants.SubmissionStatusApprovedLDK, constants.SubmissionStatusActive, caller, "Data kader diperbarui, kode kader tidak berubah")
		return nil
	}
	org, err := s.orgRepo.FindByID(ctx, sub.OrganizationID)
	if err != nil {
		return apperror.Internal("")
	}
	prefix := fmt.Sprintf("KDR-%s-%d-", org.OrganizationCode, time.Now().Year())
	seq, err := s.repo.CountKaderCodesForOrgYear(ctx, prefix)
	if err != nil {
		return apperror.Internal("")
	}
	code := fmt.Sprintf("%s%06d", prefix, seq+1)
	now := time.Now()
	if err := s.repo.UpdateKaderStatus(ctx, kader.KaderID, constants.KaderStatusActive, sql.NullString{String: code, Valid: true}, sql.NullTime{Time: now, Valid: true}); err != nil {
		return apperror.Internal("Gagal menerbitkan kode kader")
	}
	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusActive, sql.NullTime{}, caller.UserID); err != nil {
		return apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, constants.SubmissionStatusApprovedLDK, constants.SubmissionStatusActive, caller, "Kode kader diterbitkan otomatis")
	_ = s.promoteToKaderRole(ctx, kader.UserID)
	return nil
}

func (s *ServiceImpl) Review(ctx context.Context, id int64, caller CallerScope, req submission_dto.ReviewRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}

	// Cakupan organisasi diperiksa sebelum status kelayakan review, sejalan
	// dengan EstablishLevel/Publish/Reopen — caller di luar wilayah tidak
	// perlu tahu apakah submission ini sedang bisa direview atau tidak.
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, err
	}
	tier, ok := requiredTierForStatus(sub.SubjectType, sub.Status)
	if !ok {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Status pendataan tidak dapat direview saat ini")
	}
	if !s.callerHasTier(ctx, caller, sub.OrganizationID, tier) {
		return submission_dto.Response{}, apperror.Forbidden("Anda tidak berwenang mereview pendataan pada tahap ini")
	}
	if err := checkVersion(sub, req.Version); err != nil {
		return submission_dto.Response{}, err
	}

	var toStatus string
	switch tier {
	case constants.ReviewTierLDK:
		switch req.Decision {
		case constants.ReviewDecisionApproved:
			toStatus = constants.SubmissionStatusApprovedLDK
		case constants.ReviewDecisionRevisionRequested:
			toStatus = constants.SubmissionStatusRevisionRequestedLDK
		case constants.ReviewDecisionRejected:
			toStatus = constants.SubmissionStatusRejected
		}
	case constants.ReviewTierPuskomda:
		switch req.Decision {
		case constants.ReviewDecisionApproved:
			toStatus = constants.SubmissionStatusApprovedPuskomda
		case constants.ReviewDecisionRevisionRequested:
			toStatus = constants.SubmissionStatusRevisionRequestedPuskomda
		default:
			return submission_dto.Response{}, apperror.InvalidStatusTransition("Keputusan tidak valid untuk tahap verifikasi wilayah")
		}
	case constants.ReviewTierPuskomnas:
		switch req.Decision {
		case constants.ReviewDecisionApproved:
			toStatus = constants.SubmissionStatusApprovedPuskomnas
		case constants.ReviewDecisionRevisionRequested:
			toStatus = constants.SubmissionStatusRevisionRequestedPuskomnas
		default:
			return submission_dto.Response{}, apperror.InvalidStatusTransition("Keputusan tidak valid untuk tahap verifikasi akhir")
		}
	}

	checklist, err := jsonToNullString(req.Checklist)
	if err != nil {
		return submission_dto.Response{}, err
	}
	if _, err := s.repo.CreateReview(ctx, submission_model.ReviewParams{
		SubmissionID:   sub.SubmissionID,
		TierLevel:      tier,
		ReviewerUserID: caller.UserID,
		Decision:       req.Decision,
		ChecklistJSON:  checklist,
		Note:           nullStringFrom(req.Note),
	}); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, toStatus, sql.NullTime{}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, toStatus, caller, req.Note)

	if tier == constants.ReviewTierLDK {
		switch req.Decision {
		case constants.ReviewDecisionApproved:
			if err := s.issueKaderCode(ctx, sub, caller); err != nil {
				return submission_dto.Response{}, err
			}
		case constants.ReviewDecisionRejected:
			if kader, err := s.repo.FindKaderBySubmission(ctx, sub.SubmissionID); err == nil {
				_ = s.repo.UpdateKaderStatus(ctx, kader.KaderID, constants.KaderStatusRejected, sql.NullString{}, sql.NullTime{})
			}
		}
	}

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

// wentThroughDraftSince menentukan apakah submission pernah kembali ke status
// DRAFT setelah tanggal acuan — dipakai membedakan siklus "Ajukan Reassessment"
// (selalu lewat DRAFT penuh) dari "Reopen for Correction" (langsung ke status
// revisi tanpa lewat DRAFT), sesuai Section 16.
func (s *ServiceImpl) wentThroughDraftSince(ctx context.Context, submissionID int64, since time.Time) (bool, error) {
	history, err := s.repo.ListStatusHistory(ctx, submissionID)
	if err != nil {
		return false, err
	}
	for _, h := range history {
		if h.ToStatus == constants.SubmissionStatusDraft && h.CreatedDate.After(since) {
			return true, nil
		}
	}
	return false, nil
}

func (s *ServiceImpl) EstablishLevel(ctx context.Context, id int64, caller CallerScope, req submission_dto.EstablishLevelRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if sub.SubjectType != constants.SubjectTypeOrganization {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Penetapan level hanya berlaku untuk form Levelisasi")
	}
	if !s.callerHasTier(ctx, caller, sub.OrganizationID, constants.ReviewTierPuskomnas) {
		return submission_dto.Response{}, apperror.Forbidden("Hanya Puskomnas yang dapat menetapkan level")
	}
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, err
	}
	if sub.Status != constants.SubmissionStatusApprovedPuskomnas {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Status pendataan belum siap untuk penetapan level — harus disetujui dahulu di Verifikasi Akhir")
	}
	if err := checkVersion(sub, req.Version); err != nil {
		return submission_dto.Response{}, err
	}
	if _, err := s.repo.LevelLabel(ctx, req.LevelCode); err != nil {
		return submission_dto.Response{}, apperror.BadRequest("Kode level tidak dikenal")
	}

	existing, existingErr := s.repo.FindCurrentLevelResult(ctx, sub.OrganizationID)
	insertNew := true
	if existingErr == nil {
		isReassess, err := s.wentThroughDraftSince(ctx, sub.SubmissionID, existing.EstablishedDate)
		if err != nil {
			return submission_dto.Response{}, apperror.Internal("")
		}
		insertNew = isReassess
	} else if !errors.Is(existingErr, submission_repository.ErrNotFound) {
		return submission_dto.Response{}, apperror.Internal("")
	}

	if insertNew {
		if existingErr == nil {
			if err := s.repo.ClearCurrentLevelResult(ctx, sub.OrganizationID); err != nil {
				return submission_dto.Response{}, apperror.Internal("")
			}
		}
		if _, err := s.repo.CreateLevelResult(ctx, sub.SubmissionID, sub.OrganizationID, req.LevelCode, req.JustificationNote, caller.UserID); err != nil {
			return submission_dto.Response{}, apperror.Internal("")
		}
	} else {
		if err := s.repo.UpdateLevelResult(ctx, existing.ResultID, req.LevelCode, req.JustificationNote, caller.UserID); err != nil {
			return submission_dto.Response{}, apperror.Internal("")
		}
	}

	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusLevelEstablished, sql.NullTime{}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, constants.SubmissionStatusLevelEstablished, caller, req.JustificationNote)

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

// SaveFieldScores menyimpan skor MANUAL (hanya Puskomnas — dikonfirmasi
// 2026-08-19, bukan Puskomda seperti draft awal dokumen enhancement) untuk
// field UseScoring bertipe MANUAL pada satu submission Levelisasi. Dapat
// dipanggil berkali-kali (upsert per field, pola sama seperti SaveAnswers)
// sebelum/sesudah keputusan Verifikasi Akhir maupun Penetapan Levelisasi —
// skor bersifat informatif murni, tidak divalidasi terhadap status tertentu
// selain memastikan submission sudah pernah dikirim (bukan DRAFT/CANCELLED).
func (s *ServiceImpl) SaveFieldScores(ctx context.Context, id int64, caller CallerScope, req submission_dto.SaveFieldScoresRequest) (submission_dto.DetailResponse, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if sub.SubjectType != constants.SubjectTypeOrganization {
		return submission_dto.DetailResponse{}, apperror.InvalidStatusTransition("Skor hanya berlaku untuk form Levelisasi")
	}
	if !s.callerHasTier(ctx, caller, sub.OrganizationID, constants.ReviewTierPuskomnas) {
		return submission_dto.DetailResponse{}, apperror.Forbidden("Hanya Puskomnas yang dapat memberikan skor")
	}
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.DetailResponse{}, err
	}
	if sub.Status == constants.SubmissionStatusDraft || sub.Status == constants.SubmissionStatusCancelled {
		return submission_dto.DetailResponse{}, apperror.InvalidStatusTransition("Belum ada yang dapat dinilai selama pendataan masih draf")
	}

	fields, err := s.formRepo.ListFieldsByVersion(ctx, sub.FormVersionID)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.Internal("")
	}
	fieldByID := make(map[int64]submission_form_model.Field, len(fields))
	for _, f := range fields {
		fieldByID[f.FieldID] = f
	}

	for _, item := range req.Scores {
		field, ok := fieldByID[item.FieldID]
		if !ok {
			return submission_dto.DetailResponse{}, apperror.BadRequest(fmt.Sprintf("Field %d bukan bagian dari form ini", item.FieldID))
		}
		if !field.UseScoring || field.ScoringMethod.String != constants.ScoringMethodManual {
			return submission_dto.DetailResponse{}, apperror.BadRequest(fmt.Sprintf("Field %q tidak menerima skor manual", field.FieldLabel))
		}
		if field.MinScore.Valid && item.RawScore < field.MinScore.Float64 {
			return submission_dto.DetailResponse{}, apperror.BadRequest(fmt.Sprintf("Skor field %q tidak boleh kurang dari %.2f", field.FieldLabel, field.MinScore.Float64))
		}
		if field.MaxScore.Valid && item.RawScore > field.MaxScore.Float64 {
			return submission_dto.DetailResponse{}, apperror.BadRequest(fmt.Sprintf("Skor field %q tidak boleh lebih dari %.2f", field.FieldLabel, field.MaxScore.Float64))
		}
		if err := s.repo.UpsertFieldScore(ctx, sub.SubmissionID, item.FieldID, item.RawScore, caller.UserID); err != nil {
			return submission_dto.DetailResponse{}, apperror.Internal("")
		}
	}

	return s.Get(ctx, id, caller)
}

func (s *ServiceImpl) Publish(ctx context.Context, id int64, caller CallerScope, req submission_dto.VersionedRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if !s.callerHasTier(ctx, caller, sub.OrganizationID, constants.ReviewTierPuskomnas) {
		return submission_dto.Response{}, apperror.Forbidden("Hanya Puskomnas yang dapat mempublikasikan hasil")
	}
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, err
	}
	if sub.Status != constants.SubmissionStatusLevelEstablished {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Level belum ditetapkan")
	}
	if err := checkVersion(sub, req.Version); err != nil {
		return submission_dto.Response{}, err
	}

	result, err := s.repo.FindLevelResultBySubmission(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("Hasil levelisasi tidak ditemukan")
	}
	if err := s.repo.PublishLevelResult(ctx, result.ResultID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusPublished, sql.NullTime{}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, constants.SubmissionStatusPublished, caller, "")

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

func (s *ServiceImpl) Reopen(ctx context.Context, id int64, caller CallerScope, req submission_dto.ReopenRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if !s.callerHasTier(ctx, caller, sub.OrganizationID, constants.ReviewTierPuskomnas) {
		return submission_dto.Response{}, apperror.Forbidden("Hanya Puskomnas yang dapat membuka kembali untuk koreksi")
	}
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, err
	}
	if sub.Status != constants.SubmissionStatusPublished {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Hanya pendataan berstatus PUBLISHED yang dapat dibuka kembali")
	}
	if err := checkVersion(sub, req.Version); err != nil {
		return submission_dto.Response{}, err
	}

	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusRevisionRequestedPuskomnas, sql.NullTime{}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	// Dicatat dengan label semantik "REOPENED_FOR_CORRECTION" (bukan nama
	// status literal) agar tidak disalahartikan sebagai siklus reassessment
	// baru saat riwayat status ditelusuri (lihat wentThroughDraftSince).
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, "REOPENED_FOR_CORRECTION", caller, req.Reason)

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

func (s *ServiceImpl) Reassess(ctx context.Context, id int64, caller CallerScope, req submission_dto.VersionedRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}

	authorized := caller.WildcardTierAccess != ""
	if !authorized && s.callerHasTier(ctx, caller, sub.OrganizationID, constants.ReviewTierPuskomnas) {
		if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err == nil {
			authorized = true
		}
	}
	if !authorized && caller.OrganizationTypeCode == constants.OrgTypeLDK && caller.OrganizationID != nil && *caller.OrganizationID == sub.OrganizationID {
		authorized = true
	}
	if !authorized {
		return submission_dto.Response{}, apperror.Forbidden("Anda tidak berwenang mengajukan reassessment")
	}
	if sub.Status != constants.SubmissionStatusPublished {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Hanya pendataan berstatus PUBLISHED yang dapat diajukan reassessment")
	}
	if err := checkVersion(sub, req.Version); err != nil {
		return submission_dto.Response{}, err
	}

	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusDraft, sql.NullTime{}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	// Level resmi (isCurrent=1) sengaja langsung dicabut di sini, BUKAN
	// ditunda sampai EstablishLevel() siklus berikutnya — per keputusan
	// produk terbaru, LDK yang mengajukan reassessment tidak lagi
	// menampilkan level lama ke publik/agregat selama siklus baru berjalan
	// (membalik DL-15/OQ-11 yang sebelumnya sengaja mempertahankannya).
	// Baris lama TIDAK dihapus (riwayat tetap ada untuk audit), hanya
	// isCurrent di-set 0 — EstablishLevel() berikutnya otomatis membuat
	// baris isCurrent=1 baru lewat CreateLevelResult() saat tidak
	// menemukan baris current (lihat FindCurrentLevelResult/ErrNotFound).
	if err := s.repo.ClearCurrentLevelResult(ctx, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, constants.SubmissionStatusDraft, caller, "REASSESSMENT_REQUESTED")

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

// ---------- Kader ----------

func (s *ServiceImpl) ListKaders(ctx context.Context, caller CallerScope, q dto.ListQuery, status string) ([]submission_dto.KaderResponse, int, error) {
	ids, err := s.resolveScopedOrganizationIDs(ctx, caller)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []submission_dto.KaderResponse{}, 0, nil
	}
	kaders, err := s.repo.ListKadersByOrganizations(ctx, ids, status)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	total := len(kaders)
	start := q.Offset()
	if start > total {
		start = total
	}
	end := start + q.Limit
	if end > total {
		end = total
	}
	page := kaders[start:end]
	out := make([]submission_dto.KaderResponse, 0, len(page))
	for _, k := range page {
		out = append(out, toKaderResponse(k))
	}
	return out, total, nil
}

func (s *ServiceImpl) GetKaderCode(ctx context.Context, kaderID int64, caller CallerScope) (submission_dto.KaderResponse, error) {
	k, err := s.repo.FindKaderByID(ctx, kaderID)
	if err != nil {
		return submission_dto.KaderResponse{}, apperror.NotFound("Data kader tidak ditemukan")
	}
	if caller.UserID != k.UserID {
		if err := s.checkOrgAccess(ctx, caller, k.OrganizationID); err != nil {
			return submission_dto.KaderResponse{}, err
		}
	}
	return toKaderResponse(k), nil
}

func (s *ServiceImpl) DeactivateKader(ctx context.Context, kaderID int64, caller CallerScope) error {
	k, err := s.repo.FindKaderByID(ctx, kaderID)
	if err != nil {
		return apperror.NotFound("Data kader tidak ditemukan")
	}
	if caller.WildcardTierAccess == "" && (caller.OrganizationID == nil || *caller.OrganizationID != k.OrganizationID) {
		// Delegasi Puskomda/Puskomnas atas nama LDK di bawahnya (2026-08-18,
		// sejalan callerHasTier/canEdit — lihat komentar callerHasTier).
		ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, k.OrganizationID)
		if err != nil {
			return apperror.Internal("")
		}
		if !ok {
			return apperror.Forbidden("Anda hanya dapat menonaktifkan kader milik LDK Anda sendiri")
		}
	}
	if k.Status != constants.KaderStatusActive {
		return apperror.InvalidStatusTransition("Hanya kader berstatus ACTIVE yang dapat dinonaktifkan")
	}
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusInactive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	// Cascade: kalau kader ini sedang memegang role pengurus (LDK Admin/
	// Puskomda/Puskomnas Verifikator), otomatis turunkan balik ke Kader +
	// organizationID balik ke LDK asalnya — tidak masuk akal akun nonaktif
	// tetap pegang akses admin. Best-effort seperti promoteToKaderRole:
	// kegagalan di sini TIDAK membatalkan nonaktifkan kader-nya sendiri.
	if u, err := s.userRepo.FindByID(ctx, k.UserID); err == nil {
		if u.RoleName == constants.RoleLDKAdmin || u.RoleName == constants.RolePuskomdaVerifikator || u.RoleName == constants.RolePuskomnasVerifikator {
			_ = s.setPengurusRole(ctx, k.UserID, k.OrganizationID, roleNameKader, caller.UserID)
		}
	}
	return nil
}

// ReactivateKader membalikkan DeactivateKader — mengembalikan kader
// berstatus INACTIVE menjadi ACTIVE lagi TANPA mereset submission (data yang
// sudah lolos review sebelumnya tetap dipakai apa adanya, uniqueCode yang
// sudah terbit juga tidak disentuh). Beda dari ReinstateKader (REJECTED→
// PENDING, submission direset total ke DRAFT) — kader INACTIVE memang sudah
// pernah disetujui, cuma dinonaktifkan sementara, jadi tidak perlu isi ulang
// dari nol.
func (s *ServiceImpl) ReactivateKader(ctx context.Context, kaderID int64, caller CallerScope) error {
	k, err := s.repo.FindKaderByID(ctx, kaderID)
	if err != nil {
		return apperror.NotFound("Data kader tidak ditemukan")
	}
	if caller.WildcardTierAccess == "" && (caller.OrganizationID == nil || *caller.OrganizationID != k.OrganizationID) {
		// Delegasi Puskomda/Puskomnas atas nama LDK di bawahnya, sama seperti
		// DeactivateKader.
		ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, k.OrganizationID)
		if err != nil {
			return apperror.Internal("")
		}
		if !ok {
			return apperror.Forbidden("Anda hanya dapat mengaktifkan kembali kader milik LDK Anda sendiri")
		}
	}
	if k.Status != constants.KaderStatusInactive {
		return apperror.InvalidStatusTransition("Hanya kader berstatus Nonaktif yang dapat diaktifkan kembali")
	}
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusActive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// setPengurusRole adalah inti mutasi role pengurus, dipakai UpdatePengurusRole
// (promosi/demosi manual lewat endpoint) MAUPUN cascade DeactivateKader
// (revert otomatis ke Kader) — satu fungsi supaya kedua jalur itu tidak
// pernah berbeda perilaku. organizationID BARU dihitung dari roleName
// TUJUAN (bukan role asal): Kader/LDK Admin selalu balik ke
// homeOrganizationID (ms_kader.organizationID, tidak pernah berubah);
// Puskomda Verifikator pindah ke Puskomda induk LDK itu; Puskomnas
// Verifikator pindah ke Puskomnas tunggal. Wajib begini karena
// ms_user.organizationTypeCode SELALU di-join live dari organizationID
// (lihat user_repository_impl.go selectCols) dan itulah yang menentukan
// cakupan akses caller di seluruh backend — kalau organizationID tidak
// ikut pindah, role Verifikator-nya tidak akan berfungsi penuh.
func (s *ServiceImpl) setPengurusRole(ctx context.Context, userID, homeOrganizationID int64, roleName string, actorID int64) error {
	var targetOrgID int64
	switch roleName {
	case roleNameKader, constants.RoleLDKAdmin:
		targetOrgID = homeOrganizationID
	case constants.RolePuskomdaVerifikator:
		homeOrg, err := s.orgRepo.FindByID(ctx, homeOrganizationID)
		if err != nil {
			return apperror.Internal("")
		}
		if !homeOrg.ParentOrganizationID.Valid {
			return apperror.Internal("LDK ini tidak memiliki Puskomda induk")
		}
		targetOrgID = homeOrg.ParentOrganizationID.Int64
	case constants.RolePuskomnasVerifikator:
		rootID, err := s.orgRepo.RootPuskomnasID(ctx)
		if err != nil {
			return apperror.Internal("")
		}
		targetOrgID = rootID
	default:
		return apperror.BadRequest("Role tidak dikenali")
	}
	roleID, err := s.roleRepo.IDByName(ctx, roleName)
	if err != nil {
		return apperror.BadRequest("Role tidak ditemukan")
	}
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.userRepo.Update(ctx, userID, u.FullName, u.Email, roleID, u.IsActive, sql.NullInt64{Int64: targetOrgID, Valid: true}, u.WildcardTierAccess, actorID); err != nil {
		return apperror.Internal("")
	}
	return s.userRepo.BumpTokenVersion(ctx, userID)
}

func (s *ServiceImpl) pengurusEntryFor(ctx context.Context, userID, homeOrganizationID int64) (submission_dto.PengurusEntry, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	org, err := s.orgRepo.FindByID(ctx, homeOrganizationID)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	return submission_dto.PengurusEntry{
		UserID: u.UserID, FullName: u.FullName, Email: u.Email, RoleName: u.RoleName,
		HomeOrganizationID: homeOrganizationID, HomeOrganizationName: org.OrganizationName,
	}, nil
}

func (s *ServiceImpl) ListPengurus(ctx context.Context, caller CallerScope) ([]submission_dto.PengurusEntry, error) {
	orgIDs, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return nil, apperror.Internal("")
	}
	kaders, err := s.repo.ListKadersByOrganizations(ctx, orgIDs, constants.KaderStatusActive)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]submission_dto.PengurusEntry, 0, len(kaders))
	for _, k := range kaders {
		entry, err := s.pengurusEntryFor(ctx, k.UserID, k.OrganizationID)
		if err != nil {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *ServiceImpl) UpdatePengurusRole(ctx context.Context, targetUserID int64, caller CallerScope, req submission_dto.UpdatePengurusRoleRequest) (submission_dto.PengurusEntry, error) {
	if targetUserID == caller.UserID {
		return submission_dto.PengurusEntry{}, apperror.Forbidden("Anda tidak dapat mengubah role akun Anda sendiri")
	}
	if !pengurusRoleAllowed(caller.OrganizationTypeCode, req.RoleName) {
		return submission_dto.PengurusEntry{}, apperror.Forbidden("Role tersebut tidak dapat ditetapkan dari portal ini")
	}
	orgIDs, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	kaders, err := s.repo.ListKadersByOrganizations(ctx, orgIDs, constants.KaderStatusActive)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	var homeOrgID int64
	found := false
	for _, k := range kaders {
		if k.UserID == targetUserID {
			homeOrgID = k.OrganizationID
			found = true
			break
		}
	}
	if !found {
		return submission_dto.PengurusEntry{}, apperror.NotFound("Kader tidak ditemukan atau di luar jangkauan akses Anda")
	}
	if err := s.setPengurusRole(ctx, targetUserID, homeOrgID, req.RoleName, caller.UserID); err != nil {
		return submission_dto.PengurusEntry{}, err
	}
	return s.pengurusEntryFor(ctx, targetUserID, homeOrgID)
}

// ReassessKader mengizinkan kader ACTIVE mengisi ulang data Sensus Kader-nya
// sendiri ketika Puskomnas sudah mempublikasikan version form baru — TANPA
// menerbitkan kode kader baru (dikonfirmasi 2026-08-19). Submission direset
// ke DRAFT & dipindah ke version terbaru (lihat migrateToLatestVersion untuk
// bagaimana jawaban lama direlokasi), lalu mengikuti alur Submit()/Review()
// normal seperti pengisian pertama kali — issueKaderCode() sudah dijaga
// terpisah untuk tidak menerbitkan kode baru bila kader.uniqueCode sudah ada.
func (s *ServiceImpl) ReassessKader(ctx context.Context, id int64, caller CallerScope, req submission_dto.VersionedRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if sub.SubjectType != constants.SubjectTypeKader {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Isi ulang pendataan hanya berlaku untuk Sensus Kader")
	}
	if sub.SubmittedByUserID != caller.UserID {
		return submission_dto.Response{}, apperror.Forbidden("Anda hanya dapat mengisi ulang pendataan milik Anda sendiri")
	}
	if sub.Status != constants.SubmissionStatusActive {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Isi ulang hanya dapat dilakukan saat status Aktif")
	}
	if err := checkVersion(sub, req.Version); err != nil {
		return submission_dto.Response{}, err
	}
	latest, err := s.formRepo.FindPublishedVersionByForm(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	if latest.VersionID == sub.FormVersionID {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Belum ada pembaruan formulir terbaru")
	}
	sub = s.migrateToLatestVersion(ctx, sub)

	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusDraft, sql.NullTime{}, caller.UserID); err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, constants.SubmissionStatusActive, constants.SubmissionStatusDraft, caller, "Isi ulang pendataan (formulir version baru)")

	sub, err = s.repo.FindByID(ctx, sub.SubmissionID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	form, err := s.formRepo.FindFormByID(ctx, sub.FormID)
	if err != nil {
		return submission_dto.Response{}, apperror.Internal("")
	}
	return s.toResponse(sub, form.FormCode), nil
}

// CountAffectedByForm menghitung submission (lintas organisasi) yang akan
// terdampak cascade publish — memenuhi submission_form_service.SubmissionCascader.
func (s *ServiceImpl) CountAffectedByForm(ctx context.Context, formID int64) (int, error) {
	return s.repo.CountByFormAndStatuses(ctx, formID, cascadeEligibleStatuses)
}

// EnqueueCascade memasukkan job reset/reassessment massal ke jobqueue —
// memenuhi submission_form_service.SubmissionCascader. Dipanggil
// submission_form_service.PublishVersion setelah version baru berhasil dipublish.
func (s *ServiceImpl) EnqueueCascade(ctx context.Context, in submission_form_service.CascadeInput) (int64, error) {
	payload := submission_dto.CascadeJobPayload{
		FormID: in.FormID, OldVersionID: in.OldVersionID, NewVersionID: in.NewVersionID,
		Mode: in.Mode, ActorID: in.ActorID,
	}
	return s.jobqueueSvc.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueDefault, JobType: constants.JobSubmissionFormPublishCascade, Payload: payload,
	})
}

// HandlePublishCascadeJob adalah handler jobqueue untuk
// constants.JobSubmissionFormPublishCascade (didaftarkan lewat
// jobqueueSvc.RegisterHandler di root router.go) — memproses seluruh
// submission yang mengacu ke version lama form ini (status apa pun kecuali
// DRAFT/CANCELLED/REJECTED, lihat cascadeEligibleStatuses) sesuai mode
// RESET/REASSESS yang dipilih admin saat publish. Query ulang dari
// ListByFormAndStatuses di setiap retry otomatis idempotent: submission yang
// sudah diproses sudah berstatus DRAFT (RESET) atau DRAFT (REASSESS), jadi
// tidak akan muncul lagi di hasil query (lihat TechSpec §6).
func (s *ServiceImpl) HandlePublishCascadeJob(ctx context.Context, payload string) error {
	var p submission_dto.CascadeJobPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	subs, err := s.repo.ListByFormAndStatuses(ctx, p.FormID, cascadeEligibleStatuses)
	if err != nil {
		return err
	}
	caller := CallerScope{UserID: p.ActorID}
	for _, sub := range subs {
		if p.Mode == constants.CascadeModeReset {
			if err := s.repo.WipeSubmission(ctx, sub, p.NewVersionID); err != nil {
				return fmt.Errorf("wipe submission %d: %w", sub.SubmissionID, err)
			}
			continue
		}
		// REASSESS: remap jawaban lama ke version baru by fieldCode (mekanisme
		// sama dengan migrateToLatestVersion yang sudah dipakai ReassessKader),
		// lalu reset ke DRAFT dengan histori tetap tercatat.
		migrated := s.migrateToLatestVersion(ctx, sub)
		if err := s.repo.UpdateStatus(ctx, migrated.SubmissionID, constants.SubmissionStatusDraft, sql.NullTime{}, p.ActorID); err != nil {
			return fmt.Errorf("reassess submission %d: %w", sub.SubmissionID, err)
		}
		if migrated.SubjectType != constants.SubjectTypeKader {
			if err := s.repo.ClearCurrentLevelResult(ctx, migrated.OrganizationID); err != nil {
				return fmt.Errorf("clear level result submission %d: %w", sub.SubmissionID, err)
			}
		}
		s.recordHistory(ctx, migrated.SubmissionID, sub.Status, constants.SubmissionStatusDraft, caller, "REASSESSMENT_REQUESTED (bulk - form version published)")
	}
	return nil
}

// ReinstateKader "memutihkan kembali" kader berstatus REJECTED atas permintaan
// LDK — submission direset total ke DRAFT (dikonfirmasi 2026-08-19: kader
// mengisi ulang dari nol, bukan langsung aktif atau sekadar direvisi) supaya
// bisa mendaftar ulang lewat alur normal. Kader.uniqueCode tidak pernah terisi
// untuk kader REJECTED, jadi issueKaderCode() pada persetujuan berikutnya
// otomatis menerbitkan kode baru seperti pendaftaran pertama kali.
func (s *ServiceImpl) ReinstateKader(ctx context.Context, kaderID int64, caller CallerScope) error {
	k, err := s.repo.FindKaderByID(ctx, kaderID)
	if err != nil {
		return apperror.NotFound("Data kader tidak ditemukan")
	}
	if !s.callerHasTier(ctx, caller, k.OrganizationID, constants.ReviewTierLDK) {
		return apperror.Forbidden("Hanya LDK yang dapat memutihkan kembali data kader")
	}
	if err := s.checkOrgAccess(ctx, caller, k.OrganizationID); err != nil {
		return err
	}
	if k.Status != constants.KaderStatusRejected {
		return apperror.InvalidStatusTransition("Hanya data kader berstatus Ditolak yang dapat diputihkan kembali")
	}
	sub, err := s.repo.FindByID(ctx, k.SubmissionID)
	if err != nil {
		return apperror.Internal("")
	}
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusPending, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	if err := s.repo.UpdateStatus(ctx, sub.SubmissionID, constants.SubmissionStatusDraft, sql.NullTime{}, caller.UserID); err != nil {
		return apperror.Internal("")
	}
	s.recordHistory(ctx, sub.SubmissionID, sub.Status, constants.SubmissionStatusDraft, caller, "Diputihkan kembali oleh LDK")
	return nil
}
