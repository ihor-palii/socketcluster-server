package utils

import (
	"golang.org/x/text/language"
)

func GetLanguage(s string) string {
	lang, err := language.Parse(s)
	if err != nil {
		return ""
	}

	base, confidence := lang.Base()
	if confidence < language.High {
		return ""
	}
	return base.String()
}