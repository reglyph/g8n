package naming

import (
	"strings"
	"unicode"
)

func GoFieldName(key string) string {
	var b strings.Builder
	var part []rune

	flush := func() {
		if len(part) == 0 {
			return
		}

		for i, r := range part {
			part[i] = unicode.ToLower(r)
		}

		if unicode.IsLetter(part[0]) {
			part[0] = unicode.ToUpper(part[0])
		}

		b.WriteString(string(part))
		part = part[:0]
	}

	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			part = append(part, unicode.ToLower(r))
		} else {
			flush()
		}
	}

	flush()
	return b.String()
}
