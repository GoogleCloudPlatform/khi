package generator

import (
	"regexp"
	"strings"
)

var regexToRemoveInGithubAnchorHash = regexp.MustCompile(`[^a-z0-9\s\-]+`)
var regexToHyphenInGithubAnchorHash = regexp.MustCompile(`\s+`)

// ToGithubAnchorHash convert the given text of header to the hash of anchor link.
func ToGithubAnchorHash(text string) string {
	return regexToRemoveInGithubAnchorHash.ReplaceAllString(regexToHyphenInGithubAnchorHash.ReplaceAllString(strings.ToLower(strings.TrimSpace(text)), "-"), "")
}
