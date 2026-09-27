package curation

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchObservationAndTaxonomicLineage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/observations/339647945":
			fmt.Fprint(writer, `{"results":[{"id":339647945,"observed_on":"2026-02-07","place_guess":"Cafayate, Salta, Argentina","taxon":{"id":42,"name":"Neocalvia dentatofasciata","rank":"species"},"photos":[{"id":617908244,"url":"https://static.inaturalist.org/photos/617908244/square.jpg","attribution":"(c) Rosario Douglas, some rights reserved (CC BY)","license_code":"cc-by"}]}]}`)
		case "/taxa/42":
			fmt.Fprint(writer, `{"results":[{"name":"Neocalvia dentatofasciata","rank":"species","ancestors":[{"name":"Coleoptera","rank":"order"},{"name":"Coccinellidae","rank":"family"},{"name":"Neocalvia","rank":"genus"}]}]}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := &Client{HTTP: server.Client(), BaseURL: server.URL}
	observation, err := client.Fetch(context.Background(), "339647945")
	if err != nil {
		t.Fatal(err)
	}
	if observation.Order != "Coleoptera" || observation.Family != "Coccinellidae" || observation.Species != "Neocalvia dentatofasciata" {
		t.Fatalf("unexpected taxonomy: %+v", observation)
	}
	if observation.Country != "Argentina" || len(observation.Photos) != 1 {
		t.Fatalf("unexpected observation: %+v", observation)
	}
	if PhotoAuthor(observation.Photos[0]) != "Rosario Douglas" || LicenseURL(observation.Photos[0].LicenseCode) != "https://creativecommons.org/licenses/by/4.0/" {
		t.Fatalf("unexpected photo credits: %+v", observation.Photos[0])
	}
}

func TestFetchRejectsInvalidObservationID(t *testing.T) {
	if _, err := NewClient().Fetch(context.Background(), "../42"); err == nil {
		t.Fatal("invalid ID was accepted")
	}
}
