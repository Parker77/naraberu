package main

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var quoteMap = map[rune]rune{
	'\u201c': '"',  // "
	'\u201d': '"',  // "
	'\u2018': '\'', // '
	'\u2019': '\'', // '
	'\u2013': '-',  // –
	'\u2014': '-',  // —
	'\u00d7': 'x',  // ×
	'\u2026': '.',  // …
	'\u300c': '[',  // 「
	'\u300d': ']',  // 」
	'\u300e': '[',  // 『
	'\u300f': ']',  // 』
	'\uff01': '!',  // ！
	'\uff1f': '?',  // ？
	'\u3001': ',',  // 、
	'\u3002': '.',  // 。
	'\uff0c': ',',  // ，
	'\uff1a': ':',  // ：
	'\uff1b': ';',  // ；
}

var stripCombining = transform.Chain(
	norm.NFKD,
	runes.Remove(runes.In(unicode.Mn)),
)

func normalizeSearch(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if mapped, ok := quoteMap[r]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	result, _, _ := transform.String(stripCombining, b.String())
	return result
}
