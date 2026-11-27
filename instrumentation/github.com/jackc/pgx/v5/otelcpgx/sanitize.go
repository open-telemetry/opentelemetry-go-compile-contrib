// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pgx

import (
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

const rawQueryEnv = "OTEL_GO_INSTRUMENTATION_PGX_RAW_QUERY"

// captureRawQuery returns true if raw, unsanitized query text has been explicitly opted into.
func captureRawQuery() bool {
	return strings.EqualFold(os.Getenv(rawQueryEnv), "true")
}

func skipLineComment(chars []rune, i int, b *strings.Builder) int {
	n := len(chars)
	for i < n && chars[i] != '\n' {
		b.WriteRune(chars[i])
		i++
	}
	return i
}

func skipBlockComment(chars []rune, i int, b *strings.Builder) int {
	n := len(chars)
	_, _ = b.WriteString("/*")
	i += 2
	depth := 1
	for i < n && depth > 0 {
		switch {
		case i+1 < n && chars[i] == '/' && chars[i+1] == '*':
			depth++
			_, _ = b.WriteString("/*")
			i += 2
		case i+1 < n && chars[i] == '*' && chars[i+1] == '/':
			depth--
			_, _ = b.WriteString("*/")
			i += 2
		default:
			b.WriteRune(chars[i])
			i++
		}
	}
	return i
}

// skipDoubleQuotedIdentifier preserves a double-quoted SQL identifier (e.g. "table", "o'brien", "col""name")
// verbatim in the output so that apostrophes or other characters inside it are not treated as string literals.
func skipDoubleQuotedIdentifier(chars []rune, i int, b *strings.Builder) int {
	n := len(chars)
	b.WriteRune('"')
	i++
	for i < n {
		if chars[i] == '"' {
			b.WriteRune('"')
			if i+1 < n && chars[i+1] == '"' {
				b.WriteRune('"')
				i += 2
				continue
			}
			return i + 1
		}
		b.WriteRune(chars[i])
		i++
	}
	return i
}

// sanitizeSingleQuote sanitizes a single-quoted string literal.
// In PostgreSQL with standard_conforming_strings=on (default), a backslash inside an
// ordinary '...' literal is NOT an escape character (only ” escapes single quotes).
// Backslash escaping is only active inside escape string constants E'...' or e'...'.
func sanitizeSingleQuote(chars []rune, i int, b *strings.Builder, isEscape bool) int {
	n := len(chars)
	b.WriteRune('?')
	i++
	for i < n {
		switch {
		case chars[i] == '\'':
			if i+1 < n && chars[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		case isEscape && chars[i] == '\\' && i+1 < n:
			i += 2
		default:
			i++
		}
	}
	return i
}

func sanitizeDollar(chars []rune, i int, b *strings.Builder) int {
	n := len(chars)
	if i+1 < n && unicode.IsDigit(chars[i+1]) {
		b.WriteRune('$')
		i++
		for i < n && unicode.IsDigit(chars[i]) {
			b.WriteRune(chars[i])
			i++
		}
		return i
	}

	tagEnd := -1
	for j := i + 1; j < n; j++ {
		if chars[j] == '$' {
			tagEnd = j
			break
		}
		if unicode.IsSpace(chars[j]) || chars[j] == ';' || chars[j] == ',' || chars[j] == '(' {
			break
		}
	}
	if tagEnd == -1 {
		b.WriteRune('$')
		return i + 1
	}

	tag := string(chars[i : tagEnd+1])
	b.WriteRune('?')
	i = tagEnd + 1
	rest := string(chars[i:])
	before, _, ok := strings.Cut(rest, tag)
	if ok {
		return i + utf8.RuneCountInString(before) + utf8.RuneCountInString(tag)
	}
	return n
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func skipHexDigits(chars []rune, i int) int {
	n := len(chars)
	for i < n {
		c := chars[i]
		if isHexDigit(c) || (c == '_' && i+1 < n && isHexDigit(chars[i+1])) {
			i++
			continue
		}
		break
	}
	return i
}

func skipBinaryDigits(chars []rune, i int) int {
	n := len(chars)
	for i < n {
		c := chars[i]
		if c == '0' || c == '1' || (c == '_' && i+1 < n && (chars[i+1] == '0' || chars[i+1] == '1')) {
			i++
			continue
		}
		break
	}
	return i
}

func skipOctalDigits(chars []rune, i int) int {
	n := len(chars)
	for i < n {
		c := chars[i]
		if (c >= '0' && c <= '7') || (c == '_' && i+1 < n && chars[i+1] >= '0' && chars[i+1] <= '7') {
			i++
			continue
		}
		break
	}
	return i
}

const radixPrefixLen = 2

func skipRadixNumber(chars []rune, i int) int {
	n := len(chars)
	if chars[i] != '0' || i+1 >= n {
		return i
	}
	switch chars[i+1] {
	case 'x', 'X':
		if i+radixPrefixLen < n && isHexDigit(chars[i+radixPrefixLen]) {
			return skipHexDigits(chars, i+radixPrefixLen)
		}
	case 'b', 'B':
		if i+radixPrefixLen < n && (chars[i+radixPrefixLen] == '0' || chars[i+radixPrefixLen] == '1') {
			return skipBinaryDigits(chars, i+radixPrefixLen)
		}
	case 'o', 'O':
		if i+radixPrefixLen < n && chars[i+radixPrefixLen] >= '0' && chars[i+radixPrefixLen] <= '7' {
			return skipOctalDigits(chars, i+radixPrefixLen)
		}
	}
	return i
}

func skipDecimalNumber(chars []rune, i int) int {
	n := len(chars)
	for i < n {
		c := chars[i]
		switch {
		case unicode.IsDigit(c):
			i++
		case c == '_' && i+1 < n && (unicode.IsDigit(chars[i+1]) || chars[i+1] == '.'):
			i++
		case c == '.' && i+1 < n && unicode.IsDigit(chars[i+1]):
			i++
		case (c == 'e' || c == 'E') && i+1 < n:
			next := chars[i+1]
			switch {
			case unicode.IsDigit(next):
				i += 2
			case (next == '+' || next == '-') && i+2 < n && unicode.IsDigit(chars[i+2]):
				i += 3
			default:
				return i
			}
		default:
			return i
		}
	}
	return i
}

func sanitizeNumber(chars []rune, i int, b *strings.Builder) int {
	prevIsIdent := i > 0 && (unicode.IsLetter(chars[i-1]) || chars[i-1] == '_' || chars[i-1] == '$')
	if prevIsIdent {
		b.WriteRune(chars[i])
		return i + 1
	}

	b.WriteRune('?')

	if next := skipRadixNumber(chars, i); next > i {
		return next
	}
	return skipDecimalNumber(chars, i)
}

// sanitizeSQL sanitizes literal strings, numbers, and dollar-quoted literals from SQL queries
// to prevent leaking sensitive values in telemetry spans.
func sanitizeSQL(query string) string {
	if captureRawQuery() {
		return query
	}

	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(trimmed))
	chars := []rune(trimmed)
	n := len(chars)
	i := 0

	for i < n {
		c := chars[i]
		switch {
		case c == '-' && i+1 < n && chars[i+1] == '-':
			i = skipLineComment(chars, i, &b)
		case c == '/' && i+1 < n && chars[i+1] == '*':
			i = skipBlockComment(chars, i, &b)
		case c == '"':
			i = skipDoubleQuotedIdentifier(chars, i, &b)
		case c == '\'':
			isEscape := i > 0 && (chars[i-1] == 'E' || chars[i-1] == 'e') &&
				(i == 1 || (!unicode.IsLetter(chars[i-2]) && !unicode.IsDigit(chars[i-2]) && chars[i-2] != '_'))
			i = sanitizeSingleQuote(chars, i, &b, isEscape)
		case c == '$':
			i = sanitizeDollar(chars, i, &b)
		case unicode.IsDigit(c):
			i = sanitizeNumber(chars, i, &b)
		default:
			b.WriteRune(c)
			i++
		}
	}

	return b.String()
}
