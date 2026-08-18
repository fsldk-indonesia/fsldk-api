// Package organization_service memuat logika bisnis modul organization,
// termasuk resolusi cakupan akses berjenjang (LDK/Puskomda/Puskomnas).
package organization_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/organization/organization_dto"
)

// CallerScope menampung identitas & scope organisasi pengguna pemanggil,
// diambil dari klaim token (lihat base/appctx).
type CallerScope struct {
	UserID               int64
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
}

// Service adalah kontrak logika bisnis organisasi.
type Service interface {
	// IsAccessible memenuhi kontrak middlewares.OrgScopeLoader.
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
	// TypeCodeByID mengembalikan organizationTypeCode organisasi — dipakai
	// modul lain (mis. dashboard) untuk menentukan tema/tier saat caller
	// membuka konteks organisasi lain lewat switcher.
	TypeCodeByID(ctx context.Context, id int64) (string, error)

	// AccessibleList mengembalikan daftar organisasi yang dapat diakses caller
	// (dashboard switcher). organizationTypeCode (WAJIB diisi oleh context
	// switcher lokal shell cms-ldk/cms-puskomda — TIDAK dipakai dropdown akun
	// navbar global yang boleh kosong) MEMBATASI hasil hanya ke tipe tersebut
	// — mencegah LDK/Puskomda/Puskomnas tercampur dalam satu shell (miss-
	// development-prompt-2.md poin 2). siblingOf (opsional) mempersempit lebih
	// lanjut ke organisasi dengan parentOrganizationID yang sama (default
	// "sesama Puskomda/wilayah"); q (opsional, mengalahkan siblingOf bila
	// keduanya diisi) mencari lintas seluruh accessible set caller pada tipe
	// yang sama (efektif "nasional" untuk Puskomnas/wildcard).
	AccessibleList(ctx context.Context, caller CallerScope, organizationTypeCode string, siblingOf *int64, q string) ([]organization_dto.MeOrganization, error)
	// Directory mengembalikan organisasi aktif bertipe typeCode tanpa
	// dibatasi cakupan akses pemanggil (mis. Kader memilih LDK tujuan).
	Directory(ctx context.Context, typeCode string) ([]organization_dto.DirectoryEntry, error)
	// AccessibleOrganizationIDs mengembalikan seluruh organizationID yang
	// dapat diakses caller — dipakai modul lain (mis. submission) untuk
	// menyaring data tanpa perlu mengulang logika cascade/wildcard.
	AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error)
	// AccessibleOrganizationIDsForTarget mengembalikan cascade organizationID
	// yang berakar pada targetOrganizationID itu sendiri (bukan home org
	// caller) — dipakai modul lain saat caller secara eksplisit memilih
	// organisasi lain lewat switcher (mis. `?organizationID=`). Pemanggil
	// WAJIB memvalidasi targetOrganizationID via IsAccessible lebih dulu.
	AccessibleOrganizationIDsForTarget(ctx context.Context, targetOrganizationID int64) ([]int64, error)
	List(ctx context.Context, caller CallerScope, q dto.ListQuery, typeFilter string) ([]organization_dto.Response, int, error)
	Get(ctx context.Context, id int64) (organization_dto.Response, error)
	Children(ctx context.Context, id int64) ([]organization_dto.Response, error)
	Create(ctx context.Context, caller CallerScope, req organization_dto.CreateRequest) (organization_dto.Response, error)
	Update(ctx context.Context, id int64, req organization_dto.UpdateRequest, actorID int64) (organization_dto.Response, error)
	SetActive(ctx context.Context, caller CallerScope, id int64, active bool) error
}
