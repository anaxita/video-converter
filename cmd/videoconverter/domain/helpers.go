package domain

import "strings"

func FormatFileName(n string) string {
	trim := strings.TrimSpace(n)
	return strings.ReplaceAll(trim, " ", "-")
}
