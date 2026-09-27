package curation

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type SaveInput struct {
	Fields           map[string]string
	PhotoBytes       []byte
	Extension        string
	Filename         string
	NegativeCategory string
	ObservationID    int64
	PhotoID          int64
	TaxonomyReviewed bool
	CreditsReviewed  bool
}

func Save(dataDir string, input SaveInput) (string, error) {
	if err := Validate(input); err != nil {
		return "", err
	}
	var relativeDir string
	if input.Fields["is_coleoptera"] == "true" {
		relativeDir = filepath.Join("images", "coleoptera", input.Fields["family"])
	} else {
		relativeDir = filepath.Join("images", "not_coleoptera", input.NegativeCategory)
	}
	relativePath := filepath.ToSlash(filepath.Join(relativeDir, input.Filename))
	imagePath := filepath.Join(dataDir, relativePath)
	creditPath := imagePath + ".credits.md"
	manifestPath := filepath.Join(dataDir, "manifest.csv")

	file, err := os.Open(manifestPath)
	if err != nil {
		return "", err
	}
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	file.Close()
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("manifest has no header")
	}
	if err := verifyHeader(rows[0]); err != nil {
		return "", err
	}
	index := map[string]int{}
	for i, name := range rows[0] {
		index[name] = i
	}
	for _, row := range rows[1:] {
		if row[index["sample_id"]] == input.Fields["sample_id"] || row[index["image_path"]] == relativePath ||
			(row[index["source"]] == input.Fields["source"] && row[index["observation_id"]] == input.Fields["observation_id"] && row[index["photo_id"]] == input.Fields["photo_id"]) {
			return "", fmt.Errorf("photo or sample ID already exists in the manifest")
		}
	}
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		return "", err
	}
	if err := writeExclusive(imagePath, input.PhotoBytes); err != nil {
		return "", fmt.Errorf("save photo: %w", err)
	}
	credit := creditText(input, relativePath)
	if err := writeExclusive(creditPath, []byte(credit)); err != nil {
		os.Remove(imagePath)
		return "", fmt.Errorf("save credits: %w", err)
	}
	input.Fields["image_path"] = relativePath
	row := make([]string, len(rows[0]))
	for name, value := range input.Fields {
		if i, ok := index[name]; ok {
			row[i] = value
		}
	}
	rows = append(rows, row)
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.WriteAll(rows); err != nil {
		os.Remove(imagePath)
		os.Remove(creditPath)
		return "", err
	}
	temporary, err := os.CreateTemp(dataDir, ".manifest-*.csv")
	if err != nil {
		os.Remove(imagePath)
		os.Remove(creditPath)
		return "", err
	}
	defer os.Remove(temporary.Name())
	if _, err = temporary.Write(buffer.Bytes()); err == nil {
		err = temporary.Close()
	} else {
		temporary.Close()
	}
	if err == nil {
		err = os.Rename(temporary.Name(), manifestPath)
	}
	if err != nil {
		os.Remove(imagePath)
		os.Remove(creditPath)
		return "", fmt.Errorf("write manifest: %w", err)
	}
	return relativePath, nil
}

// Validate checks a proposed manifest entry before the confirmation screen.
func Validate(input SaveInput) error {
	fields := input.Fields
	if !input.TaxonomyReviewed || !input.CreditsReviewed {
		return fmt.Errorf("confirm taxonomy and photo credits before saving")
	}
	if len(input.PhotoBytes) == 0 {
		return fmt.Errorf("photo has not been loaded")
	}
	if len(input.PhotoBytes) > maxPhotoBytes {
		return fmt.Errorf("photo exceeds 10 MB")
	}
	contentType := http.DetectContentType(input.PhotoBytes)
	expectedType := map[string]string{".jpg": "image/jpeg", ".png": "image/png", ".gif": "image/gif"}[input.Extension]
	if expectedType == "" || contentType != expectedType {
		return fmt.Errorf("photo content does not match the expected image format")
	}
	if !safeName.MatchString(input.Filename) || !safeName.MatchString(fields["sample_id"]) {
		return fmt.Errorf("filename and sample ID must use letters, digits, dots, underscores, or hyphens")
	}
	if !strings.EqualFold(filepath.Ext(input.Filename), input.Extension) {
		return fmt.Errorf("filename extension must match the downloaded photo format %s", input.Extension)
	}
	if fields["photographer"] == "" || fields["image_license"] == "" || fields["source_record_url"] == "" {
		return fmt.Errorf("photographer, photo license, and source URL are required")
	}
	if fields["observation_id"] == "" || fields["photo_id"] == "" {
		return fmt.Errorf("observation and photo IDs are required")
	}
	if fields["observation_id"] != fmt.Sprint(input.ObservationID) || fields["photo_id"] != fmt.Sprint(input.PhotoID) || fields["source"] != "iNaturalist" {
		return fmt.Errorf("source and photo identifiers must match the fetched observation")
	}
	if fields["split"] != "" && fields["split"] != "calibration" && fields["split"] != "test" {
		return fmt.Errorf("split must be empty, calibration, or test")
	}
	if fields["is_coleoptera"] == "true" {
		if !safeName.MatchString(fields["family"]) || fields["species"] == "" || fields["genus"] == "" {
			return fmt.Errorf("positive examples need a family, genus, and species")
		}
		if fields["brazil_catalog_taxon_url"] == "" || fields["brazil_catalog_version"] == "" || fields["brazil_catalog_checked_at"] == "" {
			return fmt.Errorf("confirm the CTFB record, publication date, and check date")
		}
		if fields["label_rank"] != "species" {
			return fmt.Errorf("positive examples must have species rank")
		}
	} else if fields["is_coleoptera"] == "false" {
		switch input.NegativeCategory {
		case "other_insects", "arachnids", "plants", "other_organisms", "objects_no_organism":
		default:
			return fmt.Errorf("choose a negative image category")
		}
		if fields["label_rank"] != "outside_scope" {
			return fmt.Errorf("negative examples must use outside_scope rank")
		}
	} else {
		return fmt.Errorf("select whether the photo contains Coleoptera")
	}
	return nil
}

func verifyHeader(header []string) error {
	present := map[string]bool{}
	for _, name := range header {
		present[name] = true
	}
	for _, required := range []string{"sample_id", "image_path", "is_coleoptera", "family", "genus", "species", "label_rank", "brazil_catalog_taxon_url", "photographer", "image_license", "source", "source_record_url", "observation_id", "photo_id"} {
		if !present[required] {
			return fmt.Errorf("manifest is missing column %s", required)
		}
	}
	return nil
}

func writeExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Close()
	} else {
		file.Close()
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

func creditText(input SaveInput, relativePath string) string {
	fields := input.Fields
	clean := func(value string) string { return strings.NewReplacer("\n", " ", "\r", " ", "|", "\\|").Replace(value) }
	return fmt.Sprintf("# Photo credits\n\nPhoto: `%s`\n\n- Photographer: %s\n- License: %s\n- Source: %s\n",
		clean(relativePath), clean(fields["photographer"]), clean(fields["image_license"]), clean(fields["source_record_url"]))
}
