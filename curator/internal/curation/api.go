package curation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maxPhotoBytes = 10 << 20

type Client struct {
	HTTP    *http.Client
	BaseURL string
}

type Photo struct {
	ID          int64  `json:"id"`
	URL         string `json:"url"`
	Attribution string `json:"attribution"`
	LicenseCode string `json:"license_code"`
}

type Observation struct {
	ID         int64
	Species    string
	Genus      string
	Family     string
	Order      string
	Rank       string
	ObservedOn string
	Country    string
	Photos     []Photo
}

func NewClient() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 20 * time.Second},
		BaseURL: "https://api.inaturalist.org/v1",
	}
}

func (c *Client) Fetch(ctx context.Context, id string) (Observation, error) {
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(id) {
		return Observation{}, fmt.Errorf("observation ID must contain only digits")
	}
	var payload struct {
		Results []struct {
			ID         int64  `json:"id"`
			ObservedOn string `json:"observed_on"`
			PlaceGuess string `json:"place_guess"`
			Taxon      struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
				Rank string `json:"rank"`
			} `json:"taxon"`
			Photos []Photo `json:"photos"`
		} `json:"results"`
	}
	if err := c.getJSON(ctx, "/observations/"+id, &payload); err != nil {
		return Observation{}, err
	}
	if len(payload.Results) != 1 {
		return Observation{}, fmt.Errorf("observation %s was not found", id)
	}
	entry := payload.Results[0]
	if len(entry.Photos) == 0 {
		return Observation{}, fmt.Errorf("observation %s has no photos", id)
	}
	result := Observation{
		ID:         entry.ID,
		Rank:       entry.Taxon.Rank,
		ObservedOn: entry.ObservedOn,
		Country:    countryFromPlace(entry.PlaceGuess),
		Photos:     entry.Photos,
	}
	if entry.Taxon.Rank == "species" {
		result.Species = entry.Taxon.Name
	}
	if entry.Taxon.ID != 0 {
		var taxonPayload struct {
			Results []struct {
				Name      string `json:"name"`
				Rank      string `json:"rank"`
				Ancestors []struct {
					Name string `json:"name"`
					Rank string `json:"rank"`
				} `json:"ancestors"`
			} `json:"results"`
		}
		if err := c.getJSON(ctx, fmt.Sprintf("/taxa/%d", entry.Taxon.ID), &taxonPayload); err != nil {
			return Observation{}, fmt.Errorf("taxonomic lineage: %w", err)
		}
		if len(taxonPayload.Results) > 0 {
			lineage := taxonPayload.Results[0]
			for _, ancestor := range lineage.Ancestors {
				setRank(&result, ancestor.Rank, ancestor.Name)
			}
			setRank(&result, lineage.Rank, lineage.Name)
		}
	}
	return result, nil
}

func setRank(observation *Observation, rank, name string) {
	switch rank {
	case "order":
		observation.Order = name
	case "family":
		observation.Family = name
	case "genus":
		observation.Genus = name
	case "species":
		observation.Species = name
	}
}

func countryFromPlace(place string) string {
	parts := strings.Split(place, ",")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func (c *Client) getJSON(ctx context.Context, path string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "coleoptera-curator/0.1")
	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("iNaturalist returned HTTP %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(destination)
}

// FetchPhoto downloads the selected photo only, after the user chooses it.
func (c *Client) FetchPhoto(ctx context.Context, photo Photo) ([]byte, string, error) {
	photoURL := strings.Replace(photo.URL, "/square.", "/large.", 1)
	parsed, err := url.Parse(photoURL)
	if err != nil || parsed.Scheme != "https" || !trustedPhotoHost(parsed.Hostname()) {
		return nil, "", fmt.Errorf("the photo URL is not a trusted iNaturalist HTTPS URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, photoURL, nil)
	if err != nil {
		return nil, "", err
	}
	photoClient := *c.HTTP
	photoClient.CheckRedirect = func(next *http.Request, previous []*http.Request) error {
		if len(previous) >= 5 || next.URL.Scheme != "https" || !trustedPhotoHost(next.URL.Hostname()) {
			return fmt.Errorf("photo redirect is not an allowed iNaturalist HTTPS URL")
		}
		return nil
	}
	response, err := photoClient.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.Request == nil || response.Request.URL.Scheme != "https" || !trustedPhotoHost(response.Request.URL.Hostname()) {
		return nil, "", fmt.Errorf("photo download redirected outside trusted iNaturalist hosts")
	}
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("photo download returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxPhotoBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxPhotoBytes {
		return nil, "", fmt.Errorf("photo exceeds 10 MB")
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg":
		return data, ".jpg", nil
	case "image/png":
		return data, ".png", nil
	case "image/gif":
		return data, ".gif", nil
	default:
		return nil, "", fmt.Errorf("unsupported photo format: %s", contentType)
	}
}

func trustedPhotoHost(host string) bool {
	host = strings.ToLower(host)
	return host == "inaturalist.org" || strings.HasSuffix(host, ".inaturalist.org") ||
		host == "inaturalist-open-data.s3.amazonaws.com"
}

func PhotoAuthor(photo Photo) string {
	value := strings.TrimSpace(photo.Attribution)
	value = strings.TrimPrefix(value, "(c) ")
	value = strings.TrimPrefix(value, "© ")
	if before, _, ok := strings.Cut(value, ", some rights reserved"); ok {
		return before
	}
	if before, _, ok := strings.Cut(value, ", all rights reserved"); ok {
		return before
	}
	return value
}

func LicenseURL(code string) string {
	if code == "cc0" {
		return "https://creativecommons.org/publicdomain/zero/1.0/"
	}
	if strings.HasPrefix(code, "cc-") {
		return "https://creativecommons.org/licenses/" + strings.TrimPrefix(code, "cc-") + "/4.0/"
	}
	return ""
}
