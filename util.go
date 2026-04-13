package main

import (
	"regexp"
	"strings"
)

var (
	reEmail  = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	reNormal = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

func parseStringSlice(s string) []string {
	val := strings.TrimSpace(s)
	if val == "" {
		return nil
	}

	// Split by comma and trim spaces
	parts := strings.Split(val, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

func isValidEmail(email string) bool {
	// RFC 5322 simplified regex for general email validation
	return reEmail.MatchString(email)
}

func normalize(s string) string {
	// s = strings.ToLower(s) maybe?
	s = reNormal.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}
