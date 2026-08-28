package pages

import "strconv"

func narrationStatus(narration string) string {
	if narration == "" {
		return "NO NARRATION YET"
	}

	return strconv.Itoa(runeLen(narration)) + " CHARS"
}
