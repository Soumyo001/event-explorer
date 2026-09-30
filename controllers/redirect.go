package controllers

import (
	"strings"

	"eventexplorer/services"

	"github.com/beego/beego/v2/core/logs"
)

type RedirectController struct {
	BaseController
	Events services.EventService
	Links  services.TicketLinkValidator
}

func (r *RedirectController) Prepare() {
	if r.Events == nil {
		r.Events = services.DefaultEventService()
	}
	if r.Links == nil {
		r.Links = services.DefaultTicketLinkValidator()
	}
	if r.Events == nil || r.Links == nil {
		panic("redirect controller dependencies are not initialised")
	}
}

func (r *RedirectController) Ticket() {
	eventID := strings.TrimSpace(r.Ctx.Input.Param(":eventId"))

	event, err := r.Events.RawEvent(r.Ctx.Request.Context(), eventID)
	if err != nil {
		logs.Warn("ticket redirect: event %q could not be resolved: %v", eventID, err)
		r.FailPage(err, "/")
		return
	}

	target, err := r.Links.Validate(event.URL)
	if err != nil {
		logs.Error("ticket redirect: link rejected for event %q: %v", eventID, err)
		r.FailPage(err, "/events/"+eventID)
		return
	}

	logs.Info("ticket redirect: event %q -> approved provider link", eventID)
	r.Ctx.Redirect(302, target)
}
