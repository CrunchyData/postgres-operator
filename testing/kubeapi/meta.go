package kubeapi

import "strings"

// SanitizeLabelValue returns a copy of value that is safe to use as a
// meta/v1.ObjectMeta label value. Invalid characters are removed or replaced
// with dashes.
func SanitizeLabelValue(value string) string {
	// "must be no more than 63 characters"
	if len(value) > 63 {
		value = value[:63]
	}

	// "a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.'"
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == '.' ||
			('A' <= r && r <= 'Z') ||
			('a' <= r && r <= 'z') ||
			('0' <= r && r <= '9') {
			return r
		}
		return '-'
	}, value)
}
