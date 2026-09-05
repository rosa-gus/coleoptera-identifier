// Package i18n provides the CLI locale resolution and embedded message catalogs.
package i18n

import (
	"fmt"
	"strings"
)

// Locale is a supported BCP 47 language tag.
type Locale string

const (
	EnglishUS     Locale = "en-US"
	PortugueseBR  Locale = "pt-BR"
	DefaultLocale        = EnglishUS
)

// ParseLocale normalizes a supported locale name.
func ParseLocale(value string) (Locale, error) {
	normalized := strings.TrimSpace(value)
	if index := strings.IndexAny(normalized, ".@"); index >= 0 {
		normalized = normalized[:index]
	}
	normalized = strings.ToLower(strings.ReplaceAll(normalized, "_", "-"))

	switch normalized {
	case "en", "en-us":
		return EnglishUS, nil
	case "pt", "pt-br":
		return PortugueseBR, nil
	default:
		return "", fmt.Errorf("unsupported language %q: use en-US or pt-BR", value)
	}
}

// ResolveLocale applies explicit CLI and application environment preferences.
func ResolveLocale(args []string, lookupEnv func(string) (string, bool)) (Locale, error) {
	if requested, ok, err := languageFlag(args); err != nil {
		return "", err
	} else if ok {
		return ParseLocale(requested)
	}

	if value, ok := lookupEnv("COLEOPTERA_LANG"); ok && strings.TrimSpace(value) != "" {
		return ParseLocale(value)
	}
	return DefaultLocale, nil
}

func languageFlag(args []string) (string, bool, error) {
	for index, argument := range args {
		if argument == "--lang" {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
				return "", false, fmt.Errorf("flag --lang requires a value: use en-US or pt-BR")
			}
			return args[index+1], true, nil
		}
		if value, ok := strings.CutPrefix(argument, "--lang="); ok {
			if value == "" {
				return "", false, fmt.Errorf("flag --lang requires a value: use en-US or pt-BR")
			}
			return value, true, nil
		}
	}
	return "", false, nil
}
