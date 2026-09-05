// Package api implementa o contrato HTTP entre o cliente Go e o backend.
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

// Client envia imagens para o backend sem executar modelos localmente.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Identification representa tanto o contrato atual quanto os campos previstos
// para a evolução taxonômica da API.
type Identification struct {
	Identification             string      `json:"identification,omitempty"`
	Confidence                 float64     `json:"confidence"`
	Order                      string      `json:"order,omitempty"`
	DetectedClass              string      `json:"detected_class,omitempty"`
	Status                     string      `json:"status,omitempty"`
	Prediction                 string      `json:"prediction,omitempty"`
	ReachedRank                string      `json:"reached_rank,omitempty"`
	Taxonomy                   *Taxonomy   `json:"taxonomy,omitempty"`
	Candidates                 []Candidate `json:"candidates,omitempty"`
	RequiresExpertConfirmation bool        `json:"requires_expert_confirmation,omitempty"`
	ModelVersion               string      `json:"model_version,omitempty"`
}

// Taxonomy descreve a classificação retornada por uma versão futura do backend.
type Taxonomy struct {
	Kingdom string `json:"kingdom,omitempty"`
	Phylum  string `json:"phylum,omitempty"`
	Class   string `json:"class,omitempty"`
	Order   string `json:"order,omitempty"`
	Family  string `json:"family,omitempty"`
	Genus   string `json:"genus,omitempty"`
	Species string `json:"species,omitempty"`
}

// Candidate é uma identificação alternativa ordenada por confiança.
type Candidate struct {
	ScientificName string  `json:"scientific_name"`
	Confidence     float64 `json:"confidence"`
}

type identifyRequest struct {
	Image string `json:"image"`
}

type errorResponse struct {
	Detail json.RawMessage `json:"detail"`
}

// NewClient valida a URL e cria um cliente com timeout explícito.
func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("URL da API inválida: %q", baseURL)
	}
	if timeout <= 0 {
		return nil, errors.New("o timeout deve ser maior que zero")
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// Identify lê uma imagem local e a envia usando o contrato Base64 atual.
func (c *Client) Identify(ctx context.Context, imagePath string) (Identification, error) {
	image, err := readImage(imagePath)
	if err != nil {
		return Identification{}, err
	}

	body, err := json.Marshal(identifyRequest{Image: base64.StdEncoding.EncodeToString(image)})
	if err != nil {
		return Identification{}, fmt.Errorf("não foi possível preparar a requisição: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/identify",
		bytes.NewReader(body),
	)
	if err != nil {
		return Identification{}, fmt.Errorf("não foi possível criar a requisição: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ColeopteraCLI/0.1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Identification{}, fmt.Errorf("não foi possível acessar a API: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return Identification{}, fmt.Errorf("não foi possível ler a resposta da API: %w", err)
	}
	if len(responseBody) > maxResponseSize {
		return Identification{}, errors.New("a resposta da API excedeu o limite de 1 MB")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Identification{}, parseAPIError(response.StatusCode, responseBody)
	}

	var result Identification
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return Identification{}, fmt.Errorf("a API retornou JSON inválido: %w", err)
	}
	return result, nil
}

func readImage(imagePath string) ([]byte, error) {
	cleanPath := filepath.Clean(imagePath)
	extension := strings.ToLower(filepath.Ext(cleanPath))
	if _, ok := allowedExtensions[extension]; !ok {
		return nil, fmt.Errorf("formato %q não suportado", extension)
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("não foi possível abrir a imagem: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("não foi possível inspecionar a imagem: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("o caminho selecionado não é um arquivo regular")
	}
	if info.Size() == 0 {
		return nil, errors.New("a imagem está vazia")
	}
	if info.Size() > MaxImageSize {
		return nil, errors.New("a imagem excede o limite de 10 MB")
	}

	content, err := io.ReadAll(io.LimitReader(file, MaxImageSize+1))
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler a imagem: %w", err)
	}
	if len(content) > MaxImageSize {
		return nil, errors.New("a imagem excede o limite de 10 MB")
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
	return fmt.Errorf("API respondeu com status %d: %s", status, message)
}
