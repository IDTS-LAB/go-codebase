package slug

import (
	"regexp"
	"strings"
)

var (
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
	multiDash   = regexp.MustCompile(`-{2,}`)
)

func Make(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	s = multiDash.ReplaceAllString(s, "-")
	return s
}
