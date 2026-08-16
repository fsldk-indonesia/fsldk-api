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
	"fsldk-api/modules/organization/organization_repository"
	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission/submission_repository"
	"fsldk-api/modules/submission_form/submission_form_model"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/modules/user/user_repository"
)

var sortColumns = map[string]string{
	"createdDate":   "createdDate",
	"submittedDate": "submittedDate",
	"status":        "status",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo     submission_repository.Repository
	formRepo submission_form_repository.Repository
	orgRepo  organization_repository.Repository
	userRepo user_repository.Repository
	orgScope OrgScopeResolver
}

// NewService membuat Service submission.
func NewService(
	repo submission_repository.Repository,
	formRepo submission_form_repository.Repository,
	orgRepo organization_repository.Repository,
	userRepo user_repository.Repository,
	orgScope OrgScopeResolver,
) Service {
	return &ServiceImpl{repo: repo, formRepo: formRepo, orgRepo: orgRepo, userRepo: userRepo, orgScope: orgScope}
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

// callerHasTier menentukan apakah pemanggil dapat bertindak sebagai tier
// tertentu — baik lewat organizationTypeCode home org, maupun wildcardTierAccess
// (mis. Super Admin dapat bertindak sebagai tier manapun sesuai kebutuhan
// submission yang sedang diproses).
func callerHasTier(caller CallerScope, tier string) bool {
	if containsTier(caller.WildcardTierAccess, tier) {
		return true
	}
	return caller.OrganizationTypeCode == tier
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
	return out
}

// ---------- mapping ----------

func (s *ServiceImpl) toResponse(sub submission_model.Submission, formCode string) submission_dto.Response {
	out := submission_dto.Response{
		SubmissionID:   sub.SubmissionID,
		FormID:         sub.FormID,
		FormCode:       formCode,
		FormVersionID:  sub.FormVersionID,
		OrganizationID: sub.OrganizationID,
		SubjectType:    sub.SubjectType,
		Status:         sub.Status,
		Version:        sub.Version,
		CreatedDate:    sub.CreatedDate,
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
func (s *ServiceImpl) canEdit(caller CallerScope, sub submission_model.Submission) bool {
	if caller.WildcardTierAccess != "" {
		return true
	}
	if sub.SubjectType == constants.SubjectTypeKader {
		return caller.UserID == sub.SubmittedByUserID
	}
	return caller.OrganizationID != nil && *caller.OrganizationID == sub.OrganizationID
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
	default:
		if caller.OrganizationTypeCode != constants.OrgTypeLDK || caller.OrganizationID == nil {
			return submission_dto.Response{}, apperror.Forbidden("Hanya LDK yang dapat mengisi form ini")
		}
		organizationID = *caller.OrganizationID
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

func (s *ServiceImpl) SaveAnswers(ctx context.Context, id int64, caller CallerScope, req submission_dto.SaveAnswersRequest) (submission_dto.DetailResponse, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.DetailResponse{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if !s.canEdit(caller, sub) {
		return submission_dto.DetailResponse{}, apperror.Forbidden("Anda tidak dapat mengubah pendataan ini")
	}
	if !editableStatuses[sub.Status] {
		return submission_dto.DetailResponse{}, apperror.InvalidStatusTransition("Jawaban hanya dapat diubah selama status DRAFT atau revisi")
	}

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
	if !s.canEdit(caller, sub) {
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

	for _, f := range fields {
		if !f.IsRequired {
			continue
		}
		if f.ConditionalOnFieldID.Valid {
			triggerAnswer, ok := answerByField[f.ConditionalOnFieldID.Int64]
			if !evaluateConditional(f.ConditionalRuleJSON, triggerAnswer, ok, optionValueByID) {
				continue
			}
		}
		a, ok := answerByField[f.FieldID]
		if !ok || !hasValue(a) {
			return submission_dto.Response{}, apperror.Unprocessable(fmt.Sprintf("Field \"%s\" wajib diisi", f.FieldLabel))
		}
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
	if !s.canEdit(caller, sub) {
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

func (s *ServiceImpl) List(ctx context.Context, caller CallerScope, q dto.ListQuery, status string) ([]submission_dto.Response, int, error) {
	filter := submission_dto.ListFilter{
		Status:  status,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "createdDate DESC"),
	}
	if caller.OrganizationID == nil && caller.WildcardTierAccess == "" {
		uid := caller.UserID
		filter.SubmittedByUserID = &uid
	} else {
		ids, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
		if err != nil {
			return nil, 0, apperror.Internal("")
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
		levelResult = &lr
	}

	var kaderResp *submission_dto.KaderResponse
	if k, err := s.repo.FindKaderBySubmission(ctx, sub.SubmissionID); err == nil {
		kr := toKaderResponse(k)
		kaderResp = &kr
	}

	return submission_dto.DetailResponse{
		Response:      s.toResponse(sub, form.FormCode),
		Answers:       answerResponses,
		StatusHistory: historyResponses,
		LevelResult:   levelResult,
		Kader:         kaderResp,
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
	return nil
}

func (s *ServiceImpl) Review(ctx context.Context, id int64, caller CallerScope, req submission_dto.ReviewRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}

	tier, ok := requiredTierForStatus(sub.SubjectType, sub.Status)
	if !ok {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Status pendataan tidak dapat direview saat ini")
	}
	if !callerHasTier(caller, tier) {
		return submission_dto.Response{}, apperror.Forbidden("Anda tidak berwenang mereview pendataan pada tahap ini")
	}
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, err
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
		case constants.ReviewDecisionRevisionRequested:
			toStatus = constants.SubmissionStatusRevisionRequestedPuskomnas
		default:
			return submission_dto.Response{}, apperror.InvalidStatusTransition("Gunakan Penetapan Levelisasi untuk menyetujui pada tahap verifikasi akhir")
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
	if !callerHasTier(caller, constants.ReviewTierPuskomnas) {
		return submission_dto.Response{}, apperror.Forbidden("Hanya Puskomnas yang dapat menetapkan level")
	}
	if err := s.checkOrgAccess(ctx, caller, sub.OrganizationID); err != nil {
		return submission_dto.Response{}, err
	}
	if sub.Status != constants.SubmissionStatusApprovedPuskomda && sub.Status != constants.SubmissionStatusPuskomnasReview {
		return submission_dto.Response{}, apperror.InvalidStatusTransition("Status pendataan belum siap untuk penetapan level")
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

func (s *ServiceImpl) Publish(ctx context.Context, id int64, caller CallerScope, req submission_dto.VersionedRequest) (submission_dto.Response, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return submission_dto.Response{}, apperror.NotFound("Submission tidak ditemukan")
	}
	if !callerHasTier(caller, constants.ReviewTierPuskomnas) {
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
	if !callerHasTier(caller, constants.ReviewTierPuskomnas) {
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
	if !authorized && callerHasTier(caller, constants.ReviewTierPuskomnas) {
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
	ids, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return nil, 0, apperror.Internal("")
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
		return apperror.Forbidden("Anda hanya dapat menonaktifkan kader milik LDK Anda sendiri")
	}
	if k.Status != constants.KaderStatusActive {
		return apperror.InvalidStatusTransition("Hanya kader berstatus ACTIVE yang dapat dinonaktifkan")
	}
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusInactive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	return nil
}
