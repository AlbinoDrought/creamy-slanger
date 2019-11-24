package support

import "strings"

// StrAfter returns everything in `haystack` after `needle`
func StrAfter(haystack string, needle string) string {
	index := strings.Index(haystack, needle)

	if index == -1 {
		return ""
	}

	return haystack[index+1:]
}
