package adminui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"MtgLeaderwebserver/internal/domain"
)

func smtpAliases(settings domain.SMTPSettings) []string {
	out := make([]string, 0, 1+len(settings.AliasEmails))
	addAlias := func(email string) {
		email = normalizeEmail(email)
		if email == "" {
			return
		}
		for _, existing := range out {
			if strings.EqualFold(existing, email) {
				return
			}
		}
		out = append(out, email)
	}
	addAlias(settings.FromEmail)
	for _, alias := range settings.AliasEmails {
		addAlias(alias)
	}
	return out
}

func aliasAllowed(aliases []string, value string) bool {
	if value == "" {
		return false
	}
	for _, alias := range aliases {
		if strings.EqualFold(alias, value) {
			return true
		}
	}
	return false
}

func (a *app) resetLink(r *http.Request, token string) string {
	if a.publicURL != nil {
		u := *a.publicURL
		u.Path = "/app/reset"
		u.RawQuery = "token=" + url.QueryEscape(token)
		return u.String()
	}
	scheme := "http"
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s/app/reset?token=%s", scheme, r.Host, url.QueryEscape(token))
}
