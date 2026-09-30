package controllers

import (
	"errors"
	"eventexplorer/models"
	"net/http"

	"github.com/beego/beego/v2/server/web"
)

type BaseController struct {
	web.Controller
}

func (c *BaseController) JSONError(status int, code, message string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = models.APIError{Error: code, Message: message}
	c.ServeJSON()
}

func (c *BaseController) JSONData(payload any) {
	c.Data["json"] = payload
	c.ServeJSON()
}

func statusForError(err error) (int, string, string) {
	switch {
	case errors.Is(err, models.ErrInvalidInput):
		return http.StatusBadRequest, "invalid_input", "That request could not be understood"
	case errors.Is(err, models.ErrNotFound):
		return http.StatusNotFound, "not_found", "We could not find what you were looking for"
	case errors.Is(err, models.ErrNoCityComponent):
		return http.StatusUnprocessableEntity, "no_city", "That place does not resolve to a city we can search"
	case errors.Is(err, models.ErrInvalidTicketURL):
		return http.StatusBadGateway, "invalid_ticket_link", "This event's ticket link could not be verified"
	default:
		return http.StatusBadGateway, "upstream_unavailable", "The service is temporarily unavailable. Please try again later."
	}
}

func (c *BaseController) FailJSON(err error) {
	status, code, message := statusForError(err)
	c.JSONError(status, code, message)
}

func heading(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "Something was missing"
	case http.StatusNotFound:
		return "Not found"
	case http.StatusUnprocessableEntity:
		return "City unavailable"
	default:
		return "Service unavailable"
	}
}

func (c *BaseController) RenderError(status int, headingText, message, backURL string) {
	if backURL == "" {
		backURL = "/"
	}
	c.Ctx.Output.SetStatus(status)
	c.Data["Page"] = models.ErrorPageData{
		Title:      headingText,
		StatusCode: status,
		Heading:    headingText,
		Message:    message,
		BackURL:    backURL,
	}
	c.TplName = "error.tpl"
}

func (c *BaseController) FailPage(err error, backURL string) {
	status, _, message := statusForError(err)
	c.RenderError(status, heading(status), message, backURL)
}
