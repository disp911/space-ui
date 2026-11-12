package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PanelController is the main controller for the Space panel, managing sub-controllers.
type PanelController struct {
	BaseController

	settingController     *SettingController
	xraySettingController *XraySettingController
}

// NewPanelController creates a new PanelController and initializes its routes.
func NewPanelController(g *gin.RouterGroup) *PanelController {
	a := &PanelController{}
	a.initRouter(g)
	return a
}

// initRouter sets up the panel's settings endpoints, behind the login check,
// and sends the old interface's pages to the same screens of the new one, so
// bookmarks keep working.
func (a *PanelController) initRouter(g *gin.RouterGroup) {
	for from, to := range map[string]string{"/panel/": "", "/panel/inbounds": "inbounds", "/panel/settings": "settings", "/panel/xray": "xray"} {
		g.GET(from, func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, c.GetString("base_path")+to)
		})
	}

	g = g.Group("/panel")
	g.Use(a.checkLogin)

	a.settingController = NewSettingController(g)
	a.xraySettingController = NewXraySettingController(g)
}
