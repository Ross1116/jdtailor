package main

import "strings"

func normalizePastedText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\u00a0", " ")
	return strings.NewReplacer(
		"Â·", "-",
		"Â", "",
		"â€™", "'",
		"â€˜", "'",
		"â€œ", "\"",
		"â€�", "\"",
		"â€”", " - ",
		"â€“", " - ",
		"â€¢", "-",
		"â€¦", "...",
	).Replace(text)
}
