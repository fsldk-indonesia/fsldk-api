package submission_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"
	"fsldk-api/modules/role/role_model"
	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
)

// ---------- pure allow-list table test (no fakes needed) ----------

func TestPengurusRoleAllowed(t *testing.T) {
	tests := []struct {
		name     string
		orgType  string
		roleName string
		want     bool
	}{
		{"LDK caller can assign Kader", constants.OrgTypeLDK, roleNameKader, true},
		{"LDK caller can assign LDK Admin", constants.OrgTypeLDK, constants.RoleLDKAdmin, true},
		{"LDK caller cannot assign Puskomda Verifikator", constants.OrgTypeLDK, constants.RolePuskomdaVerifikator, false},
		{"LDK caller cannot assign Puskomnas Verifikator", constants.OrgTypeLDK, constants.RolePuskomnasVerifikator, false},
		{"Puskomda caller can assign Kader", constants.OrgTypePuskomda, roleNameKader, true},
		{"Puskomda caller can assign LDK Admin", constants.OrgTypePuskomda, constants.RoleLDKAdmin, true},
		{"Puskomda caller can assign Puskomda Verifikator", constants.OrgTypePuskomda, constants.RolePuskomdaVerifikator, true},
		{"Puskomda caller cannot assign Puskomnas Verifikator", constants.OrgTypePuskomda, constants.RolePuskomnasVerifikator, false},
		{"Puskomnas caller can assign Kader", constants.OrgTypePuskomnas, roleNameKader, true},
		{"Puskomnas caller can assign LDK Admin", constants.OrgTypePuskomnas, constants.RoleLDKAdmin, true},
		{"Puskomnas caller can assign Puskomda Verifikator", constants.OrgTypePuskomnas, constants.RolePuskomdaVerifikator, true},
		{"Puskomnas caller can assign Puskomnas Verifikator", constants.OrgTypePuskomnas, constants.RolePuskomnasVerifikator, true},
		{"unknown org type allows nothing", "", constants.RoleLDKAdmin, false},
		{"unrecognized role name is never allowed", constants.OrgTypePuskomnas, "Super Admin", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pengurusRoleAllowed(tt.orgType, tt.roleName); got != tt.want {
				t.Errorf("pengurusRoleAllowed(%q, %q) = %v, want %v", tt.orgType, tt.roleName, got, tt.want)
			}
		})
	}
}

// ---------- fakes for ServiceImpl's 6 dependencies ----------
// Only methods exercised by this file's scenarios behave meaningfully;
// every other method is a no-op stub required to satisfy the interface
// (same approach as user_service_impl_test.go's fakeUserRepository).

type fakeSubmissionRepo struct {
	kaders         map[int64]submission_model.Kader // by kaderID
	kaderStatusSet map[int64]string                 // kaderID -> last status written by UpdateKaderStatus
}

func (f *fakeSubmissionRepo) Create(ctx context.Context, p submission_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) FindByID(ctx context.Context, id int64) (submission_model.Submission, error) {
	return submission_model.Submission{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindByOwnerOrganization(ctx context.Context, organizationID, formID int64) (submission_model.Submission, error) {
	return submission_model.Submission{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindByUserAndForm(ctx context.Context, userID, formID int64) (submission_model.Submission, error) {
	return submission_model.Submission{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) List(ctx context.Context, filt submission_dto.ListFilter) ([]submission_model.Submission, int64, error) {
	return nil, 0, nil
}
func (f *fakeSubmissionRepo) UpdateStatus(ctx context.Context, id int64, toStatus string, submittedDate sql.NullTime, updatedBy int64) error {
	return nil
}
func (f *fakeSubmissionRepo) TouchVersion(ctx context.Context, id int64, updatedBy int64) error {
	return nil
}
func (f *fakeSubmissionRepo) SetSubjectReference(ctx context.Context, id int64, subjectReferenceID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) UpdateFormVersion(ctx context.Context, id int64, formVersionID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) UpsertAnswer(ctx context.Context, submissionID int64, p submission_model.AnswerParams) error {
	return nil
}
func (f *fakeSubmissionRepo) ListAnswersBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Answer, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) RemapAnswerField(ctx context.Context, submissionID, oldFieldID, newFieldID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) InsertStatusHistory(ctx context.Context, submissionID int64, fromStatus sql.NullString, toStatus string, actorUserID int64, actorOrganizationID sql.NullInt64, note sql.NullString) error {
	return nil
}
func (f *fakeSubmissionRepo) ListStatusHistory(ctx context.Context, submissionID int64) ([]submission_model.StatusHistory, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) CreateKader(ctx context.Context, p submission_model.KaderParams) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) FindKaderBySubmission(ctx context.Context, submissionID int64) (submission_model.Kader, error) {
	return submission_model.Kader{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindKaderByID(ctx context.Context, id int64) (submission_model.Kader, error) {
	k, ok := f.kaders[id]
	if !ok {
		return submission_model.Kader{}, errors.New("not found")
	}
	return k, nil
}
func (f *fakeSubmissionRepo) UpdateKaderStatus(ctx context.Context, id int64, status string, uniqueCode sql.NullString, issuedDate sql.NullTime) error {
	if f.kaderStatusSet == nil {
		f.kaderStatusSet = map[int64]string{}
	}
	f.kaderStatusSet[id] = status
	k := f.kaders[id]
	k.Status = status
	f.kaders[id] = k
	return nil
}
func (f *fakeSubmissionRepo) ListKadersByOrganizations(ctx context.Context, organizationIDs []int64, status string) ([]submission_model.Kader, error) {
	inScope := map[int64]bool{}
	for _, id := range organizationIDs {
		inScope[id] = true
	}
	var out []submission_model.Kader
	for _, k := range f.kaders {
		if inScope[k.OrganizationID] && (status == "" || k.Status == status) {
			out = append(out, k)
		}
	}
	return out, nil
}
func (f *fakeSubmissionRepo) CountKaderCodesForOrgYear(ctx context.Context, codePrefix string) (int, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) CreateReview(ctx context.Context, p submission_model.ReviewParams) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) ListReviewsBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Review, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) UpsertFieldScore(ctx context.Context, submissionID, fieldID int64, rawScore float64, scoredByUserID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) ListFieldScoresBySubmission(ctx context.Context, submissionID int64) ([]submission_model.FieldScore, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) FindCurrentLevelResult(ctx context.Context, organizationID int64) (submission_model.LevelisasiResult, error) {
	return submission_model.LevelisasiResult{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindLevelResultBySubmission(ctx context.Context, submissionID int64) (submission_model.LevelisasiResult, error) {
	return submission_model.LevelisasiResult{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindLevelResultHistory(ctx context.Context, organizationID int64) ([]submission_model.LevelisasiResult, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) ClearCurrentLevelResult(ctx context.Context, organizationID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) CreateLevelResult(ctx context.Context, submissionID, organizationID int64, levelCode, justificationNote string, establishedByUserID int64) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) UpdateLevelResult(ctx context.Context, resultID int64, levelCode, justificationNote string, establishedByUserID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) PublishLevelResult(ctx context.Context, resultID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) LevelLabel(ctx context.Context, levelCode string) (string, error) {
	return "", nil
}
func (f *fakeSubmissionRepo) CountByFormAndStatuses(ctx context.Context, formID int64, statuses []string) (int, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) ListByFormAndStatuses(ctx context.Context, formID int64, statuses []string) ([]submission_model.Submission, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) ReassessSubmission(ctx context.Context, sub submission_model.Submission, actorUserID int64, note string) error {
	return nil
}
func (f *fakeSubmissionRepo) WipeSubmission(ctx context.Context, sub submission_model.Submission, newVersionID int64) error {
	return nil
}

type fakeFormRepo struct{}

func (fakeFormRepo) CreateForm(ctx context.Context, formCode, formName, description string, createdBy sql.NullInt64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) ExistsFormCode(ctx context.Context, code string) (bool, error) { return false, nil }
func (fakeFormRepo) ListForms(ctx context.Context) ([]submission_form_model.Form, error) {
	return nil, nil
}
func (fakeFormRepo) FindFormByID(ctx context.Context, id int64) (submission_form_model.Form, error) {
	return submission_form_model.Form{}, errors.New("not found")
}
func (fakeFormRepo) FindFormByCode(ctx context.Context, code string) (submission_form_model.Form, error) {
	return submission_form_model.Form{}, errors.New("not found")
}
func (fakeFormRepo) CreateVersion(ctx context.Context, formID int64, versionNumber int, createdBy sql.NullInt64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) MaxVersionNumber(ctx context.Context, formID int64) (int, error) { return 0, nil }
func (fakeFormRepo) FindVersionByID(ctx context.Context, id int64) (submission_form_model.Version, error) {
	return submission_form_model.Version{}, errors.New("not found")
}
func (fakeFormRepo) FindPublishedVersionByForm(ctx context.Context, formID int64) (submission_form_model.Version, error) {
	return submission_form_model.Version{}, errors.New("not found")
}
func (fakeFormRepo) ListVersionsByForm(ctx context.Context, formID int64) ([]submission_form_model.Version, error) {
	return nil, nil
}
func (fakeFormRepo) PublishVersion(ctx context.Context, id int64, publishedBy int64) error {
	return nil
}
func (fakeFormRepo) ArchiveOtherPublished(ctx context.Context, formID, exceptVersionID int64) error {
	return nil
}
func (fakeFormRepo) CreateSection(ctx context.Context, versionID int64, code, label string, sortOrder int, description sql.NullString) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) FindSectionByID(ctx context.Context, id int64) (submission_form_model.Section, error) {
	return submission_form_model.Section{}, errors.New("not found")
}
func (fakeFormRepo) UpdateSection(ctx context.Context, id int64, label string, sortOrder int, description sql.NullString) error {
	return nil
}
func (fakeFormRepo) DeleteSection(ctx context.Context, id int64) error { return nil }
func (fakeFormRepo) ListSectionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Section, error) {
	return nil, nil
}
func (fakeFormRepo) CreateField(ctx context.Context, p submission_form_model.FieldParams) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) FindFieldByID(ctx context.Context, id int64) (submission_form_model.Field, error) {
	return submission_form_model.Field{}, errors.New("not found")
}
func (fakeFormRepo) UpdateField(ctx context.Context, id int64, p submission_form_model.FieldParams) error {
	return nil
}
func (fakeFormRepo) DeleteField(ctx context.Context, id int64) error { return nil }
func (fakeFormRepo) ListFieldsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Field, error) {
	return nil, nil
}
func (fakeFormRepo) CreateOption(ctx context.Context, fieldID int64, value, label string, sortOrder int, score sql.NullFloat64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) FindOptionByID(ctx context.Context, id int64) (submission_form_model.Option, error) {
	return submission_form_model.Option{}, errors.New("not found")
}
func (fakeFormRepo) UpdateOption(ctx context.Context, id int64, value, label string, sortOrder int, isActive bool, score sql.NullFloat64) error {
	return nil
}
func (fakeFormRepo) DeleteOption(ctx context.Context, id int64) error { return nil }
func (fakeFormRepo) ListOptionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Option, error) {
	return nil, nil
}
func (fakeFormRepo) ListOptionsByField(ctx context.Context, fieldID int64) ([]submission_form_model.Option, error) {
	return nil, nil
}
func (fakeFormRepo) VersionStatusBySectionID(ctx context.Context, sectionID int64) (string, error) {
	return "", nil
}
func (fakeFormRepo) VersionStatusByFieldID(ctx context.Context, fieldID int64) (string, error) {
	return "", nil
}
func (fakeFormRepo) VersionIDByFieldID(ctx context.Context, fieldID int64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) CloneVersionStructure(ctx context.Context, fromVersionID, toVersionID int64) error {
	return nil
}

type fakeOrgRepoForPengurus struct {
	byID map[int64]organization_model.Organization
	root int64
}

func newTestOrgRepo() *fakeOrgRepoForPengurus {
	return &fakeOrgRepoForPengurus{byID: map[int64]organization_model.Organization{}}
}

func (f *fakeOrgRepoForPengurus) FindByID(ctx context.Context, id int64) (organization_model.Organization, error) {
	o, ok := f.byID[id]
	if !ok {
		return organization_model.Organization{}, errors.New("not found")
	}
	return o, nil
}
func (f *fakeOrgRepoForPengurus) ExistsByCode(ctx context.Context, code string) (bool, error) {
	return false, nil
}
func (f *fakeOrgRepoForPengurus) TypeCodeByID(ctx context.Context, id int64) (string, error) {
	return f.byID[id].OrganizationTypeCode, nil
}
func (f *fakeOrgRepoForPengurus) IsChildOf(ctx context.Context, targetID, parentID int64) (bool, error) {
	return false, nil
}
func (f *fakeOrgRepoForPengurus) ListAll(ctx context.Context, filt organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return nil, 0, nil
}
func (f *fakeOrgRepoForPengurus) ListByTypes(ctx context.Context, types []string, filt organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return nil, 0, nil
}
func (f *fakeOrgRepoForPengurus) ListSelfAndChildren(ctx context.Context, id int64, filt organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return nil, 0, nil
}
func (f *fakeOrgRepoForPengurus) Children(ctx context.Context, id int64) ([]organization_model.Organization, error) {
	return nil, nil
}
func (f *fakeOrgRepoForPengurus) RootPuskomnasID(ctx context.Context) (int64, error) {
	return f.root, nil
}
func (f *fakeOrgRepoForPengurus) ListActiveByType(ctx context.Context, typeCode string) ([]organization_model.Organization, error) {
	return nil, nil
}
func (f *fakeOrgRepoForPengurus) Create(ctx context.Context, p organization_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeOrgRepoForPengurus) Update(ctx context.Context, id int64, name, province, city, email, phone, website, photoURL string, updatedBy int64) error {
	return nil
}
func (f *fakeOrgRepoForPengurus) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return nil
}

type fakeUserRepoForPengurus struct {
	byID    map[int64]user_model.User
	updates []userUpdateCall
	bumped  []int64
}

type userUpdateCall struct {
	id             int64
	roleID         int64
	organizationID sql.NullInt64
}

func newTestUserRepo() *fakeUserRepoForPengurus {
	return &fakeUserRepoForPengurus{byID: map[int64]user_model.User{}}
}

func (f *fakeUserRepoForPengurus) FindByID(ctx context.Context, id int64) (user_model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return user_model.User{}, errors.New("not found")
	}
	return u, nil
}
func (f *fakeUserRepoForPengurus) FindByEmail(ctx context.Context, email string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepoForPengurus) FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepoForPengurus) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (f *fakeUserRepoForPengurus) ExistsByEmailExcept(ctx context.Context, email string, exceptID int64) (bool, error) {
	return false, nil
}
func (f *fakeUserRepoForPengurus) Create(ctx context.Context, p user_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeUserRepoForPengurus) List(ctx context.Context, filt user_dto.ListFilter) ([]user_model.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUserRepoForPengurus) SearchActive(ctx context.Context, search string, limit int) ([]user_model.User, error) {
	return nil, nil
}
func (f *fakeUserRepoForPengurus) Update(ctx context.Context, id int64, fullName, email string, roleID int64, isActive bool, organizationID sql.NullInt64, wildcardTierAccess sql.NullString, updatedBy int64) error {
	f.updates = append(f.updates, userUpdateCall{id: id, roleID: roleID, organizationID: organizationID})
	u := f.byID[id]
	u.RoleID = roleID
	u.OrganizationID = organizationID
	f.byID[id] = u
	return nil
}
func (f *fakeUserRepoForPengurus) SetOrganizationID(ctx context.Context, id int64, organizationID int64) error {
	return nil
}
func (f *fakeUserRepoForPengurus) UpdateContactInfo(ctx context.Context, id int64, phoneNumber, address sql.NullString) error {
	return nil
}
func (f *fakeUserRepoForPengurus) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepoForPengurus) SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error {
	return nil
}
func (f *fakeUserRepoForPengurus) LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error {
	return nil
}
func (f *fakeUserRepoForPengurus) UpdatePhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepoForPengurus) UpdateCustomPhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepoForPengurus) MarkEmailVerified(ctx context.Context, id int64) error { return nil }
func (f *fakeUserRepoForPengurus) SetEmailVerified(ctx context.Context, id int64, verified bool) error {
	return nil
}
func (f *fakeUserRepoForPengurus) SoftDelete(ctx context.Context, id int64, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepoForPengurus) LogLogin(ctx context.Context, userID int64, ip, ua, status string) error {
	return nil
}
func (f *fakeUserRepoForPengurus) TokenVersion(ctx context.Context, id int64) (int, error) {
	return 0, nil
}
func (f *fakeUserRepoForPengurus) BumpTokenVersion(ctx context.Context, id int64) error {
	f.bumped = append(f.bumped, id)
	return nil
}

type fakeRoleRepoForPengurus struct{ idsByName map[string]int64 }

func newTestRoleRepo() *fakeRoleRepoForPengurus {
	return &fakeRoleRepoForPengurus{idsByName: map[string]int64{
		roleNameKader: 1, constants.RoleLDKAdmin: 2,
		constants.RolePuskomdaVerifikator: 3, constants.RolePuskomnasVerifikator: 4,
	}}
}

func (f *fakeRoleRepoForPengurus) IDByName(ctx context.Context, name string) (int64, error) {
	id, ok := f.idsByName[name]
	if !ok {
		return 0, errors.New("role not found")
	}
	return id, nil
}
func (f *fakeRoleRepoForPengurus) FindByID(ctx context.Context, id int64) (role_model.Role, error) {
	return role_model.Role{}, nil
}
func (f *fakeRoleRepoForPengurus) ListWithUserCount(ctx context.Context, search string) ([]role_model.Role, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) Create(ctx context.Context, name, desc string) (int64, error) {
	return 0, nil
}
func (f *fakeRoleRepoForPengurus) Update(ctx context.Context, id int64, name, desc string, isActive bool) error {
	return nil
}
func (f *fakeRoleRepoForPengurus) Delete(ctx context.Context, id int64) error { return nil }
func (f *fakeRoleRepoForPengurus) CountUsers(ctx context.Context, id int64) (int, error) {
	return 0, nil
}
func (f *fakeRoleRepoForPengurus) PermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) PermissionNames(ctx context.Context, roleID int64) ([]string, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	return nil
}
func (f *fakeRoleRepoForPengurus) UsersByRole(ctx context.Context, roleID int64) ([]role_model.RoleUser, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) BumpTokenVersionForRole(ctx context.Context, roleID int64) error {
	return nil
}

type fakeOrgScopeForPengurus struct{ accessibleIDs []int64 }

func (f *fakeOrgScopeForPengurus) IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error) {
	for _, id := range f.accessibleIDs {
		if id == targetOrganizationID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeOrgScopeForPengurus) AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error) {
	return f.accessibleIDs, nil
}
// AccessibleOrganizationIDsForTarget returns ONLY the target itself (not the
// caller's full accessibleIDs) -- a real org-switcher narrowing would cascade
// to the target's children too, but for these tests "narrower than the
// caller's broad scope" is what matters: it must differ from
// AccessibleOrganizationIDs so a test can prove resolveScopedOrganizationIDs
// genuinely switches between the two based on caller.RequestedOrganizationID.
func (f *fakeOrgScopeForPengurus) AccessibleOrganizationIDsForTarget(ctx context.Context, targetOrganizationID int64) ([]int64, error) {
	return []int64{targetOrganizationID}, nil
}

// fakeJobqueueServiceForPengurus is a no-op stub for jobqueue_service.Service
// — ServiceImpl gained this 7th dependency (jobqueueSvc) after this plan was
// written, via unrelated concurrent work (publish-cascade job handler,
// commit 0a95bf0). None of this file's scenarios exercise jobqueue behavior,
// so every method is a zero-value/no-op stub, same pattern as the other 5
// fakes above.
type fakeJobqueueServiceForPengurus struct{}

func (fakeJobqueueServiceForPengurus) Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error) {
	return 0, nil
}
func (fakeJobqueueServiceForPengurus) RegisterHandler(jobType string, fn func(ctx context.Context, payload string) error) {
}
func (fakeJobqueueServiceForPengurus) ResolveCorrelation(ctx context.Context, waMessageID string) (string, int64, bool, error) {
	return "", 0, false, nil
}
func (fakeJobqueueServiceForPengurus) FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error) {
	return nil, nil
}
func (fakeJobqueueServiceForPengurus) HandleDeliveryStatus(ctx context.Context, waMessageID, status, errorDetail string) error {
	return nil
}
func (fakeJobqueueServiceForPengurus) HandleMessageSent(ctx context.Context, kirimdevMessageID, wamid string) error {
	return nil
}
func (fakeJobqueueServiceForPengurus) CMSList(ctx context.Context, q dto.ListQuery, statuses, queues []string, dateFrom, dateTo string) ([]jobqueue_dto.Response, int, error) {
	return nil, 0, nil
}
func (fakeJobqueueServiceForPengurus) CMSGet(ctx context.Context, id int64) (jobqueue_dto.Response, error) {
	return jobqueue_dto.Response{}, nil
}
func (fakeJobqueueServiceForPengurus) CMSStats(ctx context.Context) (jobqueue_dto.StatsResponse, error) {
	return jobqueue_dto.StatsResponse{}, nil
}
func (fakeJobqueueServiceForPengurus) Retry(ctx context.Context, id int64) error  { return nil }
func (fakeJobqueueServiceForPengurus) Delete(ctx context.Context, id int64) error { return nil }
func (fakeJobqueueServiceForPengurus) BulkDelete(ctx context.Context, ids []int64) error {
	return nil
}
func (fakeJobqueueServiceForPengurus) RunWorker(workerID int) {}
func (fakeJobqueueServiceForPengurus) RunStuckSweeper()       {}

// ---------- scenario fixtures ----------

const (
	ldkOrgID       int64 = 10
	puskomdaOrgID  int64 = 20
	puskomnasOrgID int64 = 30
	kaderUserID    int64 = 101
	callerUserID   int64 = 999
)

func buildTestService() (Service, *fakeSubmissionRepo, *fakeUserRepoForPengurus, *fakeOrgRepoForPengurus) {
	subRepo := &fakeSubmissionRepo{kaders: map[int64]submission_model.Kader{
		1: {KaderID: 1, UserID: kaderUserID, OrganizationID: ldkOrgID, FullName: "Budi", Status: constants.KaderStatusActive},
	}}
	userRepo := newTestUserRepo()
	userRepo.byID[kaderUserID] = user_model.User{UserID: kaderUserID, FullName: "Budi", Email: "budi@example.com", RoleID: 1, RoleName: roleNameKader, IsActive: true, OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true}}
	userRepo.byID[callerUserID] = user_model.User{UserID: callerUserID, FullName: "Caller", Email: "caller@example.com", RoleID: 2, RoleName: constants.RoleLDKAdmin, IsActive: true, OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true}}

	orgRepo := newTestOrgRepo()
	orgRepo.byID[ldkOrgID] = organization_model.Organization{OrganizationID: ldkOrgID, OrganizationTypeCode: constants.OrgTypeLDK, OrganizationName: "LDK Syahid", ParentOrganizationID: sql.NullInt64{Int64: puskomdaOrgID, Valid: true}}
	orgRepo.byID[puskomdaOrgID] = organization_model.Organization{OrganizationID: puskomdaOrgID, OrganizationTypeCode: constants.OrgTypePuskomda, OrganizationName: "Puskomda Banten"}
	orgRepo.byID[puskomnasOrgID] = organization_model.Organization{OrganizationID: puskomnasOrgID, OrganizationTypeCode: constants.OrgTypePuskomnas, OrganizationName: "Puskomnas"}
	orgRepo.root = puskomnasOrgID

	roleRepo := newTestRoleRepo()
	orgScope := &fakeOrgScopeForPengurus{accessibleIDs: []int64{ldkOrgID, puskomdaOrgID, puskomnasOrgID}}

	svc := NewService(subRepo, fakeFormRepo{}, orgRepo, userRepo, roleRepo, orgScope, fakeJobqueueServiceForPengurus{})
	return svc, subRepo, userRepo, orgRepo
}

func ldkCaller() CallerScope {
	id := ldkOrgID
	return CallerScope{UserID: callerUserID, OrganizationID: &id, OrganizationTypeCode: constants.OrgTypeLDK}
}

// ---------- org-switcher (caller.RequestedOrganizationID) scenarios ----------
// Regression tests for "switch org in the switcher, the table doesn't
// change" -- ListPengurus/UpdatePengurusRole must route through
// resolveScopedOrganizationIDs (already proven correct for submission
// listing) instead of calling orgScope.AccessibleOrganizationIDs directly.

func TestListPengurus_RequestedOrganizationID_NarrowsScope(t *testing.T) {
	svc, _, _, _ := buildTestService()
	caller := CallerScope{UserID: callerUserID, OrganizationID: func() *int64 { id := puskomnasOrgID; return &id }(), OrganizationTypeCode: constants.OrgTypePuskomnas}
	target := puskomdaOrgID
	caller.RequestedOrganizationID = &target

	entries, err := svc.ListPengurus(context.Background(), caller)
	if err != nil {
		t.Fatalf("ListPengurus() error = %v", err)
	}
	// Without narrowing, this Puskomnas caller's broad scope [ldkOrgID,
	// puskomdaOrgID, puskomnasOrgID] would include the fixture's only kader
	// (home LDK = ldkOrgID), per TestListPengurus_ReturnsActiveKaderInScope.
	// The fake's AccessibleOrganizationIDsForTarget(puskomdaOrgID) narrows to
	// just {puskomdaOrgID}, which does NOT include ldkOrgID -- an empty
	// result here proves the switcher's target was actually honored instead
	// of the broad default being used regardless.
	if len(entries) != 0 {
		t.Errorf("ListPengurus() with RequestedOrganizationID=%d = %+v, want empty (org-switcher narrowing was not applied)", target, entries)
	}
}

func TestListPengurus_RequestedOrganizationIDOutsideCallerScope_ReturnsForbidden(t *testing.T) {
	svc, _, _, _ := buildTestService()
	caller := ldkCaller()
	outside := int64(999999)
	caller.RequestedOrganizationID = &outside

	if _, err := svc.ListPengurus(context.Background(), caller); err == nil {
		t.Fatal("ListPengurus() error = nil, want Forbidden -- requested org is outside caller's own accessible scope (IDOR check)")
	}
}

func TestUpdatePengurusRole_RequestedOrganizationID_NarrowsScope(t *testing.T) {
	svc, _, _, _ := buildTestService()
	caller := CallerScope{UserID: callerUserID, OrganizationID: func() *int64 { id := puskomnasOrgID; return &id }(), OrganizationTypeCode: constants.OrgTypePuskomnas}
	target := puskomdaOrgID
	caller.RequestedOrganizationID = &target

	// Same reasoning as TestListPengurus_RequestedOrganizationID_NarrowsScope:
	// narrowed to {puskomdaOrgID}, so the fixture's kader (home LDK=ldkOrgID)
	// must be reported NotFound -- "outside the currently switched scope" --
	// even though the caller's broad Puskomnas scope would normally reach it.
	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want NotFound -- target's home LDK is outside the org-switcher's narrowed scope")
	}
}

// ---------- UpdatePengurusRole scenarios ----------

// TestUpdatePengurusRole_RejectsDemotingHigherTierCurrentRole guards against
// a lower-tier caller demoting someone who already outranks their own tier.
// pengurusRoleAllowed alone only validates the DESTINATION role -- without
// also checking the target's CURRENT role, an LDK Admin (allow-list {Kader,
// LDK Admin}) could set ANY user whose home LDK is theirs to "Kader",
// including a Puskomda/Puskomnas Verifikator whose ms_kader.organizationID
// still points at that same LDK (it never changes on promotion).
func TestUpdatePengurusRole_RejectsDemotingHigherTierCurrentRole(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	// kaderUserID's home LDK (ldkOrgID) is the caller's own LDK, but their
	// CURRENT role is Puskomda Verifikator -- outranks an LDK Admin caller's
	// allow-list ({Kader, LDK Admin}), even though "Kader" (the requested
	// destination) is itself in that allow-list.
	userRepo.byID[kaderUserID] = user_model.User{
		UserID: kaderUserID, FullName: "Budi", Email: "budi@example.com",
		RoleID: 3, RoleName: constants.RolePuskomdaVerifikator, IsActive: true,
		OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true},
	}

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: roleNameKader})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want Forbidden — LDK Admin caller cannot demote a Puskomda Verifikator, even one whose home LDK is their own")
	}
	if len(userRepo.updates) != 0 {
		t.Errorf("expected no userRepo.Update() calls, got %+v", userRepo.updates)
	}
}

func TestUpdatePengurusRole_RejectsSelfAction(t *testing.T) {
	svc, _, _, _ := buildTestService()
	caller := ldkCaller()
	caller.UserID = kaderUserID // caller targeting themselves

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want Forbidden for self-action")
	}
}

func TestUpdatePengurusRole_RejectsRoleOutsideCallerTierAllowList(t *testing.T) {
	svc, _, _, _ := buildTestService()

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RolePuskomdaVerifikator})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want Forbidden — LDK caller cannot assign Puskomda Verifikator")
	}
}

func TestUpdatePengurusRole_RejectsTargetOutsideCallerScope(t *testing.T) {
	svc, subRepo, _, _ := buildTestService()
	subRepo.kaders[2] = submission_model.Kader{KaderID: 2, UserID: 777, OrganizationID: 999999, Status: constants.KaderStatusActive}

	_, err := svc.UpdatePengurusRole(context.Background(), 777, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want NotFound — target's org is outside caller's accessible scope")
	}
}

func TestUpdatePengurusRole_PromoteToLDKAdmin_KeepsHomeOrganizationID(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()

	entry, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err != nil {
		t.Fatalf("UpdatePengurusRole() error = %v", err)
	}
	if entry.HomeOrganizationID != ldkOrgID {
		t.Errorf("entry.HomeOrganizationID = %d, want %d (unchanged)", entry.HomeOrganizationID, ldkOrgID)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].organizationID.Int64 != ldkOrgID {
		t.Errorf("userRepo.Update() organizationID = %+v, want %d", userRepo.updates, ldkOrgID)
	}
	if len(userRepo.bumped) != 1 || userRepo.bumped[0] != kaderUserID {
		t.Errorf("expected BumpTokenVersion(%d) exactly once, got %v", kaderUserID, userRepo.bumped)
	}
}

func TestUpdatePengurusRole_PromoteToPuskomdaVerifikator_ReassignsToParentOrg(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	caller := CallerScope{UserID: callerUserID, OrganizationID: func() *int64 { id := puskomdaOrgID; return &id }(), OrganizationTypeCode: constants.OrgTypePuskomda}

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RolePuskomdaVerifikator})
	if err != nil {
		t.Fatalf("UpdatePengurusRole() error = %v", err)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].organizationID.Int64 != puskomdaOrgID {
		t.Errorf("userRepo.Update() organizationID = %+v, want %d (LDK's parent Puskomda)", userRepo.updates, puskomdaOrgID)
	}
}

func TestUpdatePengurusRole_PromoteToPuskomnasVerifikator_ReassignsToRoot(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	caller := CallerScope{UserID: callerUserID, OrganizationID: func() *int64 { id := puskomnasOrgID; return &id }(), OrganizationTypeCode: constants.OrgTypePuskomnas}

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RolePuskomnasVerifikator})
	if err != nil {
		t.Fatalf("UpdatePengurusRole() error = %v", err)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].organizationID.Int64 != puskomnasOrgID {
		t.Errorf("userRepo.Update() organizationID = %+v, want %d (Puskomnas root)", userRepo.updates, puskomnasOrgID)
	}
}

// ---------- ListPengurus ----------

func TestListPengurus_ReturnsActiveKaderInScope(t *testing.T) {
	svc, _, _, _ := buildTestService()

	entries, err := svc.ListPengurus(context.Background(), ldkCaller())
	if err != nil {
		t.Fatalf("ListPengurus() error = %v", err)
	}
	if len(entries) != 1 || entries[0].UserID != kaderUserID {
		t.Fatalf("ListPengurus() = %+v, want exactly one entry for userID %d", entries, kaderUserID)
	}
	if entries[0].RoleName != roleNameKader {
		t.Errorf("entries[0].RoleName = %q, want %q", entries[0].RoleName, roleNameKader)
	}
	if entries[0].HomeOrganizationName != "LDK Syahid" {
		t.Errorf("entries[0].HomeOrganizationName = %q, want %q", entries[0].HomeOrganizationName, "LDK Syahid")
	}
}

// ---------- DeactivateKader cascade ----------

func TestDeactivateKader_RevertsPromotedUserToKader(t *testing.T) {
	svc, subRepo, userRepo, _ := buildTestService()
	userRepo.byID[kaderUserID] = user_model.User{UserID: kaderUserID, FullName: "Budi", Email: "budi@example.com", RoleID: 2, RoleName: constants.RoleLDKAdmin, IsActive: true, OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true}}

	if err := svc.DeactivateKader(context.Background(), 1, ldkCaller()); err != nil {
		t.Fatalf("DeactivateKader() error = %v", err)
	}
	if subRepo.kaderStatusSet[1] != constants.KaderStatusInactive {
		t.Errorf("kader status = %q, want %q", subRepo.kaderStatusSet[1], constants.KaderStatusInactive)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].roleID != 1 {
		t.Fatalf("expected exactly one role revert to Kader (roleID 1), got %+v", userRepo.updates)
	}
	if userRepo.updates[0].organizationID.Int64 != ldkOrgID {
		t.Errorf("reverted organizationID = %d, want %d (home LDK)", userRepo.updates[0].organizationID.Int64, ldkOrgID)
	}
}

func TestDeactivateKader_PlainKader_NeverTouchesUserRecord(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	// kaderUserID is already role Kader in buildTestService()'s fixture.

	if err := svc.DeactivateKader(context.Background(), 1, ldkCaller()); err != nil {
		t.Fatalf("DeactivateKader() error = %v", err)
	}
	if len(userRepo.updates) != 0 {
		t.Errorf("expected no userRepo.Update() calls for a plain Kader, got %+v", userRepo.updates)
	}
}

// TestDeactivateKader_RejectsSelfAction guards Scope decision #5 ("A caller
// can never ... (via the existing cascade) deactivate their own account")
// at the cascade's actual entry point, not just at UpdatePengurusRole.
// Anyone who holds kader.deactivate (the permission guarding this endpoint)
// already holds a promoted role, so self-targeting here always means
// triggering the self-lockout cascade this decision forbids.
func TestDeactivateKader_RejectsSelfAction(t *testing.T) {
	svc, subRepo, userRepo, _ := buildTestService()
	subRepo.kaders[2] = submission_model.Kader{KaderID: 2, UserID: callerUserID, OrganizationID: ldkOrgID, FullName: "Caller", Status: constants.KaderStatusActive}

	if err := svc.DeactivateKader(context.Background(), 2, ldkCaller()); err == nil {
		t.Fatal("DeactivateKader() error = nil, want Forbidden — caller cannot deactivate their own Kader record")
	}
	if subRepo.kaderStatusSet[2] != "" {
		t.Errorf("expected kader status 2 to be untouched, got %q", subRepo.kaderStatusSet[2])
	}
	if len(userRepo.updates) != 0 {
		t.Errorf("expected no userRepo.Update() calls, got %+v", userRepo.updates)
	}
}
