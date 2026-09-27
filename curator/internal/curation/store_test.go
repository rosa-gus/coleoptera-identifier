package curation

import (
	"bytes"
	"encoding/csv"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testHeader = "sample_id,image_path,split,is_coleoptera,family,genus,species,label_rank,brazil_catalog_source,brazil_catalog_taxon_url,brazil_catalog_version,brazil_catalog_checked_at,model_catalog_presence,observation_id,photo_id,source,source_record_url,observed_country,photographer,image_license,observed_at,accessed_at,reviewer,notes\n"

func testInput() SaveInput {
	var photo bytes.Buffer
	_ = jpeg.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	return SaveInput{
		Fields: map[string]string{
			"sample_id": "inat-100-200", "is_coleoptera": "true", "family": "Coccinellidae",
			"genus": "Neocalvia", "species": "Neocalvia dentatofasciata", "label_rank": "species",
			"brazil_catalog_source": "CTFB", "brazil_catalog_taxon_url": "https://api.gbif.org/v1/species/240171674",
			"brazil_catalog_version": "2026-08-01", "brazil_catalog_checked_at": "2026-09-27",
			"observation_id": "100", "photo_id": "200", "source": "iNaturalist",
			"source_record_url": "https://www.inaturalist.org/observations/100",
			"photographer":      "Rosario Douglas", "image_license": "https://creativecommons.org/licenses/by/4.0/",
		},
		PhotoBytes: photo.Bytes(), Extension: ".jpg", Filename: "photo-200.jpg",
		ObservationID: 100, PhotoID: 200, TaxonomyReviewed: true, CreditsReviewed: true,
	}
}

func TestSaveCreatesPhotoCreditsAndManifestRow(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "manifest.csv"), []byte(testHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := Save(directory, testInput())
	if err != nil {
		t.Fatal(err)
	}
	if path != "images/coleoptera/Coccinellidae/photo-200.jpg" {
		t.Fatalf("unexpected path %q", path)
	}
	if _, err := os.Stat(filepath.Join(directory, path)); err != nil {
		t.Fatal(err)
	}
	credit, err := os.ReadFile(filepath.Join(directory, path+".credits.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(credit), "Rosario Douglas") || !strings.Contains(string(credit), "creativecommons.org/licenses/by/4.0") {
		t.Fatalf("credits missing: %s", credit)
	}
	file, err := os.Open(filepath.Join(directory, "manifest.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil || len(rows) != 2 {
		t.Fatalf("manifest rows %d, error %v", len(rows), err)
	}
	if len(rows[1]) != len(rows[0]) {
		t.Fatal("manifest row does not match header")
	}
}

func TestSaveRejectsSamePhotoWithDifferentFilename(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "manifest.csv"), []byte(testHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(directory, testInput()); err != nil {
		t.Fatal(err)
	}
	other := testInput()
	other.Fields["sample_id"] = "another-id"
	other.Filename = "different.jpg"
	if _, err := Save(directory, other); err == nil {
		t.Fatal("duplicate photo was accepted")
	}
	if _, err := os.Stat(filepath.Join(directory, "images", "coleoptera", "Coccinellidae", "different.jpg")); !os.IsNotExist(err) {
		t.Fatalf("unexpected duplicate file: %v", err)
	}
}

func TestValidateRequiresReviewAndCTFB(t *testing.T) {
	input := testInput()
	input.TaxonomyReviewed = false
	if err := Validate(input); err == nil {
		t.Fatal("unreviewed taxonomy was accepted")
	}
	input.TaxonomyReviewed = true
	input.Fields["brazil_catalog_taxon_url"] = ""
	if err := Validate(input); err == nil {
		t.Fatal("positive without catalog record was accepted")
	}
}

func TestSaveDeliberateNegativeWithoutCTFB(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "manifest.csv"), []byte(testHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	input := testInput()
	input.Fields["is_coleoptera"] = "false"
	input.Fields["label_rank"] = "outside_scope"
	input.Fields["brazil_catalog_taxon_url"] = ""
	input.Fields["brazil_catalog_version"] = ""
	input.Fields["brazil_catalog_checked_at"] = ""
	input.NegativeCategory = "other_insects"
	path, err := Save(directory, input)
	if err != nil {
		t.Fatal(err)
	}
	if path != "images/not_coleoptera/other_insects/photo-200.jpg" {
		t.Fatalf("negative photo saved to %s", path)
	}
}
