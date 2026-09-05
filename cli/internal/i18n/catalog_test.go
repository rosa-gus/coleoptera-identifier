package i18n

import (
	"reflect"
	"sort"
	"testing"
)

func TestCatalogsHaveTheSameKeys(t *testing.T) {
	english := sortedKeys(catalogs[EnglishUS])
	portuguese := sortedKeys(catalogs[PortugueseBR])
	if !reflect.DeepEqual(english, portuguese) {
		t.Fatalf("message catalog keys differ:\nen-US: %v\npt-BR: %v", english, portuguese)
	}
}

func TestResolveLocalePrecedenceAndNormalization(t *testing.T) {
	environment := map[string]string{
		"COLEOPTERA_LANG": "pt_BR.UTF-8",
		"LANG":            "en_US.UTF-8",
	}
	lookup := func(name string) (string, bool) {
		value, ok := environment[name]
		return value, ok
	}

	locale, err := ResolveLocale([]string{"ui", "--lang=en-us"}, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if locale != EnglishUS {
		t.Fatalf("expected CLI flag to win, got %s", locale)
	}

	locale, err = ResolveLocale(nil, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if locale != PortugueseBR {
		t.Fatalf("expected application environment to win, got %s", locale)
	}
}

func TestDefaultLocaleIsEnglish(t *testing.T) {
	locale, err := ResolveLocale(nil, func(name string) (string, bool) {
		if name == "LANG" {
			return "pt_BR.UTF-8", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if locale != EnglishUS {
		t.Fatalf("expected en-US, got %s", locale)
	}
}

func TestPercentUsesLocaleDecimalSeparator(t *testing.T) {
	if got := NewLocalizer(EnglishUS).Percent(0.781); got != "78.1%" {
		t.Fatalf("unexpected English percentage: %s", got)
	}
	if got := NewLocalizer(PortugueseBR).Percent(0.781); got != "78,1%" {
		t.Fatalf("unexpected Portuguese percentage: %s", got)
	}
}

func sortedKeys(messages map[string]string) []string {
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
