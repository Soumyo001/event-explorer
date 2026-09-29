package controllers

import "github.com/beego/beego/v2/server/web"

type PageController struct {
	web.Controller
}

func (p *PageController) Home() {
	p.Data["Title"] = "Event Explorer"
	p.TplName = "home.tpl"
}
