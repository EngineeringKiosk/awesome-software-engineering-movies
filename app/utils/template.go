// Package utils contains small, reusable helpers for the template
// rendering pipeline. Functions here are pure (no I/O, no global
// state) so they can be safely registered as template FuncMap
// entries and unit tested in isolation.
package utils

import (
	"strings"
	"unicode"
)

// TruncateDescription returns s shortened to roughly max runes,
// respecting word and sentence boundaries.
//
// Behaviour:
//   - If s has at most max runes, it is returned unchanged.
//   - If the rune at position max is a space or the previous rune
//     is a sentence terminator (. ! ?), the cut happens there.
//   - Otherwise the function scans forward from position max for
//     the next space or sentence terminator. A terminator is
//     included in the cut; whitespace is excluded.
//   - The result has trailing whitespace trimmed and " ..." appended.
//   - If no boundary is found before the end of s (one giant word),
//     s is returned unchanged — better than a hard mid-word cut.
//
// The function works on runes, not bytes, so multibyte content
// (emojis, accented characters) counts intuitively.
func TruncateDescription(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}

	cut := findCut(runes, max)
	if cut < 0 {
		return s
	}

	truncated := strings.TrimRightFunc(string(runes[:cut]), unicode.IsSpace)
	return truncated + " ..."
}

// findCut returns the index (exclusive) at which to slice runes,
// or -1 if no acceptable boundary exists.
func findCut(runes []rune, max int) int {
	// Boundary already at position max?
	if isSpace(runes[max]) {
		return max
	}
	if max > 0 && isSentenceEnd(runes[max-1]) {
		return max
	}

	// Scan forward for the next boundary.
	for i := max; i < len(runes); i++ {
		if isSentenceEnd(runes[i]) {
			return i + 1 // include the terminator
		}
		if isSpace(runes[i]) {
			return i // exclude the whitespace
		}
	}
	return -1
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?'
}
