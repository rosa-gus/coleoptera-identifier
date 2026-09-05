# Coleoptera CLI

A Go terminal client for the Coleoptera Identifier API. The vision model remains
on the backend: this binary only selects an image, sends the request, and displays
the result.

Displayed percentages are uncalibrated model scores, not probabilities of a
correct identification. The interface presents the taxonomic rank as a suggestion
and displays the scientific disclaimer returned by the API.

## Requirements

- Go 1.25 or newer to compile the client.
- A backend available at `http://localhost:8000` by default.

## Running

```bash
go run . ui
```

Running without a subcommand also opens the interface:

```bash
go run .
```

Run a direct identification without the TUI:

```bash
go run . identify ../beetle.jpg
go run . identify ../beetle.jpg --output json
```

To use a different backend:

```bash
export COLEOPTERA_API_URL=https://api.example.com
go run . ui
```

The same value can be provided with `--api-url`.

The interface defaults to US English. Select Português do Brasil with a flag
or environment variable:

```bash
go run . --lang pt-BR
COLEOPTERA_LANG=pt-BR go run . ui
```

The `--lang` flag has precedence over `COLEOPTERA_LANG`. If neither is set, the
interface remains in English regardless of the system locale. Error messages
remain in English.

## Building

```bash
go build -o coleoptera .
./coleoptera ui
```

## Interface shortcuts

- `↑`/`↓` or `j`/`k`: navigate.
- `enter`: open a directory or select an image.
- `←` or `h`: return to the parent directory.
- `r`: analyze another image after displaying a result.
- `q`: quit.

## Current API contract

The client sends a Base64 image to `POST /api/identify`, preserving the current
backend contract. Communication is isolated in `internal/api`, allowing a future
migration to multipart uploads without affecting the TUI.
