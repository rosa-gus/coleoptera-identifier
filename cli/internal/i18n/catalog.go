package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Key identifies one user-interface message.
type Key string

const (
	CLIRootShort         Key = "cli.root.short"
	CLIAPIURLHelp        Key = "cli.flag.api_url"
	CLITimeoutHelp       Key = "cli.flag.timeout"
	CLILanguageHelp      Key = "cli.flag.language"
	CLIIdentifyShort     Key = "cli.identify.short"
	CLIOutputHelp        Key = "cli.identify.output"
	CLIUIShort           Key = "cli.ui.short"
	TUISelectImage       Key = "tui.select_image"
	TUINavigationHelp    Key = "tui.navigation_help"
	TUIAnalyzing         Key = "tui.analyzing"
	TUISendingImage      Key = "tui.sending_image"
	TUICancelHelp        Key = "tui.cancel_help"
	TUIResult            Key = "tui.result"
	TUIResultHelp        Key = "tui.result_help"
	TUIFailure           Key = "tui.failure"
	TUIFailureHelp       Key = "tui.failure_help"
	TUIEmptyDirectory    Key = "tui.empty_directory"
	ResultIdentification Key = "result.identification"
	ResultCalibration    Key = "result.calibration"
	ResultUncalibrated   Key = "result.uncalibrated"
	ResultSuggestedRank  Key = "result.suggested_rank"
	ResultOrder          Key = "result.order"
	ResultDetectedClass  Key = "result.detected_class"
	ResultSuggestedOrder Key = "result.suggested_order"
	ResultAlternatives   Key = "result.alternatives"
	ResultModelScore     Key = "result.model_score"
	ResultNotProvided    Key = "result.not_provided"
	ResultTaxonomy       Key = "result.taxonomy"
	RankKingdom          Key = "rank.kingdom"
	RankPhylum           Key = "rank.phylum"
	RankClass            Key = "rank.class"
	RankOrder            Key = "rank.order"
	RankFamily           Key = "rank.family"
	RankGenus            Key = "rank.genus"
	RankSpecies          Key = "rank.species"
)

//go:embed messages/*.json
var catalogFiles embed.FS

var catalogs = mustLoadCatalogs()

// Localizer reads messages from one immutable embedded catalog.
type Localizer struct {
	locale   Locale
	messages map[string]string
}

// NewLocalizer constructs a localizer for a supported locale.
func NewLocalizer(locale Locale) *Localizer {
	messages, ok := catalogs[locale]
	if !ok {
		locale = DefaultLocale
		messages = catalogs[locale]
	}
	return &Localizer{locale: locale, messages: messages}
}

// Locale returns the localizer's canonical language tag.
func (localizer *Localizer) Locale() Locale {
	return localizer.locale
}

// Text returns a translated message and falls back to English for missing keys.
func (localizer *Localizer) Text(key Key) string {
	if message, ok := localizer.messages[string(key)]; ok {
		return message
	}
	if message, ok := catalogs[DefaultLocale][string(key)]; ok {
		return message
	}
	return string(key)
}

// Percent formats a percentage according to the selected locale.
func (localizer *Localizer) Percent(value float64) string {
	formatted := fmt.Sprintf("%.1f%%", value*100)
	if localizer.locale == PortugueseBR {
		return strings.Replace(formatted, ".", ",", 1)
	}
	return formatted
}

func mustLoadCatalogs() map[Locale]map[string]string {
	loaded := make(map[Locale]map[string]string, 2)
	for _, locale := range []Locale{EnglishUS, PortugueseBR} {
		content, err := catalogFiles.ReadFile("messages/" + string(locale) + ".json")
		if err != nil {
			panic(fmt.Sprintf("load %s message catalog: %v", locale, err))
		}
		var messages map[string]string
		if err := json.Unmarshal(content, &messages); err != nil {
			panic(fmt.Sprintf("parse %s message catalog: %v", locale, err))
		}
		loaded[locale] = messages
	}
	return loaded
}
