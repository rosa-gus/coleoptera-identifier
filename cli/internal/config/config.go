// Package config provides minimal client configuration without creating local files.
package config

import "os"

const defaultAPIURL = "http://localhost:8000"

// APIURL returns the environment value or the default local address.
func APIURL() string {
	if value := os.Getenv("COLEOPTERA_API_URL"); value != "" {
		return value
	}
	return defaultAPIURL
}
