package organization_service

import (
	"context"
	"testing"

	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"
	"fsldk-api/modules/organization/organization_repository"
)

// typedRepo menambah perilaku ListByTypes/TypeCodeByID/IsChildOf yang
// menghormati tipe organisasi, supaya wildcardTierAccess bisa diuji nyata.
type typedRepo struct{ fakeRepo }

func (r *typedRepo) ListByTypes(ctx context.Context, types []string, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	var out []organization_model.Organization
	for _, o := range r.filtered(f) {
		for _, t := range types {
			if o.OrganizationTypeCode == t {
				out = append(out, o)
				break
			}
		}
	}
	return out, int64(len(out)), nil
}

func (r *typedRepo) TypeCodeByID(ctx context.Context, id int64) (string, error) {
	for _, o := range r.orgs {
		if o.OrganizationID == id {
			return o.OrganizationTypeCode, nil
		}
	}
	return "", organization_repository.ErrNotFound
}

func (r *typedRepo) IsChildOf(ctx context.Context, id, parentID int64) (bool, error) {
	for _, o := range r.orgs {
		if o.OrganizationID == id {
			return o.ParentOrganizationID.Valid && o.ParentOrganizationID.Int64 == parentID, nil
		}
	}
	return false, nil
}

func newTypedSvc() Service {
	return NewService(&typedRepo{fakeRepo{orgs: sampleOrgs()}})
}

// Puskomda Verifikator (Puskomda 2) + wildcard LDK: tetap boleh Puskomda
// sendiri, boleh semua LDK, tapi TIDAK Puskomda lain.
func TestIsAccessible_PuskomdaWithLdkWildcard(t *testing.T) {
	svc := newTypedSvc()
	own := int64(2)
	cases := map[int64]bool{2: true, 3: false, 10: true, 20: true, 1: false}
	for target, want := range cases {
		got, err := svc.IsAccessible(context.Background(), &own, "PUSKOMDA", "LDK", target)
		if err != nil {
			t.Fatalf("target %d: %v", target, err)
		}
		if got != want {
			t.Errorf("target %d: got %v, want %v", target, got, want)
		}
	}
}

// Puskomnas Verifikator + wildcard LDK,PUSKOMDA: tetap boleh Puskomnas
// sendiri (root), yang menjadi tujuan laporan error sebelumnya.
func TestIsAccessible_PuskomnasWithLowerWildcard(t *testing.T) {
	svc := newTypedSvc()
	root := int64(1)
	for _, target := range []int64{1, 2, 3, 10, 20} {
		got, err := svc.IsAccessible(context.Background(), &root, "PUSKOMNAS", "LDK,PUSKOMDA", target)
		if err != nil {
			t.Fatalf("target %d: %v", target, err)
		}
		if !got {
			t.Errorf("target %d: expected accessible", target)
		}
	}
}

// LDK Admin + wildcard LDK: hanya LDK (semua), bukan Puskomda/Puskomnas.
func TestIsAccessible_LdkWithLdkWildcard(t *testing.T) {
	svc := newTypedSvc()
	own := int64(10)
	cases := map[int64]bool{10: true, 11: true, 20: true, 2: false, 1: false}
	for target, want := range cases {
		got, err := svc.IsAccessible(context.Background(), &own, "LDK", "LDK", target)
		if err != nil {
			t.Fatalf("target %d: %v", target, err)
		}
		if got != want {
			t.Errorf("target %d: got %v, want %v", target, got, want)
		}
	}
}

// Switcher Puskomda dengan wildcard LDK harus memuat Puskomda sendiri.
func TestAccessibleList_PuskomdaWithLdkWildcard_IncludesOwnPuskomda(t *testing.T) {
	svc := newTypedSvc()
	own := int64(2)
	caller := CallerScope{OrganizationTypeCode: "PUSKOMDA", OrganizationID: &own, WildcardTierAccess: "LDK"}

	got, err := svc.AccessibleList(context.Background(), caller, "PUSKOMDA", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].OrganizationID != 2 {
		t.Fatalf("expected only own Puskomda (2), got %+v", got)
	}
}
