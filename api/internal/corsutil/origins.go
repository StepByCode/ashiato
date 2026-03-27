package corsutil

import (
	"net/url"
	"strings"
)

var defaultAllowedOrigins = []string{
	"https://backatage.stepbycode.work",
	"https://*.stepbycode.work",
	"https://*.vercel.app",
	"http://localhost:3000",
	"http://127.0.0.1:3000",
}

func BuildAllowedOrigins(configured []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(defaultAllowedOrigins)+len(configured))

	for _, origin := range append(defaultAllowedOrigins, configured...) {
		value := strings.TrimSpace(origin)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func IsOriginAllowed(origin string, allowed []string) bool {
	normalizedOrigin, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}

	for _, candidate := range allowed {
		if matchesOriginPattern(normalizedOrigin, candidate) {
			return true
		}
	}
	return false
}

type normalizedPattern struct {
	scheme   string
	host     string
	wildcard bool
}

func matchesOriginPattern(origin, pattern string) bool {
	normalizedPattern, ok := normalizePattern(pattern)
	if !ok {
		return false
	}

	if !normalizedPattern.wildcard {
		return origin == normalizedPattern.scheme+"://"+normalizedPattern.host
	}
	if normalizedPattern.scheme != "" && !strings.HasPrefix(origin, normalizedPattern.scheme+"://") {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}

	host := strings.ToLower(u.Hostname())
	return host == normalizedPattern.host || strings.HasSuffix(host, "."+normalizedPattern.host)
}

func normalizePattern(pattern string) (normalizedPattern, bool) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return normalizedPattern{}, false
	}

	if strings.HasPrefix(pattern, "*.") {
		host := strings.ToLower(strings.TrimPrefix(pattern, "*."))
		return normalizedPattern{host: host, wildcard: true}, host != ""
	}

	if strings.Contains(pattern, "://") {
		u, err := url.Parse(pattern)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return normalizedPattern{}, false
		}
		host := strings.ToLower(u.Hostname())
		if strings.HasPrefix(host, "*.") {
			host = strings.TrimPrefix(host, "*.")
			return normalizedPattern{
				scheme:   strings.ToLower(u.Scheme),
				host:     host,
				wildcard: true,
			}, host != ""
		}

		return normalizedPattern{
			scheme: strings.ToLower(u.Scheme),
			host:   host,
		}, true
	}

	host := strings.ToLower(pattern)
	return normalizedPattern{host: host}, host != ""
}

func normalizeOrigin(origin string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Hostname()), true
}
