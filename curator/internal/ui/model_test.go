package ui

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/rosa-gus/coleoptera_identifier/curator/internal/curation"
)

func reviewModel() Model {
	model := New(nil, "", "")
	observation := curation.Observation{
		ID: 339647945, Species: "Neocalvia dentatofasciata", Genus: "Neocalvia",
		Family: "Coccinellidae", Order: "Coleoptera", Country: "Argentina",
		Photos: []curation.Photo{{ID: 617908244, Attribution: "(c) Rosario Douglas, some rights reserved (CC BY)", LicenseCode: "cc-by"}},
	}
	model.phase = review
	model.observation = observation
	model.populate()
	model.extension = ".jpg"
	model.preview = "PHOTO"
	model.setPhotoFields()
	return model
}

func TestPasteIntoObservationID(t *testing.T) {
	model := New(nil, "", "")
	updated, _ := model.Update(tea.PasteMsg{Content: "339647945"})
	model = updated.(Model)
	if got := model.input.Value(); got != "339647945" {
		t.Fatalf("pasted observation ID = %q", got)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("Ctrl+V did not request clipboard contents")
	}
}

func TestPasteIntoReviewField(t *testing.T) {
	model := reviewModel()
	model.groupIndex, model.fieldIndex = 2, 1 // Empty CTFB taxon URL.
	updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)
	if !model.editing {
		t.Fatal("field editor did not open")
	}
	updated, _ = model.Update(tea.PasteMsg{Content: "https://example.org/taxon"})
	model = updated.(Model)
	if got := model.input.Value(); got != "https://example.org/taxon" {
		t.Fatalf("pasted field value = %q", got)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("Ctrl+V did not request clipboard contents while editing")
	}
}

func TestLoadingIndicatorDuringObservationAndPhotoFetch(t *testing.T) {
	model := New(nil, "", "339647945")
	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)
	if !model.busy || cmd == nil || !strings.Contains(model.View().Content, "Fetching observation...") {
		t.Fatal("observation fetch did not show a loading state")
	}
	firstFrame := model.spinner.View()
	updated, cmd = model.Update(model.spinner.Tick())
	model = updated.(Model)
	if cmd == nil || model.spinner.View() == firstFrame {
		t.Fatal("loading indicator did not advance")
	}

	updated, cmd = model.Update(observationMsg{observation: reviewModel().observation})
	model = updated.(Model)
	if !model.busy || cmd == nil || !strings.Contains(model.View().Content, "Fetching photo...") {
		t.Fatal("photo fetch did not show a loading state")
	}
	updated, _ = model.Update(photoMsg{preview: "PHOTO", extension: ".jpg"})
	model = updated.(Model)
	if model.busy || strings.Contains(model.View().Content, "Fetching photo...") {
		t.Fatal("loading state remained after photo fetch")
	}
}

func TestReviewShowsPhotoAndFetchedCredits(t *testing.T) {
	model := reviewModel()
	view := model.View().Content
	for _, expected := range []string{"Observation 339647945", "PHOTO 1/1", "Rosario Douglas", "Neocalvia"} {
		if !strings.Contains(view, expected) && expected != "Rosario Douglas" {
			t.Fatalf("view missing %q", expected)
		}
	}
	if model.fields["photographer"] != "Rosario Douglas" || model.fields["photo_id"] != "617908244" {
		t.Fatalf("credits not populated: %+v", model.fields)
	}
}

func TestReviewPhotoBrowserActionUsesSelectedPhoto(t *testing.T) {
	model := reviewModel()
	model.observation.Photos = append(model.observation.Photos, curation.Photo{ID: 617908245})
	model.photoIndex = 1
	view := model.View().Content
	if !strings.Contains(view, "Open full photo") || !strings.Contains(view, photoPageURL(617908245)) {
		t.Fatal("photo browser button does not link to the selected photo")
	}
	updated, cmd := model.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	model = updated.(Model)
	if cmd == nil || model.phase != review {
		t.Fatal("browser shortcut did not keep the review open")
	}
	updated, _ = model.Update(openedPhotoMsg{err: errors.New("browser unavailable")})
	model = updated.(Model)
	if !strings.Contains(model.message, "browser unavailable") {
		t.Fatal("browser error was not shown")
	}
}

func TestReviewDoesNotConfirmSaveWithoutCatalog(t *testing.T) {
	model := reviewModel()
	var photo bytes.Buffer
	if err := jpeg.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	model.photoBytes = photo.Bytes()
	model.taxonomyReviewed = true
	model.creditsReviewed = true
	updated, _ := model.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	model = updated.(Model)
	if model.phase != review || !strings.Contains(model.message, "CTFB record") {
		t.Fatalf("incomplete photo was accepted: phase=%v message=%q", model.phase, model.message)
	}
}

func TestReviewFitsStandardTerminal(t *testing.T) {
	model := reviewModel()
	model.width, model.height = 80, 24
	model.preview = strings.Repeat("████████████████████████\n", 4) + "████████████████████████"
	if height := lipgloss.Height(model.View().Content); height > model.height {
		t.Fatalf("review height %d exceeds terminal height %d", height, model.height)
	}
}
