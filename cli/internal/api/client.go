// Package api implements the HTTP contract between the Go client and backend.
package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxImageSize    = 10 * 1024 * 1024
	maxResponseSize = 1024 * 1024
)

var allowedExtensions = map[string]struct{}{
	".bmp":  {},
	".gif":  {},
	".jpeg": {},
	".jpg":  {},
	".png":  {},
	".webp": {},
}

// Client sends images to the backend without running models locally.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Identification supports both the current contract and its taxonomic evolution.
type Identification struct {
	Identification             string      `json:"identification,omitempty"`
	Confidence                 float64     `json:"confidence"`
	ConfidenceCalibrated       bool        `json:"confidence_calibrated"`
	ConfidenceInterpretation   string      `json:"confidence_interpretation,omitempty"`
	Order                      string      `json:"order,omitempty"`
	DetectedClass              string      `json:"detected_class,omitempty"`
	DetectedOrder              string      `json:"detected_order,omitempty"`
	Status                     string      `json:"status,omitempty"`
	Prediction                 string      `json:"prediction,omitempty"`
	ReachedRank                string      `json:"reached_rank,omitempty"`
	Taxonomy                   *Taxonomy   `json:"taxonomy,omitempty"`
	Candidates                 []Candidate `json:"candidates,omitempty"`
	RequiresExpertConfirmation bool        `json:"requires_expert_confirmation,omitempty"`
	ModelVersion               string      `json:"model_version,omitempty"`
	ScoreScope                 string      `json:"score_scope,omitempty"`
	CandidateScope             string      `json:"candidate_scope,omitempty"`
	Disclaimer                 string      `json:"disclaimer,omitempty"`
}

// Taxonomy describes the classification returned by the taxonomic backend.
type Taxonomy struct {
	Kingdom string `json:"kingdom,omitempty"`
	Phylum  string `json:"phylum,omitempty"`
	Class   string `json:"class,omitempty"`
	Order   string `json:"order,omitempty"`
	Family  string `json:"family,omitempty"`
	Genus   string `json:"genus,omitempty"`
	Species string `json:"species,omitempty"`
}

// Candidate is an alternative identification ordered by model score.
type Candidate struct {
	ScientificName         string   `json:"scientific_name"`
	CommonName             string   `json:"common_name,omitempty"`
	Confidence             float64  `json:"confidence"`
	Taxonomy               Taxonomy `json:"taxonomy"`
	GBIFTaxonKey           string   `json:"gbif_taxon_key,omitempty"`
	AcceptedScientificName string   `json:"accepted_scientific_name,omitempty"`
	TaxonomicStatus        string   `json:"taxonomic_status,omitempty"`
}

type identifyRequest struct {
	Image string `json:"image"`
}

type errorResponse struct {
	Detail json.RawMessage `json:"detail"`
}

// NewClient validates the URL and creates a client with an explicit timeout.
func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid API URL: %q", baseURL)
	}
	if timeout <= 0 {
		return nil, errors.New("timeout must be greater than zero")
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// Identify reads a local image and sends it using the current Base64 contract.
func (c *Client) Identify(ctx context.Context, imagePath string) (Identification, error) {
	image, err := readImage(imagePath)
	if err != nil {
		return Identification{}, err
	}

	body, err := json.Marshal(identifyRequest{Image: base64.StdEncoding.EncodeToString(image)})
	if err != nil {
		return Identification{}, fmt.Errorf("could not prepare the request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/identify",
		bytes.NewReader(body),
	)
	if err != nil {
		return Identification{}, fmt.Errorf("could not create the request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ColeopteraCLI/0.1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Identification{}, fmt.Errorf("could not access the API: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return Identification{}, fmt.Errorf("could not read the API response: %w", err)
	}
	if len(responseBody) > maxResponseSize {
		return Identification{}, errors.New("the API response exceeds the 1 MB limit")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Identification{}, parseAPIError(response.StatusCode, responseBody)
	}

	var result Identification
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return Identification{}, fmt.Errorf("the API returned invalid JSON: %w", err)
	}
	return result, nil
}

func readImage(imagePath string) ([]byte, error) {
	cleanPath := filepath.Clean(imagePath)
	extension := strings.ToLower(filepath.Ext(cleanPath))
	if _, ok := allowedExtensions[extension]; !ok {
		return nil, fmt.Errorf("unsupported image format %q", extension)
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the image: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not inspect the image: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("the selected path is not a regular file")
	}
	if info.Size() == 0 {
		return nil, errors.New("the image is empty")
	}
	if info.Size() > MaxImageSize {
		return nil, errors.New("the image exceeds the 10 MB limit")
	}

	content, err := io.ReadAll(io.LimitReader(file, MaxImageSize+1))
	if err != nil {
		return nil, fmt.Errorf("could not read the image: %w", err)
	}
	if len(content) > MaxImageSize {
		return nil, errors.New("the image exceeds the 10 MB limit")
	}
	return content, nil
}

func parseAPIError(status int, body []byte) error {
	message := strings.TrimSpace(string(body))
	var apiError errorResponse
	if json.Unmarshal(body, &apiError) == nil && len(apiError.Detail) > 0 {
		var detail string
		if json.Unmarshal(apiError.Detail, &detail) == nil {
			message = detail
		} else {
			message = string(apiError.Detail)
		}
	}
	if message == "" {
		message = http.StatusText(status)
	}
	return fmt.Errorf("API returned status %d: %s", status, message)
}
