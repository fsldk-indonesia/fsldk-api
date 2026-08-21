package submission_form_service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/submission_form/submission_form_dto"
	"fsldk-api/modules/submission_form/submission_form_model"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/pkg/auditlog"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo  submission_form_repository.Repository
	audit *auditlog.Logger
}

// NewService membuat Service submission_form.
func NewService(repo submission_form_repository.Repository, audit *auditlog.Logger) Service {
	return &ServiceImpl{repo: repo, audit: audit}
}

// ---------- helper ----------

func nullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func jsonToNullString(raw json.RawMessage) (sql.NullString, error) {
	if len(raw) == 0 {
		return sql.NullString{}, nil
	}
	if !json.Valid(raw) {
		return sql.NullString{}, apperror.BadRequest("Format JSON tidak valid")
	}
	return sql.NullString{String: string(raw), Valid: true}, nil
}

func nullStringToJSON(s sql.NullString) json.RawMessage {
	if !s.Valid || s.String == "" {
		return nil
	}
	return json.RawMessage(s.String)
}

func nullFloat(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

func floatPtr(f sql.NullFloat64) *float64 {
	if !f.Valid {
		return nil
	}
	v := f.Float64
	return &v
}

// singleChoiceFieldTypes adalah tipe field yang boleh memakai scoring
// AUTOMATIC (Single Choice — tepat satu opsi terpilih per jawaban).
var singleChoiceFieldTypes = map[string]bool{
	constants.FieldTypeSelect: true,
	constants.FieldTypeRadio:  true,
}

// validateScoringConfig memvalidasi & menyiapkan nilai scoring field siap
// pakai untuk FieldParams — dipanggil dari CreateField/UpdateField. Aturan
// (design/development/enahnce-development-submission-dashboard/
// new-enhance-development.md §2/§10): scoring tidak dikunci ke skala 1-4
// tertentu, minScore/maxScore/weight bebas dikonfigurasi per field; AUTOMATIC
// hanya valid untuk field Single Choice (SELECT/RADIO).
func validateScoringConfig(fieldType string, useScoring bool, scoringMethod string, minScore, maxScore, weight *float64) (sql.NullString, sql.NullFloat64, sql.NullFloat64, sql.NullFloat64, error) {
	if !useScoring {
		return sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{}, sql.NullFloat64{}, nil
	}
	if scoringMethod == "" {
		return sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{}, sql.NullFloat64{}, apperror.BadRequest("scoringMethod wajib diisi saat useScoring aktif")
	}
	if scoringMethod == constants.ScoringMethodAutomatic && !singleChoiceFieldTypes[fieldType] {
		return sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{}, sql.NullFloat64{}, apperror.BadRequest("Scoring otomatis hanya berlaku untuk field Single Choice (SELECT/RADIO)")
	}
	if minScore == nil || maxScore == nil {
		return sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{}, sql.NullFloat64{}, apperror.BadRequest("minScore dan maxScore wajib diisi saat useScoring aktif")
	}
	if *maxScore <= *minScore {
		return sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{}, sql.NullFloat64{}, apperror.BadRequest("maxScore harus lebih besar dari minScore")
	}
	if weight == nil || *weight <= 0 || *weight > 100 {
		return sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{}, sql.NullFloat64{}, apperror.BadRequest("weight wajib diisi, antara 0 (eksklusif) dan 100 saat useScoring aktif")
	}
	return sql.NullString{String: scoringMethod, Valid: true}, nullFloat(minScore), nullFloat(maxScore), nullFloat(weight), nil
}

func toFormResponse(f submission_form_model.Form) submission_form_dto.FormResponse {
	return submission_form_dto.FormResponse{
		FormID:      f.FormID,
		FormCode:    f.FormCode,
		FormName:    f.FormName,
		Description: f.Description.String,
		IsActive:    f.IsActive,
		CreatedDate: f.CreatedDate,
	}
}

func toVersionSummary(v submission_form_model.Version) submission_form_dto.VersionSummary {
	out := submission_form_dto.VersionSummary{
		VersionID:     v.VersionID,
		VersionNumber: v.VersionNumber,
		Status:        v.Status,
	}
	if v.PublishedDate.Valid {
		out.PublishedDate = &v.PublishedDate.Time
	}
	return out
}

func toOptionResponse(o submission_form_model.Option) submission_form_dto.OptionResponse {
	return submission_form_dto.OptionResponse{
		OptionID:    o.OptionID,
		OptionValue: o.OptionValue,
		OptionLabel: o.OptionLabel,
		SortOrder:   o.SortOrder,
		IsActive:    o.IsActive,
		Score:       floatPtr(o.Score),
	}
}

func toFieldResponse(f submission_form_model.Field, options []submission_form_dto.OptionResponse) submission_form_dto.FieldResponse {
	out := submission_form_dto.FieldResponse{
		FieldID:         f.FieldID,
		SectionID:       f.SectionID,
		FieldCode:       f.FieldCode,
		FieldLabel:      f.FieldLabel,
		FieldType:       f.FieldType,
		IsRequired:      f.IsRequired,
		SortOrder:       f.SortOrder,
		ValidationRule:  nullStringToJSON(f.ValidationRuleJSON),
		ConditionalRule: nullStringToJSON(f.ConditionalRuleJSON),
		HelpText:        f.HelpText.String,
		UseScoring:      f.UseScoring,
		ScoringMethod:   f.ScoringMethod.String,
		MinScore:        floatPtr(f.MinScore),
		MaxScore:        floatPtr(f.MaxScore),
		Weight:          floatPtr(f.Weight),
		Options:         options,
	}
	if f.ConditionalOnFieldID.Valid {
		id := f.ConditionalOnFieldID.Int64
		out.ConditionalOnFieldID = &id
	}
	if out.Options == nil {
		out.Options = []submission_form_dto.OptionResponse{}
	}
	return out
}

// ---------- Form ----------

func (s *ServiceImpl) ListForms(ctx context.Context) ([]submission_form_dto.FormResponse, error) {
	forms, err := s.repo.ListForms(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]submission_form_dto.FormResponse, 0, len(forms))
	for _, f := range forms {
		out = append(out, toFormResponse(f))
	}
	return out, nil
}

func (s *ServiceImpl) CreateForm(ctx context.Context, req submission_form_dto.CreateFormRequest, actorID int64) (submission_form_dto.FormResponse, error) {
	exists, err := s.repo.ExistsFormCode(ctx, req.FormCode)
	if err != nil {
		return submission_form_dto.FormResponse{}, apperror.Internal("")
	}
	if exists {
		return submission_form_dto.FormResponse{}, apperror.Conflict("Kode form sudah digunakan")
	}
	id, err := s.repo.CreateForm(ctx, strings.TrimSpace(req.FormCode), strings.TrimSpace(req.FormName), strings.TrimSpace(req.Description),
		sql.NullInt64{Int64: actorID, Valid: actorID > 0})
	if err != nil {
		return submission_form_dto.FormResponse{}, apperror.Internal("Gagal membuat form")
	}
	s.audit.LogForm(ctx, auditlog.Entry{
		ActorUserID: actorID, Action: "CREATE", Entity: "ms_submission_form", EntityID: id,
		After: map[string]interface{}{"formCode": req.FormCode, "formName": req.FormName},
	})
	f, err := s.repo.FindFormByID(ctx, id)
	if err != nil {
		return submission_form_dto.FormResponse{}, apperror.Internal("")
	}
	return toFormResponse(f), nil
}

func (s *ServiceImpl) GetForm(ctx context.Context, formID int64) (submission_form_dto.FormDetailResponse, error) {
	f, err := s.repo.FindFormByID(ctx, formID)
	if err != nil {
		return submission_form_dto.FormDetailResponse{}, apperror.NotFound("Form tidak ditemukan")
	}
	versions, err := s.repo.ListVersionsByForm(ctx, formID)
	if err != nil {
		return submission_form_dto.FormDetailResponse{}, apperror.Internal("")
	}
	out := submission_form_dto.FormDetailResponse{FormResponse: toFormResponse(f), Versions: make([]submission_form_dto.VersionSummary, 0, len(versions))}
	for _, v := range versions {
		out.Versions = append(out.Versions, toVersionSummary(v))
	}
	return out, nil
}

// ---------- Version ----------

func (s *ServiceImpl) buildVersionDetail(ctx context.Context, v submission_form_model.Version) (submission_form_dto.VersionDetailResponse, error) {
	sections, err := s.repo.ListSectionsByVersion(ctx, v.VersionID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	fields, err := s.repo.ListFieldsByVersion(ctx, v.VersionID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	options, err := s.repo.ListOptionsByVersion(ctx, v.VersionID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}

	optionsByField := map[int64][]submission_form_dto.OptionResponse{}
	for _, o := range options {
		optionsByField[o.FieldID] = append(optionsByField[o.FieldID], toOptionResponse(o))
	}
	fieldsBySection := map[int64][]submission_form_dto.FieldResponse{}
	for _, f := range fields {
		fieldsBySection[f.SectionID] = append(fieldsBySection[f.SectionID], toFieldResponse(f, optionsByField[f.FieldID]))
	}

	out := submission_form_dto.VersionDetailResponse{
		VersionID:     v.VersionID,
		FormID:        v.FormID,
		VersionNumber: v.VersionNumber,
		Status:        v.Status,
		Sections:      make([]submission_form_dto.SectionResponse, 0, len(sections)),
	}
	if v.PublishedDate.Valid {
		out.PublishedDate = &v.PublishedDate.Time
	}
	for _, sec := range sections {
		secResp := submission_form_dto.SectionResponse{
			SectionID:    sec.SectionID,
			SectionCode:  sec.SectionCode,
			SectionLabel: sec.SectionLabel,
			SortOrder:    sec.SortOrder,
			Description:  sec.Description.String,
			Fields:       fieldsBySection[sec.SectionID],
		}
		if secResp.Fields == nil {
			secResp.Fields = []submission_form_dto.FieldResponse{}
		}
		out.Sections = append(out.Sections, secResp)
	}
	return out, nil
}

func (s *ServiceImpl) CreateVersion(ctx context.Context, formID int64, req submission_form_dto.CreateVersionRequest, actorID int64) (submission_form_dto.VersionDetailResponse, error) {
	if _, err := s.repo.FindFormByID(ctx, formID); err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.NotFound("Form tidak ditemukan")
	}

	var cloneFrom submission_form_model.Version
	if req.CloneFromVersionID != nil {
		v, err := s.repo.FindVersionByID(ctx, *req.CloneFromVersionID)
		if err != nil {
			return submission_form_dto.VersionDetailResponse{}, apperror.BadRequest("Version sumber salin tidak ditemukan")
		}
		if v.FormID != formID {
			return submission_form_dto.VersionDetailResponse{}, apperror.BadRequest("Version sumber salin bukan milik form ini")
		}
		cloneFrom = v
	}

	maxNumber, err := s.repo.MaxVersionNumber(ctx, formID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	newID, err := s.repo.CreateVersion(ctx, formID, maxNumber+1, sql.NullInt64{Int64: actorID, Valid: actorID > 0})
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("Gagal membuat version")
	}

	if req.CloneFromVersionID != nil {
		if err := s.repo.CloneVersionStructure(ctx, cloneFrom.VersionID, newID); err != nil {
			return submission_form_dto.VersionDetailResponse{}, apperror.Internal("Gagal menyalin struktur version")
		}
	}

	v, err := s.repo.FindVersionByID(ctx, newID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	return s.buildVersionDetail(ctx, v)
}

func (s *ServiceImpl) GetVersion(ctx context.Context, versionID int64) (submission_form_dto.VersionDetailResponse, error) {
	v, err := s.repo.FindVersionByID(ctx, versionID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.NotFound("Version tidak ditemukan")
	}
	return s.buildVersionDetail(ctx, v)
}

func (s *ServiceImpl) GetPublishedByFormCode(ctx context.Context, formCode string) (submission_form_dto.VersionDetailResponse, error) {
	form, err := s.repo.FindFormByCode(ctx, formCode)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.NotFound("Form tidak ditemukan")
	}
	v, err := s.repo.FindPublishedVersionByForm(ctx, form.FormID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.NotFound("Form belum memiliki version yang dipublish")
	}
	return s.buildVersionDetail(ctx, v)
}

// validateScoringBeforePublish adalah gerbang akhir konsistensi scoring
// sebelum struktur version jadi immutable (design/development/
// enahnce-development-submission-dashboard/new-enhance-development.md §10):
// total weight seluruh field UseScoring harus tepat 100%, dan field AUTOMATIC
// harus punya seluruh opsi aktif berskor dengan minScore/maxScore field yang
// persis sama dengan skor opsi terendah/tertinggi (supaya normalisasi raw/max
// selalu tepat 0%-100% sesuai contoh dokumen).
func (s *ServiceImpl) validateScoringBeforePublish(ctx context.Context, versionID int64) error {
	fields, err := s.repo.ListFieldsByVersion(ctx, versionID)
	if err != nil {
		return apperror.Internal("")
	}
	options, err := s.repo.ListOptionsByVersion(ctx, versionID)
	if err != nil {
		return apperror.Internal("")
	}
	optionsByField := map[int64][]submission_form_model.Option{}
	for _, o := range options {
		optionsByField[o.FieldID] = append(optionsByField[o.FieldID], o)
	}

	var totalWeight float64
	for _, f := range fields {
		if !f.UseScoring {
			continue
		}
		if f.Weight.Valid {
			totalWeight += f.Weight.Float64
		}
		if f.ScoringMethod.String != constants.ScoringMethodAutomatic {
			continue
		}
		var minSeen, maxSeen float64
		seenAny := false
		for _, o := range optionsByField[f.FieldID] {
			if !o.IsActive {
				continue
			}
			if !o.Score.Valid {
				return apperror.Unprocessable(fmt.Sprintf("Opsi %q pada field %q belum memiliki score", o.OptionLabel, f.FieldLabel))
			}
			if !seenAny || o.Score.Float64 < minSeen {
				minSeen = o.Score.Float64
			}
			if !seenAny || o.Score.Float64 > maxSeen {
				maxSeen = o.Score.Float64
			}
			seenAny = true
		}
		if !seenAny {
			return apperror.Unprocessable(fmt.Sprintf("Field %q menggunakan scoring otomatis tapi belum memiliki opsi aktif", f.FieldLabel))
		}
		if !f.MinScore.Valid || !f.MaxScore.Valid || f.MinScore.Float64 != minSeen || f.MaxScore.Float64 != maxSeen {
			return apperror.Unprocessable(fmt.Sprintf("minScore/maxScore field %q harus persis sama dengan score opsi terendah/tertinggi (%.2f/%.2f)", f.FieldLabel, minSeen, maxSeen))
		}
	}
	if totalWeight > 0 && math.Abs(totalWeight-100) > 0.01 {
		return apperror.Unprocessable(fmt.Sprintf("Total bobot (weight) seluruh field scoring pada version ini harus 100%%, saat ini %.2f%%", totalWeight))
	}
	return nil
}

func (s *ServiceImpl) PublishVersion(ctx context.Context, versionID int64, actorID int64) (submission_form_dto.VersionDetailResponse, error) {
	v, err := s.repo.FindVersionByID(ctx, versionID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.NotFound("Version tidak ditemukan")
	}
	if v.Status != constants.FormVersionDraft {
		return submission_form_dto.VersionDetailResponse{}, apperror.Unprocessable("Hanya version berstatus DRAFT yang dapat dipublish")
	}
	if err := s.validateScoringBeforePublish(ctx, versionID); err != nil {
		return submission_form_dto.VersionDetailResponse{}, err
	}
	if err := s.repo.PublishVersion(ctx, versionID, actorID); err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	if err := s.repo.ArchiveOtherPublished(ctx, v.FormID, versionID); err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	s.audit.LogForm(ctx, auditlog.Entry{
		ActorUserID: actorID, Action: "PUBLISH", Entity: "ms_submission_form_version", EntityID: versionID,
		After: map[string]interface{}{"formID": v.FormID, "versionNumber": v.VersionNumber},
	})
	v, err = s.repo.FindVersionByID(ctx, versionID)
	if err != nil {
		return submission_form_dto.VersionDetailResponse{}, apperror.Internal("")
	}
	return s.buildVersionDetail(ctx, v)
}

// ---------- Section ----------

func (s *ServiceImpl) CreateSection(ctx context.Context, versionID int64, req submission_form_dto.CreateSectionRequest) (submission_form_dto.SectionResponse, error) {
	v, err := s.repo.FindVersionByID(ctx, versionID)
	if err != nil {
		return submission_form_dto.SectionResponse{}, apperror.NotFound("Version tidak ditemukan")
	}
	if v.Status != constants.FormVersionDraft {
		return submission_form_dto.SectionResponse{}, apperror.Unprocessable("Struktur form hanya dapat diubah selama version berstatus DRAFT")
	}
	id, err := s.repo.CreateSection(ctx, versionID, strings.TrimSpace(req.SectionCode), strings.TrimSpace(req.SectionLabel), req.SortOrder, nullString(req.Description))
	if err != nil {
		return submission_form_dto.SectionResponse{}, apperror.Conflict("Kode section sudah digunakan pada version ini")
	}
	sec, err := s.repo.FindSectionByID(ctx, id)
	if err != nil {
		return submission_form_dto.SectionResponse{}, apperror.Internal("")
	}
	return submission_form_dto.SectionResponse{
		SectionID: sec.SectionID, SectionCode: sec.SectionCode, SectionLabel: sec.SectionLabel,
		SortOrder: sec.SortOrder, Description: sec.Description.String, Fields: []submission_form_dto.FieldResponse{},
	}, nil
}

func (s *ServiceImpl) requireDraftBySection(ctx context.Context, sectionID int64) error {
	status, err := s.repo.VersionStatusBySectionID(ctx, sectionID)
	if err != nil {
		return apperror.NotFound("Section tidak ditemukan")
	}
	if status != constants.FormVersionDraft {
		return apperror.Unprocessable("Struktur form hanya dapat diubah selama version berstatus DRAFT")
	}
	return nil
}

func (s *ServiceImpl) requireDraftByField(ctx context.Context, fieldID int64) error {
	status, err := s.repo.VersionStatusByFieldID(ctx, fieldID)
	if err != nil {
		return apperror.NotFound("Field tidak ditemukan")
	}
	if status != constants.FormVersionDraft {
		return apperror.Unprocessable("Struktur form hanya dapat diubah selama version berstatus DRAFT")
	}
	return nil
}

func (s *ServiceImpl) UpdateSection(ctx context.Context, sectionID int64, req submission_form_dto.UpdateSectionRequest) (submission_form_dto.SectionResponse, error) {
	if err := s.requireDraftBySection(ctx, sectionID); err != nil {
		return submission_form_dto.SectionResponse{}, err
	}
	if err := s.repo.UpdateSection(ctx, sectionID, strings.TrimSpace(req.SectionLabel), req.SortOrder, nullString(req.Description)); err != nil {
		return submission_form_dto.SectionResponse{}, apperror.Internal("")
	}
	sec, err := s.repo.FindSectionByID(ctx, sectionID)
	if err != nil {
		return submission_form_dto.SectionResponse{}, apperror.Internal("")
	}
	return submission_form_dto.SectionResponse{
		SectionID: sec.SectionID, SectionCode: sec.SectionCode, SectionLabel: sec.SectionLabel,
		SortOrder: sec.SortOrder, Description: sec.Description.String, Fields: []submission_form_dto.FieldResponse{},
	}, nil
}

func (s *ServiceImpl) DeleteSection(ctx context.Context, sectionID int64) error {
	if err := s.requireDraftBySection(ctx, sectionID); err != nil {
		return err
	}
	if err := s.repo.DeleteSection(ctx, sectionID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// ---------- Field ----------

func (s *ServiceImpl) validateConditionalField(ctx context.Context, sectionVersionID int64, conditionalOnFieldID *int64) (sql.NullInt64, error) {
	if conditionalOnFieldID == nil {
		return sql.NullInt64{}, nil
	}
	targetVersionID, err := s.repo.VersionIDByFieldID(ctx, *conditionalOnFieldID)
	if err != nil {
		return sql.NullInt64{}, apperror.BadRequest("Field acuan kondisional tidak ditemukan")
	}
	if targetVersionID != sectionVersionID {
		return sql.NullInt64{}, apperror.BadRequest("Field acuan kondisional harus berada pada version yang sama")
	}
	return sql.NullInt64{Int64: *conditionalOnFieldID, Valid: true}, nil
}

func (s *ServiceImpl) CreateField(ctx context.Context, sectionID int64, req submission_form_dto.CreateFieldRequest) (submission_form_dto.FieldResponse, error) {
	sec, err := s.repo.FindSectionByID(ctx, sectionID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.NotFound("Section tidak ditemukan")
	}
	if err := s.requireDraftBySection(ctx, sectionID); err != nil {
		return submission_form_dto.FieldResponse{}, err
	}

	conditionalOnFieldID, err := s.validateConditionalField(ctx, sec.VersionID, req.ConditionalOnFieldID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	validationRule, err := jsonToNullString(req.ValidationRule)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	conditionalRule, err := jsonToNullString(req.ConditionalRule)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	scoringMethod, minScore, maxScore, weight, err := validateScoringConfig(req.FieldType, req.UseScoring, req.ScoringMethod, req.MinScore, req.MaxScore, req.Weight)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}

	id, err := s.repo.CreateField(ctx, submission_form_model.FieldParams{
		SectionID:            sectionID,
		FieldCode:            strings.TrimSpace(req.FieldCode),
		FieldLabel:           strings.TrimSpace(req.FieldLabel),
		FieldType:            req.FieldType,
		IsRequired:           req.IsRequired,
		SortOrder:            req.SortOrder,
		ValidationRuleJSON:   validationRule,
		ConditionalOnFieldID: conditionalOnFieldID,
		ConditionalRuleJSON:  conditionalRule,
		HelpText:             nullString(req.HelpText),
		UseScoring:           req.UseScoring,
		ScoringMethod:        scoringMethod,
		MinScore:             minScore,
		MaxScore:             maxScore,
		Weight:               weight,
	})
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.Conflict("Kode field sudah digunakan pada section ini")
	}
	f, err := s.repo.FindFieldByID(ctx, id)
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.Internal("")
	}
	return toFieldResponse(f, []submission_form_dto.OptionResponse{}), nil
}

func (s *ServiceImpl) UpdateField(ctx context.Context, fieldID int64, req submission_form_dto.UpdateFieldRequest) (submission_form_dto.FieldResponse, error) {
	existing, err := s.repo.FindFieldByID(ctx, fieldID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.NotFound("Field tidak ditemukan")
	}
	if err := s.requireDraftByField(ctx, fieldID); err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	sectionVersionID, err := s.repo.VersionIDByFieldID(ctx, fieldID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.Internal("")
	}
	if req.ConditionalOnFieldID != nil && *req.ConditionalOnFieldID == fieldID {
		return submission_form_dto.FieldResponse{}, apperror.BadRequest("Field tidak dapat bergantung pada dirinya sendiri")
	}
	conditionalOnFieldID, err := s.validateConditionalField(ctx, sectionVersionID, req.ConditionalOnFieldID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	validationRule, err := jsonToNullString(req.ValidationRule)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	conditionalRule, err := jsonToNullString(req.ConditionalRule)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}
	scoringMethod, minScore, maxScore, weight, err := validateScoringConfig(req.FieldType, req.UseScoring, req.ScoringMethod, req.MinScore, req.MaxScore, req.Weight)
	if err != nil {
		return submission_form_dto.FieldResponse{}, err
	}

	if err := s.repo.UpdateField(ctx, fieldID, submission_form_model.FieldParams{
		SectionID:            existing.SectionID,
		FieldCode:            existing.FieldCode,
		FieldLabel:           strings.TrimSpace(req.FieldLabel),
		FieldType:            req.FieldType,
		IsRequired:           req.IsRequired,
		SortOrder:            req.SortOrder,
		ValidationRuleJSON:   validationRule,
		ConditionalOnFieldID: conditionalOnFieldID,
		ConditionalRuleJSON:  conditionalRule,
		HelpText:             nullString(req.HelpText),
		UseScoring:           req.UseScoring,
		ScoringMethod:        scoringMethod,
		MinScore:             minScore,
		MaxScore:             maxScore,
		Weight:               weight,
	}); err != nil {
		return submission_form_dto.FieldResponse{}, apperror.Internal("")
	}
	f, err := s.repo.FindFieldByID(ctx, fieldID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.Internal("")
	}
	options, err := s.repo.ListOptionsByVersion(ctx, sectionVersionID)
	if err != nil {
		return submission_form_dto.FieldResponse{}, apperror.Internal("")
	}
	optResp := make([]submission_form_dto.OptionResponse, 0)
	for _, o := range options {
		if o.FieldID == fieldID {
			optResp = append(optResp, toOptionResponse(o))
		}
	}
	return toFieldResponse(f, optResp), nil
}

func (s *ServiceImpl) DeleteField(ctx context.Context, fieldID int64) error {
	if err := s.requireDraftByField(ctx, fieldID); err != nil {
		return err
	}
	if err := s.repo.DeleteField(ctx, fieldID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// ---------- Option ----------

// validateOptionScore memastikan score opsi konsisten dengan konfigurasi
// scoring field induknya: wajib diisi & berada dalam [minScore,maxScore] bila
// field memakai scoring otomatis (Single Choice); diabaikan (boleh nil) untuk
// field lain — score pada opsi field non-automatic tidak punya arti.
func validateOptionScore(field submission_form_model.Field, score *float64) error {
	if !field.UseScoring || field.ScoringMethod.String != constants.ScoringMethodAutomatic {
		return nil
	}
	if score == nil {
		return apperror.BadRequest("Score wajib diisi untuk opsi pada field dengan scoring otomatis")
	}
	if field.MinScore.Valid && *score < field.MinScore.Float64 {
		return apperror.BadRequest("Score opsi tidak boleh kurang dari minScore field")
	}
	if field.MaxScore.Valid && *score > field.MaxScore.Float64 {
		return apperror.BadRequest("Score opsi tidak boleh lebih dari maxScore field")
	}
	return nil
}

func (s *ServiceImpl) CreateOption(ctx context.Context, fieldID int64, req submission_form_dto.CreateOptionRequest) (submission_form_dto.OptionResponse, error) {
	field, err := s.repo.FindFieldByID(ctx, fieldID)
	if err != nil {
		return submission_form_dto.OptionResponse{}, apperror.NotFound("Field tidak ditemukan")
	}
	if err := s.requireDraftByField(ctx, fieldID); err != nil {
		return submission_form_dto.OptionResponse{}, err
	}
	if err := validateOptionScore(field, req.Score); err != nil {
		return submission_form_dto.OptionResponse{}, err
	}
	id, err := s.repo.CreateOption(ctx, fieldID, strings.TrimSpace(req.OptionValue), strings.TrimSpace(req.OptionLabel), req.SortOrder, nullFloat(req.Score))
	if err != nil {
		return submission_form_dto.OptionResponse{}, apperror.Conflict("Nilai pilihan sudah digunakan pada field ini")
	}
	o, err := s.repo.FindOptionByID(ctx, id)
	if err != nil {
		return submission_form_dto.OptionResponse{}, apperror.Internal("")
	}
	return toOptionResponse(o), nil
}

func (s *ServiceImpl) UpdateOption(ctx context.Context, optionID int64, req submission_form_dto.UpdateOptionRequest) (submission_form_dto.OptionResponse, error) {
	opt, err := s.repo.FindOptionByID(ctx, optionID)
	if err != nil {
		return submission_form_dto.OptionResponse{}, apperror.NotFound("Pilihan tidak ditemukan")
	}
	if err := s.requireDraftByField(ctx, opt.FieldID); err != nil {
		return submission_form_dto.OptionResponse{}, err
	}
	field, err := s.repo.FindFieldByID(ctx, opt.FieldID)
	if err != nil {
		return submission_form_dto.OptionResponse{}, apperror.Internal("")
	}
	if err := validateOptionScore(field, req.Score); err != nil {
		return submission_form_dto.OptionResponse{}, err
	}
	if err := s.repo.UpdateOption(ctx, optionID, strings.TrimSpace(req.OptionValue), strings.TrimSpace(req.OptionLabel), req.SortOrder, req.IsActive, nullFloat(req.Score)); err != nil {
		return submission_form_dto.OptionResponse{}, apperror.Internal("")
	}
	o, err := s.repo.FindOptionByID(ctx, optionID)
	if err != nil {
		return submission_form_dto.OptionResponse{}, apperror.Internal("")
	}
	return toOptionResponse(o), nil
}

func (s *ServiceImpl) DeleteOption(ctx context.Context, optionID int64) error {
	opt, err := s.repo.FindOptionByID(ctx, optionID)
	if err != nil {
		return apperror.NotFound("Pilihan tidak ditemukan")
	}
	if err := s.requireDraftByField(ctx, opt.FieldID); err != nil {
		return err
	}
	if err := s.repo.DeleteOption(ctx, optionID); err != nil {
		return apperror.Internal("")
	}
	return nil
}
