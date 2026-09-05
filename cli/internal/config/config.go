// Package config reúne a configuração mínima do cliente, sem criar arquivos no
// dispositivo do usuário.
package config

import "os"

const defaultAPIURL = "http://localhost:8000"

// APIURL retorna a URL configurada pelo ambiente ou o endereço local padrão.
func APIURL() string {
	if value := os.Getenv("COLEOPTERA_API_URL"); value != "" {
		return value
	}
	return defaultAPIURL
}
