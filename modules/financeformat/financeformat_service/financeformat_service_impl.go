package financeformat_service

import (
	"context"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/slug"
	"fsldk-api/modules/financeformat/financeformat_dto"
	"fsldk-api/modules/financeformat/financeformat_model"
	"fsldk-api/modules/financeformat/financeformat_repository"
)

// Contact-person setting keys — one source of truth shared with the seed
// migration (0019_financeformat.up.sql).
const (
	settingGroupFinanceFormat = "format_keuangan"
	settingKeyCpName          = "finance_format_cp_name"
	settingKeyCpWhatsapp      = "finance_format_cp_whatsapp"
)

// maxFileSize caps an uploaded template at 10 MB — stricter than the shared
// upload endpoint's 20 MB document limit, and specific to this module.
const maxFileSize = 10 << 20

// cmsSortColumns is the whitelist of sortable columns for the CMS listing.
var cmsSortColumns = map[string]string{
	"fileName":    "f.fileName",
	"createdDate": "f.createdDate",
}

// FileStore is the narrow slice of pkg/upload.Uploader this service needs —
// accepting an interface keeps the module decoupled from the uploader impl.
type FileStore interface {
	DeleteFile(publicURL string) error
	FileSize(publicURL string) (int64, error)
	LocalPath(publicURL string) string
}

// SettingReader is the narrow slice of setting_service.Service this service
// reads the optional contact-person card from.
type SettingReader interface {
	GetValue(ctx context.Context, group, key string) (string, error)
}

// ServiceImpl is the Service implementation.
type ServiceImpl struct {
	repo    financeformat_repository.Repository
	upload  FileStore
	setting SettingReader
}

// NewService creates the financeformat Service.
func NewService(repo financeformat_repository.Repository, upload FileStore, setting SettingReader) Service {
	return &ServiceImpl{repo: repo, upload: upload, setting: setting}
}

func (s *ServiceImpl) PublicList(ctx context.Context) (financeformat_dto.PublicListResponse, error) {
	types, err := s.repo.ListFormatTypes(ctx)
	if err != nil {
		return financeformat_dto.PublicListResponse{}, apperror.Internal("")
	}
	formats, _, err := s.repo.List(ctx, financeformat_dto.Filter{
		ActiveOnly: true,
		// Limit 0 = no pagination: the public page renders every active file
		// under its category, so return them all in one shot.
		OrderBy: "f.formatTypeID ASC, f.createdDate DESC",
	})
	if err != nil {
		return financeformat_dto.PublicListResponse{}, apperror.Internal("")
	}
	if types == nil {
		types = []financeformat_model.FormatType{}
	}
	if formats == nil {
		formats = []financeformat_model.FinanceFormat{}
	}

	// Contact-person card is optional: a missing/empty setting is not an error,
	// the frontend just hides the card.
	cpName, _ := s.setting.GetValue(ctx, settingGroupFinanceFormat, settingKeyCpName)
	cpPhone, _ := s.setting.GetValue(ctx, settingGroupFinanceFormat, settingKeyCpWhatsapp)

	return financeformat_dto.PublicListResponse{
		FormatTypes: types,
		Formats:     formats,
		CpName:      cpName,
		CpPhone:     cpPhone,
	}, nil
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, f financeformat_dto.Filter) ([]financeformat_model.FinanceFormat, int, error) {
	f.ActiveOnly = false
	f.Limit = q.Limit
	f.Offset = q.Offset()
	f.OrderBy = q.OrderBy(cmsSortColumns, "f.createdDate DESC")
	if f.Search == "" {
		f.Search = q.Search
	}
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []financeformat_model.FinanceFormat{}
	}
	return data, int(total), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (financeformat_model.FinanceFormat, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return financeformat_model.FinanceFormat{}, apperror.NotFound("Format keuangan tidak ditemukan")
	}
	return m, nil
}

// PrepareDownload resolves an active format to its on-disk path plus the
// filename to hand the browser: the user's fileName as a kebab-case slug with
// a .xlsx extension (the stored fileName itself is left exactly as typed).
func (s *ServiceImpl) PrepareDownload(ctx context.Context, id int64) (localPath string, downloadName string, err error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil || !m.IsActive {
		return "", "", apperror.NotFound("Format keuangan tidak ditemukan")
	}
	if _, statErr := s.upload.FileSize(m.FileURL); statErr != nil {
		return "", "", apperror.NotFound("Berkas tidak ditemukan")
	}
	base := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(m.FileName)), ".xlsx")
	return s.upload.LocalPath(m.FileURL), slug.Make(base) + ".xlsx", nil
}

func (s *ServiceImpl) FormatTypes(ctx context.Context) ([]financeformat_model.FormatType, error) {
	data, err := s.repo.ListFormatTypes(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []financeformat_model.FormatType{}
	}
	return data, nil
}

func (s *ServiceImpl) Create(ctx context.Context, req financeformat_dto.Request, actorID int64) (financeformat_model.FinanceFormat, error) {
	if err := s.validate(ctx, req); err != nil {
		return financeformat_model.FinanceFormat{}, err
	}
	id, err := s.repo.Create(ctx, s.fromRequest(req), actorID)
	if err != nil {
		return financeformat_model.FinanceFormat{}, apperror.Internal("Gagal menyimpan format keuangan")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req financeformat_dto.Request, actorID int64) (financeformat_model.FinanceFormat, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return financeformat_model.FinanceFormat{}, apperror.NotFound("Format keuangan tidak ditemukan")
	}
	if err := s.validate(ctx, req); err != nil {
		return financeformat_model.FinanceFormat{}, err
	}
	if err := s.repo.Update(ctx, id, s.fromRequest(req), actorID); err != nil {
		return financeformat_model.FinanceFormat{}, apperror.Internal("")
	}
	// Best-effort cleanup of the replaced file, only after the DB update
	// succeeds and the URL actually changed.
	if existing.FileURL != req.FileURL {
		_ = s.upload.DeleteFile(existing.FileURL)
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Format keuangan tidak ditemukan")
	}
	if err := s.repo.SetActive(ctx, id, isActive, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Format keuangan tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	// DB row gone first, then remove the file from disk (best-effort).
	_ = s.upload.DeleteFile(existing.FileURL)
	return nil
}

// validate enforces the module-specific rules: the uploaded file must be an
// Excel workbook no larger than maxFileSize, and the chosen category must be
// one of the 9 seeded types.
func (s *ServiceImpl) validate(ctx context.Context, req financeformat_dto.Request) error {
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(req.FileURL)), ".xlsx") {
		return apperror.BadRequest("Berkas wajib berformat Excel (.xlsx)")
	}
	// Best-effort: if the file can be resolved on disk, reject it when it
	// exceeds this module's 10 MB cap (the shared upload endpoint only
	// enforces its own looser 20 MB document limit).
	if size, err := s.upload.FileSize(req.FileURL); err == nil && size > maxFileSize {
		return apperror.BadRequest("Ukuran berkas melebihi 10MB")
	}
	ok, err := s.repo.FormatTypeExists(ctx, req.FormatTypeID)
	if err != nil {
		return apperror.Internal("")
	}
	if !ok {
		return apperror.BadRequest("Kategori format tidak valid")
	}
	return nil
}

func (s *ServiceImpl) fromRequest(req financeformat_dto.Request) financeformat_model.FinanceFormat {
	return financeformat_model.FinanceFormat{
		FileName:     strings.TrimSpace(req.FileName),
		FileURL:      strings.TrimSpace(req.FileURL),
		FormatTypeID: req.FormatTypeID,
	}
}
