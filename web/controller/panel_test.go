package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOldPagesLeadToTheNewScreens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("base_path", "/secret/") })
	NewPanelController(engine.Group("/secret/"))

	for from, to := range map[string]string{
		"/secret/panel/":         "/secret/",
		"/secret/panel/inbounds": "/secret/inbounds",
		"/secret/panel/settings": "/secret/settings",
		"/secret/panel/xray":     "/secret/xray",
	} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, from, nil))
		if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != to {
			t.Fatalf("%s: HTTP %d to %q, want %q", from, rec.Code, rec.Header().Get("Location"), to)
		}
	}
}
