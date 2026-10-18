package organization_service

import (
	"context"
	"database/sql"
	"testing"

	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"
	"fsldk-api/modules/organization/organization_repository"
)

type fakeRepo struct {
	organization_repository.Repository
	orgs []organization_model.Organization
}

func (f *fakeRepo) FindByID(ctx context.Context, id int64) (organization_model.Organization, error) {
	for _, o := range f.orgs {
		if o.OrganizationID == id {
			return o, nil
		}
	}
	return organization_model.Organization{}, organization_repository.ErrNotFound
}

func (f *fakeRepo) ListAll(ctx context.Context, filter organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return f.filtered(filter), int64(len(f.orgs)), nil
}

func (f *fakeRepo) ListByTypes(ctx context.Context, types []string, filter organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return f.filtered(filter), int64(len(f.orgs)), nil
}

func (f *fakeRepo) filtered(filter organization_dto.ListFilter) []organization_model.Organization {
	var out []organization_model.Organization
	for _, o := range f.orgs {
		if filter.OrganizationTypeCode != "" && o.OrganizationTypeCode != filter.OrganizationTypeCode {
			continue
		}
		out = append(out, o)
	}
	return out
}

func nullInt(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }

// Puskomnas (1) -> Puskomda A (2), Puskomda B (3) -> LDK 10/11 (anak A), LDK 20 (anak B).
func sampleOrgs() []organization_model.Organization {
	return []organization_model.Organization{
		{OrganizationID: 1, OrganizationTypeCode: "PUSKOMNAS"},
		{OrganizationID: 2, OrganizationTypeCode: "PUSKOMDA", ParentOrganizationID: nullInt(1)},
		{OrganizationID: 3, OrganizationTypeCode: "PUSKOMDA", ParentOrganizationID: nullInt(1)},
		{OrganizationID: 10, OrganizationTypeCode: "LDK", ParentOrganizationID: nullInt(2)},
		{OrganizationID: 11, OrganizationTypeCode: "LDK", ParentOrganizationID: nullInt(2)},
		{OrganizationID: 20, OrganizationTypeCode: "LDK", ParentOrganizationID: nullInt(3)},
	}
}

func TestAccessibleList_WildcardCaller_IgnoresSiblingFilter(t *testing.T) {
	svc := NewService(&fakeRepo{orgs: sampleOrgs()})
	caller := CallerScope{WildcardTierAccess: "LDK,PUSKOMDA,PUSKOMNAS"}
	siblingOf := int64(10)

	got, err := svc.AccessibleList(context.Background(), caller, "LDK", &siblingOf, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected all 3 LDKs nationally for a wildcard caller, got %d: %+v", len(got), got)
	}
}

func TestAccessibleList_NonWildcardCaller_StillFiltersToSiblings(t *testing.T) {
	svc := NewService(&fakeRepo{orgs: sampleOrgs()})
	rootID := int64(1)
	caller := CallerScope{OrganizationTypeCode: "PUSKOMNAS", OrganizationID: &rootID}
	siblingOf := int64(10)

	got, err := svc.AccessibleList(context.Background(), caller, "LDK", &siblingOf, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected only the 2 LDKs sharing organizationID 10's parent, got %d: %+v", len(got), got)
	}
}
