// Package primaryredirect redirects requests arriving at alternate hosts
// to a single primary origin.
package primaryredirect

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Handler adopts the origin of originURL
// as the primary origin for the returned handler.
// Requests for this origin are passed through to h.
// Requests for any other origin are served a redirect
// to the primary origin, using the path and query
// from the request URL.
//
// Only the scheme and host parts of originURL are used.
// Other components are ignored.
//
// If originURL is empty, Handler returns h unmodified.
// If originURL is not a valid http or https URL, Handler panics.
func Handler(originURL string, h http.Handler) http.Handler {
	if originURL == "" {
		return h
	}
	primaryScheme, primaryHost := parseOrigin(originURL)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.EqualFold(req.Host, primaryHost) {
			h.ServeHTTP(w, req)
			return
		}
		u := url.URL{
			Scheme:   primaryScheme,
			Host:     primaryHost,
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
		}
		http.Redirect(w, req, u.String(), http.StatusTemporaryRedirect)
	})
}

func parseOrigin(u string) (scheme, host string) {
	parsed, err := url.Parse(u)
	if err != nil {
		panic(fmt.Errorf("primaryredirect: bad url: %w", err))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		panic(fmt.Errorf("primaryredirect: bad scheme %s in %q; want http or https", parsed.Scheme, u))
	}
	if parsed.Host == "" {
		panic(fmt.Errorf("primaryredirect: empty host in %q", u))
	}
	return parsed.Scheme, parsed.Host
}
