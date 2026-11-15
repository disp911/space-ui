package controller

import (
	"strconv"

	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/web/service"

	"github.com/gin-gonic/gin"
)

// RuleListController edits the lists of domains and subnets served to a
// router's podkop as sing-box rule-sets.
type RuleListController struct {
	BaseController
	service service.RuleListService
}

func NewRuleListController(g *gin.RouterGroup) *RuleListController {
	a := &RuleListController{}
	g.GET("", a.list)
	g.POST("/save", a.save)
	g.POST("/delete/:id", a.delete)
	return a
}

// list returns the lists and the address the website answers at, which
// the lists' addresses start with.
func (a *RuleListController) list(c *gin.Context) {
	lists, err := a.service.All()
	if err != nil {
		jsonMsg(c, "lists", err)
		return
	}
	base := service.SiteBase(c.Request.Host)
	if base == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + c.Request.Host
	}
	jsonObj(c, gin.H{"base": base, "lists": lists}, nil)
}

func (a *RuleListController) save(c *gin.Context) {
	var in model.RuleList
	if err := c.ShouldBindJSON(&in); err != nil {
		jsonMsg(c, "list", err)
		return
	}
	list, err := a.service.Save(&in)
	if err != nil {
		jsonMsg(c, "list", err)
		return
	}
	jsonObj(c, list, nil)
}

func (a *RuleListController) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		jsonMsg(c, "list", err)
		return
	}
	jsonMsg(c, "", a.service.Delete(id))
}
