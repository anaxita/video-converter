package domain

import (
	"regexp"
	"strings"
)

func FormatFileName(n string) string {
	s := strings.ReplaceAll(n, ".mp4", "")
	trim := strings.TrimSpace(s)
	r := regexp.MustCompile(`([^\\w\\-А-ЯЁа-яё0-9])`)
	clear := r.ReplaceAllString(trim, "_")
	return clear + ".mp4"
}
