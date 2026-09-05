package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIdentifySendsCurrentBackendContract(t *testing.T) {
	t.Parallel()

	image := []byte("fake png content")
	imagePath := filepath.Join(t.TempDir(), "beetle.png")
	if err := os.WriteFile(imagePath, image, 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/identify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var payload identifyRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("invalid payload: %v", err)
		}
		decoded, err := base64.StdEncoding.DecodeString(payload.Image)
		if err != nil || string(decoded) != string(image) {
			t.Errorf("Base64 image differs from the original")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"identification":"ladybug","confidence":0.91,"order":"Coleoptera"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Identify(context.Background(), imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Identification != "ladybug" || result.Order != "Coleoptera" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestIdentifyReturnsBackendError(t *testing.T) {
	t.Parallel()

	imagePath := filepath.Join(t.TempDir(), "beetle.jpg")
	if err := os.WriteFile(imagePath, []byte("fake jpg content"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"invalid image"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Identify(context.Background(), imagePath)
	if err == nil || !strings.Contains(err.Error(), "invalid image") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewClientRejectsInvalidURL(t *testing.T) {
	t.Parallel()
	if _, err := NewClient("file:///tmp/api", time.Second); err == nil {
		t.Fatal("expected an error for a non-HTTP(S) URL")
	}
}

func TestIdentifyRejectsUnsupportedExtension(t *testing.T) {
	t.Parallel()
	client, err := NewClient("http://localhost:8000", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Identify(context.Background(), "beetle.txt"); err == nil {
		t.Fatal("expected an error for an unsupported extension")
	}
}
