package dynamicform_repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_model"

	"gorm.io/gorm"
)

// ErrLockBusy is returned by WithAdvisoryLock when the lock is already held.
var ErrLockBusy = errors.New("dynamicform: advisory lock busy")

// RepositoryImpl is the GORM-based Repository implementation.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository creates the dynamicform Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) DB() *gorm.DB { return r.db }

const formSelect = "f.*, u.fullName AS creatorName"

// ---------------------------------------------------------------------------
// forms
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) GetByID(ctx context.Context, id int64) (dynamicform_model.Form, error) {
	var f dynamicform_model.Form
	err := r.db.WithContext(ctx).Table(constants.TableDynamicForm+" f").
		Joins("LEFT JOIN ms_user u ON u.userID = f.createdBy").
		Select(formSelect).Where("f.formID = ?", id).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dynamicform_model.Form{}, ErrNotFound
	}
	return f, err
}

func (r *RepositoryImpl) GetBySlug(ctx context.Context, slug string) (dynamicform_model.Form, error) {
	var f dynamicform_model.Form
	err := r.db.WithContext(ctx).Table(constants.TableDynamicForm+" f").
		Joins("LEFT JOIN ms_user u ON u.userID = f.createdBy").
		Select(formSelect).Where("f.slug = ? AND f.isActive = 1", slug).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dynamicform_model.Form{}, ErrNotFound
	}
	return f, err
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableDynamicForm).
		Where("slug = ? AND formID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) CreateForm(ctx context.Context, values map[string]any) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableDynamicForm).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) UpdateForm(ctx context.Context, id int64, values map[string]any) error {
	return r.db.WithContext(ctx).Table(constants.TableDynamicForm).
		Where("formID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) SoftDeleteForm(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Table(constants.TableDynamicForm).
		Where("formID = ?", id).Updates(map[string]any{"isActive": 0, "updatedDate": time.Now()}).Error
}

func (r *RepositoryImpl) PurgeFormChildren(ctx context.Context, formID int64) ([]string, error) {
	var urls []string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableDynamicFormFile+" tf").
			Joins("JOIN "+constants.TableDynamicFormSubmission+" s ON s.submissionID = tf.submissionID").
			Where("s.formID = ?", formID).Pluck("tf.fileURL", &urls).Error; err != nil {
			return err
		}
		// submissions first (answers + files cascade off submissionID)
		if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormSubmission+" WHERE formID = ?", formID).Error; err != nil {
			return err
		}
		for _, table := range []string{
			constants.TableDynamicFormDraft,
			constants.TableDynamicFormField,
			constants.TableDynamicFormSection,
			constants.TableDynamicFormCollaborator,
		} {
			if err := tx.Exec("DELETE FROM "+table+" WHERE formID = ?", formID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return urls, err
}

var formSortColumns = map[string]string{
	"title":           "f.title",
	"status":          "f.status",
	"totalSubmission": "f.totalSubmission",
	"createdDate":     "f.createdDate",
}

func (r *RepositoryImpl) ListForms(ctx context.Context, f dynamicform_dto.FormFilter) ([]dynamicform_model.Form, int64, []int, error) {
	q := r.db.WithContext(ctx).Table(constants.TableDynamicForm + " f").
		Joins("LEFT JOIN ms_user u ON u.userID = f.createdBy").
		Where("f.isActive = 1")
	if f.Search != "" {
		q = q.Where("f.title LIKE ?", "%"+f.Search+"%")
	}
	if f.Status != "" {
		q = q.Where("f.status = ?", f.Status)
	}
	if f.DateFrom != "" {
		q = q.Where("f.createdDate >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("f.createdDate <= ?", f.DateTo)
	}
	if f.MineOnly {
		q = q.Where("(f.createdBy = ? OR f.formID IN (SELECT formID FROM "+constants.TableDynamicFormCollaborator+" WHERE userID = ?))",
			f.ActorID, f.ActorID)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, nil, err
	}

	orderBy := f.OrderBy
	if orderBy == "" {
		orderBy = "f.createdDate DESC"
	}
	var forms []dynamicform_model.Form
	if err := q.Select(formSelect).Order(orderBy).Limit(f.Limit).Offset(f.Offset).Find(&forms).Error; err != nil {
		return nil, 0, nil, err
	}

	counts := make([]int, len(forms))
	for i, form := range forms {
		var c int64
		_ = r.db.WithContext(ctx).Table(constants.TableDynamicFormField).
			Where("formID = ? AND isActive = 1 AND fieldType NOT IN ?", form.FormID, constants.DynamicFormDisplayFieldTypes).
			Count(&c).Error
		counts[i] = int(c)
	}
	return forms, total, counts, nil
}

// ---------------------------------------------------------------------------
// collaborators
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) ListCollaborators(ctx context.Context, formID int64) ([]dynamicform_model.Collaborator, error) {
	var out []dynamicform_model.Collaborator
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormCollaborator+" c").
		Joins("JOIN ms_user u ON u.userID = c.userID").
		Select("c.*, u.fullName AS userName, u.email AS userEmail").
		Where("c.formID = ?", formID).Order("c.addedDate ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) ReplaceCollaborators(ctx context.Context, formID int64, rows []dynamicform_dto.CollaboratorInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormCollaborator+" WHERE formID = ?", formID).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if row.UserID == 0 {
				continue
			}
			if err := tx.Table(constants.TableDynamicFormCollaborator).Create(map[string]any{
				"formID": formID, "userID": row.UserID, "role": row.Role, "addedDate": time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryImpl) IsCollaborator(ctx context.Context, formID, userID int64, roles ...string) (bool, error) {
	q := r.db.WithContext(ctx).Table(constants.TableDynamicFormCollaborator).
		Where("formID = ? AND userID = ?", formID, userID)
	if len(roles) > 0 {
		q = q.Where("role IN ?", roles)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

// ---------------------------------------------------------------------------
// sections & fields
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) ListSections(ctx context.Context, formID int64) ([]dynamicform_model.Section, error) {
	var out []dynamicform_model.Section
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSection).
		Where("formID = ? AND isActive = 1", formID).Order("sortOrder ASC, sectionID ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) ListFields(ctx context.Context, formID int64, activeOnly bool) ([]dynamicform_model.Field, error) {
	q := r.db.WithContext(ctx).Table(constants.TableDynamicFormField).Where("formID = ?", formID)
	if activeOnly {
		q = q.Where("isActive = 1")
	}
	var out []dynamicform_model.Field
	err := q.Order("sortOrder ASC, fieldID ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) GetField(ctx context.Context, formID, fieldID int64) (dynamicform_model.Field, error) {
	var f dynamicform_model.Field
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormField).
		Where("formID = ? AND fieldID = ?", formID, fieldID).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dynamicform_model.Field{}, ErrNotFound
	}
	return f, err
}

// AddField computes MAX(sortOrder)+1 and inserts atomically so a double-click
// cannot create two rows racing for the same order.
func (r *RepositoryImpl) AddField(ctx context.Context, formID int64, values map[string]any) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxOrder int
		if err := tx.Raw("SELECT COALESCE(MAX(sortOrder), -1) FROM "+constants.TableDynamicFormField+" WHERE formID = ?", formID).
			Scan(&maxOrder).Error; err != nil {
			return err
		}
		values["formID"] = formID
		values["sortOrder"] = maxOrder + 1
		values["createdDate"] = time.Now()
		if err := tx.Table(constants.TableDynamicFormField).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) UpdateField(ctx context.Context, formID, fieldID int64, values map[string]any) error {
	values["updatedDate"] = time.Now()
	return r.db.WithContext(ctx).Table(constants.TableDynamicFormField).
		Where("formID = ? AND fieldID = ?", formID, fieldID).Updates(values).Error
}

func (r *RepositoryImpl) SoftDeleteField(ctx context.Context, formID, fieldID int64) error {
	return r.db.WithContext(ctx).Table(constants.TableDynamicFormField).
		Where("formID = ? AND fieldID = ?", formID, fieldID).
		Updates(map[string]any{"isActive": 0, "updatedDate": time.Now()}).Error
}

func (r *RepositoryImpl) ReorderFields(ctx context.Context, formID int64, order []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for idx, fieldID := range order {
			if err := tx.Table(constants.TableDynamicFormField).
				Where("formID = ? AND fieldID = ?", formID, fieldID).
				Update("sortOrder", idx).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// submissions
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) CountSubmissionsSince(ctx context.Context, formID int64, ip string, since time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("formID = ? AND ipAddress = ? AND submittedDate >= ?", formID, ip, since).Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) OldestSubmissionSince(ctx context.Context, formID int64, ip string, since time.Time) (time.Time, bool, error) {
	var t time.Time
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Select("MIN(submittedDate)").
		Where("formID = ? AND ipAddress = ? AND submittedDate >= ?", formID, ip, since).Scan(&t).Error
	if err != nil {
		return time.Time{}, false, err
	}
	return t, !t.IsZero(), nil
}

func (r *RepositoryImpl) HasSubmitted(ctx context.Context, formID int64, email string, userID *int64) (bool, error) {
	q := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).Where("formID = ?", formID)
	if userID != nil {
		q = q.Where("respondentUserID = ?", *userID)
	} else {
		q = q.Where("respondentEmail = ?", email)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Submit(ctx context.Context, in SubmitData) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sub := map[string]any{
			"formID": in.FormID, "respondentEmail": in.RespondentEmail, "respondentName": in.RespondentName,
			"respondentUserID": in.RespondentUserID, "ipAddress": in.IPAddress, "userAgent": in.UserAgent,
			"isValid": in.IsValid, "formVersion": in.FormVersion, "submittedDate": time.Now(),
		}
		if err := tx.Table(constants.TableDynamicFormSubmission).Create(sub).Error; err != nil {
			return err
		}
		if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error; err != nil {
			return err
		}
		for fieldID, val := range in.Answers {
			v := val
			if err := tx.Table(constants.TableDynamicFormAnswer).Create(map[string]any{
				"submissionID": newID, "fieldID": fieldID, "answerValue": v,
			}).Error; err != nil {
				return err
			}
		}
		for fieldID, file := range in.Files {
			if err := tx.Table(constants.TableDynamicFormFile).Create(map[string]any{
				"submissionID": newID, "fieldID": fieldID, "fileURL": file.FileURL,
				"originalFileName": file.OriginalFileName, "mimeType": file.MimeType,
				"fileSizeKB": file.FileSizeKB, "createdDate": time.Now(),
			}).Error; err != nil {
				return err
			}
			// Keep answerValue = fileURL so CSV / rekap stay uniform.
			if err := tx.Table(constants.TableDynamicFormAnswer).Create(map[string]any{
				"submissionID": newID, "fieldID": fieldID, "answerValue": file.FileURL,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("UPDATE "+constants.TableDynamicForm+
			" SET totalSubmission = totalSubmission + 1 WHERE formID = ?", in.FormID).Error; err != nil {
			return err
		}
		if in.DeleteDraftUserID != nil {
			if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormDraft+
				" WHERE formID = ? AND userID = ?", in.FormID, *in.DeleteDraftUserID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return newID, err
}

var submissionSortDefault = "s.submittedDate DESC"

func (r *RepositoryImpl) ListSubmissions(ctx context.Context, formID int64, f dynamicform_dto.SubmissionFilter) ([]dynamicform_model.Submission, int64, error) {
	q := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission+" s").Where("s.formID = ?", formID)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(s.respondentEmail LIKE ? OR s.respondentName LIKE ?)", like, like)
	}
	if f.ValidOnly {
		q = q.Where("s.isValid = 1")
	}
	if f.DateFrom != "" {
		q = q.Where("s.submittedDate >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("s.submittedDate <= ?", f.DateTo)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []dynamicform_model.Submission
	err := q.Select("s.*").Order(submissionSortDefault).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) AllSubmissionsAsc(ctx context.Context, formID int64) ([]dynamicform_model.Submission, error) {
	var out []dynamicform_model.Submission
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("formID = ?", formID).Order("submittedDate ASC, submissionID ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) GetSubmission(ctx context.Context, formID, submissionID int64) (dynamicform_model.Submission, error) {
	var s dynamicform_model.Submission
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("formID = ? AND submissionID = ?", formID, submissionID).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dynamicform_model.Submission{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) AnswersFor(ctx context.Context, submissionIDs []int64) (map[int64]map[int64]string, error) {
	out := map[int64]map[int64]string{}
	if len(submissionIDs) == 0 {
		return out, nil
	}
	var rows []dynamicform_model.Answer
	if err := r.db.WithContext(ctx).Table(constants.TableDynamicFormAnswer).
		Where("submissionID IN ?", submissionIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, a := range rows {
		if out[a.SubmissionID] == nil {
			out[a.SubmissionID] = map[int64]string{}
		}
		v := ""
		if a.AnswerValue != nil {
			v = *a.AnswerValue
		}
		out[a.SubmissionID][a.FieldID] = v
	}
	return out, nil
}

func (r *RepositoryImpl) FilesFor(ctx context.Context, submissionIDs []int64) (map[int64][]dynamicform_model.File, error) {
	out := map[int64][]dynamicform_model.File{}
	if len(submissionIDs) == 0 {
		return out, nil
	}
	var rows []dynamicform_model.File
	if err := r.db.WithContext(ctx).Table(constants.TableDynamicFormFile).
		Where("submissionID IN ?", submissionIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, f := range rows {
		out[f.SubmissionID] = append(out[f.SubmissionID], f)
	}
	return out, nil
}

func (r *RepositoryImpl) EditSubmission(ctx context.Context, in EditData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for fieldID, val := range in.Answers {
			v := val
			var count int64
			if err := tx.Table(constants.TableDynamicFormAnswer).
				Where("submissionID = ? AND fieldID = ?", in.SubmissionID, fieldID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				if err := tx.Table(constants.TableDynamicFormAnswer).
					Where("submissionID = ? AND fieldID = ?", in.SubmissionID, fieldID).
					Update("answerValue", v).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Table(constants.TableDynamicFormAnswer).Create(map[string]any{
					"submissionID": in.SubmissionID, "fieldID": fieldID, "answerValue": v,
				}).Error; err != nil {
					return err
				}
			}
		}
		for fieldID, file := range in.ReplacedFiles {
			if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormFile+
				" WHERE submissionID = ? AND fieldID = ?", in.SubmissionID, fieldID).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormAnswer+
				" WHERE submissionID = ? AND fieldID = ?", in.SubmissionID, fieldID).Error; err != nil {
				return err
			}
			if err := tx.Table(constants.TableDynamicFormFile).Create(map[string]any{
				"submissionID": in.SubmissionID, "fieldID": fieldID, "fileURL": file.FileURL,
				"originalFileName": file.OriginalFileName, "mimeType": file.MimeType,
				"fileSizeKB": file.FileSizeKB, "createdDate": time.Now(),
			}).Error; err != nil {
				return err
			}
			if err := tx.Table(constants.TableDynamicFormAnswer).Create(map[string]any{
				"submissionID": in.SubmissionID, "fieldID": fieldID, "answerValue": file.FileURL,
			}).Error; err != nil {
				return err
			}
		}
		return tx.Table(constants.TableDynamicFormSubmission).
			Where("submissionID = ?", in.SubmissionID).
			Updates(map[string]any{"respondentEmail": in.RespondentEmail, "respondentName": in.RespondentName}).Error
	})
}

func (r *RepositoryImpl) DeleteSubmission(ctx context.Context, formID, submissionID int64) ([]string, *int, error) {
	var urls []string
	var newTotal int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableDynamicFormFile).
			Where("submissionID = ?", submissionID).Pluck("fileURL", &urls).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormSubmission+
			" WHERE submissionID = ? AND formID = ?", submissionID, formID).Error; err != nil {
			return err
		}
		var c int64
		if err := tx.Table(constants.TableDynamicFormSubmission).Where("formID = ?", formID).Count(&c).Error; err != nil {
			return err
		}
		newTotal = int(c)
		return tx.Table(constants.TableDynamicForm).Where("formID = ?", formID).
			Update("totalSubmission", newTotal).Error
	})
	return urls, &newTotal, err
}

func (r *RepositoryImpl) DeleteAllSubmissions(ctx context.Context, formID int64) ([]string, error) {
	var urls []string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableDynamicFormFile+" tf").
			Joins("JOIN "+constants.TableDynamicFormSubmission+" s ON s.submissionID = tf.submissionID").
			Where("s.formID = ?", formID).Pluck("tf.fileURL", &urls).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM "+constants.TableDynamicFormSubmission+" WHERE formID = ?", formID).Error; err != nil {
			return err
		}
		return tx.Table(constants.TableDynamicForm).Where("formID = ?", formID).
			Update("totalSubmission", 0).Error
	})
	return urls, err
}

func (r *RepositoryImpl) SetGsheetRowIndex(ctx context.Context, submissionID int64, rowIndex int) error {
	return r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("submissionID = ?", submissionID).Update("gsheetRowIndex", rowIndex).Error
}

func (r *RepositoryImpl) DecrementRowIndexesAfter(ctx context.Context, formID int64, rowIndex int) error {
	return r.db.WithContext(ctx).Exec("UPDATE "+constants.TableDynamicFormSubmission+
		" SET gsheetRowIndex = gsheetRowIndex - 1 WHERE formID = ? AND gsheetRowIndex > ?", formID, rowIndex).Error
}

func (r *RepositoryImpl) TouchGsheetSync(ctx context.Context, formID int64, syncErr string) error {
	values := map[string]any{"gsheetLastSyncDate": time.Now()}
	if syncErr == "" {
		values["gsheetLastSyncError"] = nil
	} else {
		if len(syncErr) > 480 {
			syncErr = syncErr[:480]
		}
		values["gsheetLastSyncError"] = syncErr
	}
	return r.db.WithContext(ctx).Table(constants.TableDynamicForm).
		Where("formID = ?", formID).Updates(values).Error
}

// ---------------------------------------------------------------------------
// drafts
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) GetDraft(ctx context.Context, formID, userID int64) (dynamicform_model.Draft, bool, error) {
	var d dynamicform_model.Draft
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormDraft).
		Where("formID = ? AND userID = ?", formID, userID).Take(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dynamicform_model.Draft{}, false, nil
	}
	if err != nil {
		return dynamicform_model.Draft{}, false, err
	}
	return d, true, nil
}

// MutateDraft serialises draft mutations per (formID, userID) with a row lock:
// it ensures the row exists, locks it FOR UPDATE, hands the decoded answers map
// to mutate, then writes the result back (or deletes the row when empty).
func (r *RepositoryImpl) MutateDraft(ctx context.Context, formID, userID int64, mutate func(map[string]any) (map[string]any, error)) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("INSERT INTO "+constants.TableDynamicFormDraft+
			" (formID, userID, answersJSON, createdDate, updatedDate) VALUES (?, ?, '{}', NOW(), NOW())"+
			" ON DUPLICATE KEY UPDATE draftID = draftID", formID, userID).Error; err != nil {
			return err
		}
		var raw string
		if err := tx.Raw("SELECT answersJSON FROM "+constants.TableDynamicFormDraft+
			" WHERE formID = ? AND userID = ? FOR UPDATE", formID, userID).Scan(&raw).Error; err != nil {
			return err
		}
		current := map[string]any{}
		if raw != "" {
			_ = json.Unmarshal([]byte(raw), &current)
		}
		next, err := mutate(current)
		if err != nil {
			return err
		}
		if len(next) == 0 {
			return tx.Exec("DELETE FROM "+constants.TableDynamicFormDraft+
				" WHERE formID = ? AND userID = ?", formID, userID).Error
		}
		b, err := json.Marshal(next)
		if err != nil {
			return err
		}
		return tx.Exec("UPDATE "+constants.TableDynamicFormDraft+
			" SET answersJSON = ?, updatedDate = NOW() WHERE formID = ? AND userID = ?",
			string(b), formID, userID).Error
	})
}

func (r *RepositoryImpl) DeleteDraft(ctx context.Context, formID, userID int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM "+constants.TableDynamicFormDraft+
		" WHERE formID = ? AND userID = ?", formID, userID).Error
}

func (r *RepositoryImpl) StaleDrafts(ctx context.Context, olderThan time.Time) ([]dynamicform_model.Draft, error) {
	var out []dynamicform_model.Draft
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormDraft).
		Where("updatedDate < ?", olderThan).Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) DeleteDraftByID(ctx context.Context, draftID int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM "+constants.TableDynamicFormDraft+" WHERE draftID = ?", draftID).Error
}

// ---------------------------------------------------------------------------
// analytics
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) SubmissionsPerDay(ctx context.Context, formID int64, since time.Time) (map[string]int, error) {
	var rows []struct {
		D string
		C int
	}
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Select("DATE(submittedDate) AS d, COUNT(*) AS c").
		Where("formID = ? AND submittedDate >= ?", formID, since).
		Group("DATE(submittedDate)").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, row := range rows {
		out[strings.SplitN(row.D, "T", 2)[0]] = row.C
	}
	return out, nil
}

func (r *RepositoryImpl) ValidCounts(ctx context.Context, formID int64) (int, int, error) {
	var valid, total int64
	if err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("formID = ?", formID).Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("formID = ? AND isValid = 1", formID).Count(&valid).Error; err != nil {
		return 0, 0, err
	}
	return int(valid), int(total - valid), nil
}

func (r *RepositoryImpl) TotalFiles(ctx context.Context, formID int64) (int, error) {
	var c int64
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormFile+" tf").
		Joins("JOIN "+constants.TableDynamicFormSubmission+" s ON s.submissionID = tf.submissionID").
		Where("s.formID = ?", formID).Count(&c).Error
	return int(c), err
}

func (r *RepositoryImpl) RecentSubmissions(ctx context.Context, formID int64, limit int) ([]dynamicform_model.Submission, error) {
	var out []dynamicform_model.Submission
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("formID = ?", formID).Order("submittedDate DESC").Limit(limit).Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) AnswerValueCounts(ctx context.Context, formID, fieldID int64) ([]ValueCount, error) {
	var rows []ValueCount
	err := r.db.WithContext(ctx).Table(constants.TableDynamicFormAnswer+" a").
		Joins("JOIN "+constants.TableDynamicFormSubmission+" s ON s.submissionID = a.submissionID").
		Select("a.answerValue AS value, COUNT(*) AS count").
		Where("s.formID = ? AND a.fieldID = ? AND a.answerValue IS NOT NULL AND a.answerValue <> ''", formID, fieldID).
		Group("a.answerValue").Scan(&rows).Error
	return rows, err
}

// ---------------------------------------------------------------------------
// sweep
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) ReferencedFileURLs(ctx context.Context) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	var fileURLs []string
	if err := r.db.WithContext(ctx).Table(constants.TableDynamicFormFile).Pluck("fileURL", &fileURLs).Error; err != nil {
		return nil, err
	}
	for _, u := range fileURLs {
		out[u] = struct{}{}
	}
	var drafts []string
	if err := r.db.WithContext(ctx).Table(constants.TableDynamicFormDraft).Pluck("answersJSON", &drafts).Error; err != nil {
		return nil, err
	}
	for _, raw := range drafts {
		m := map[string]json.RawMessage{}
		if json.Unmarshal([]byte(raw), &m) != nil {
			continue
		}
		for _, entry := range m {
			var fe struct {
				File    bool   `json:"__file"`
				FileURL string `json:"fileURL"`
			}
			if json.Unmarshal(entry, &fe) == nil && fe.File && fe.FileURL != "" {
				out[fe.FileURL] = struct{}{}
			}
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// concurrency
// ---------------------------------------------------------------------------

func (r *RepositoryImpl) WithAdvisoryLock(ctx context.Context, key string, timeoutSeconds int, fn func() error) error {
	conn, err := r.db.WithContext(ctx).DB()
	if err != nil {
		return err
	}
	c, err := conn.Conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	var got int
	if err := c.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", key, timeoutSeconds).Scan(&got); err != nil {
		return err
	}
	if got != 1 {
		return ErrLockBusy
	}
	defer func() { _, _ = c.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", key) }()
	return fn()
}
