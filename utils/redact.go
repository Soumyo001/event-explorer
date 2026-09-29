package utils

import (
	"errors"
	"net/url"
)

var sensitiveParams = []string{"apikey", "key", "api_key", "sessionToken"}

func redactURL(raw string) string {
	parsedUrl, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}

	queryParams := parsedUrl.Query()
	for _, param := range sensitiveParams {
		if queryParams.Has(param) {
			queryParams.Set(param, "REDACTED")
		}
	}

	parsedUrl.RawQuery = queryParams.Encode()
	return parsedUrl.String()
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &url.Error{
			Op:  urlErr.Op,
			URL: redactURL(urlErr.URL),
			Err: urlErr.Err,
		}
	}
	return err
}

func RedactURL(raw string) string   { return redactURL(raw) }
func SanitizeError(err error) error { return sanitizeError(err) }
