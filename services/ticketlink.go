package services

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"eventexplorer/config"
	"eventexplorer/models"

	"github.com/beego/beego/v2/core/logs"
)

var (
	initDefaultValidatorOnce sync.Once
	defaultTicketValidator   TicketLinkValidator
)

type TicketLinkValidator interface {
	Validate(rawURL string) (string, error)
}

type ticketLinkValidator struct {
	allowedHosts []string
}

func NewTicketLinkValidator(hosts []string) TicketLinkValidator {
	normalised := make([]string, 0, len(hosts))
	for _, h := range hosts {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			normalised = append(normalised, h)
		}
	}
	return &ticketLinkValidator{allowedHosts: normalised}
}

func (v *ticketLinkValidator) Validate(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("%w: the event carries no ticket url", models.ErrInvalidTicketURL)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: url could not be parsed", models.ErrInvalidTicketURL)
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("%w: scheme %q is not https", models.ErrInvalidTicketURL, parsed.Scheme)
	}

	if parsed.User != nil {
		return "", fmt.Errorf("%w: url carries credentials", models.ErrInvalidTicketURL)
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("%w: url has no host", models.ErrInvalidTicketURL)
	}

	if !v.hostAllowed(host) {
		return "", fmt.Errorf("%w: host %q is not approved", models.ErrInvalidTicketURL, host)
	}

	return parsed.String(), nil
}

func (v *ticketLinkValidator) hostAllowed(host string) bool {
	for _, allowed := range v.allowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

var _ TicketLinkValidator = (*ticketLinkValidator)(nil)

func DefaultTicketLinkValidator() TicketLinkValidator {
	initDefaultValidatorOnce.Do(func() {
		hosts := config.GetConfig().AllowedTicketHosts
		logs.Info("ticket link validator active for %d approved hosts", len(hosts))
		defaultTicketValidator = NewTicketLinkValidator(hosts)
	})
	return defaultTicketValidator
}
