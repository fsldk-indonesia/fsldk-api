// Package organization merangkai routing modul organization.
package organization

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/organization/organization_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul organization.
func RegisterRoutes(rg *gin.RouterGroup, h organization_handler.Handler, mw *middlewares.Middleware) {
	// Daftar organisasi yang dapat diakses pemanggil (dashboard switcher) —
	// cukup terautentikasi, sama seperti /me/menus.
	rg.GET("/me/organizations", mw.Auth(), mw.RequireVerified(), h.Me)

	// Direktori organisasi aktif lintas cakupan akses — dipakai skenario
	// pemilihan bebas seperti Kader memilih LDK tujuan pendaftaran Sensus
	// Kader, bukan navigasi/manajemen organisasi sehingga tidak dikunci
	// permission admin organization.*.
	rg.GET("/organizations/directory", mw.Auth(), mw.RequireVerified(), h.Directory)

	viewPerm := mw.RequirePermission(
		constants.PermOrganizationProfileManage,
		constants.PermOrganizationLDKList,
		constants.PermOrganizationLDKListNational,
		constants.PermOrganizationPuskomdaList,
	)

	g := rg.Group("/organizations")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", viewPerm, h.List)
		g.GET("/:id", viewPerm, mw.RequireOrganizationScope("id"), h.Get)
		g.GET("/:id/children", viewPerm, mw.RequireOrganizationScope("id"), h.Children)
		g.POST("", mw.RequirePermission(constants.PermOrganizationCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermOrganizationProfileManage), mw.RequireOrganizationScope("id"), h.Update)
		g.POST("/:id/deactivate", mw.RequirePermission(constants.PermOrganizationDeactivate), mw.RequireOrganizationScope("id"), h.Deactivate)
		g.POST("/:id/reactivate", mw.RequirePermission(constants.PermOrganizationDeactivate), mw.RequireOrganizationScope("id"), h.Reactivate)
	}
}
