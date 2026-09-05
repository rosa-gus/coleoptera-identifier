# Coleoptera CLI

Cliente de terminal em Go para a API Coleoptera Identifier. O modelo de visão
continua no backend: este binário apenas seleciona a imagem, envia a requisição
e apresenta o resultado.

Os percentuais exibidos são escores não calibrados do modelo, não probabilidades
de acerto. A interface apresenta o nível taxonômico como sugestão e mantém o
aviso de confirmação especializada retornado pela API.

## Requisitos

- Go 1.25 ou mais recente para compilar.
- Backend disponível em `http://localhost:8000` por padrão.

## Executar

```bash
go run . ui
```

Executar sem subcomando também abre a interface:

```bash
go run .
```

Identificação direta, sem TUI:

```bash
go run . identify ../minha-imagem.jpg
go run . identify ../minha-imagem.jpg --output json
```

Para apontar a outro backend:

```bash
export COLEOPTERA_API_URL=https://api.example.com
go run . ui
```

A mesma configuração pode ser passada com `--api-url`.

## Compilar

```bash
go build -o coleoptera .
./coleoptera ui
```

## Atalhos da interface

- `↑`/`↓` ou `j`/`k`: navegar.
- `enter`: abrir diretório ou selecionar imagem.
- `←` ou `h`: voltar ao diretório anterior.
- `r`: analisar outra imagem após um resultado.
- `q`: sair.

## Contrato atual

O cliente envia `POST /api/identify` com a imagem em Base64, preservando o
contrato atual do backend. A comunicação está isolada em `internal/api` para
que uma futura migração para upload multipart não afete a TUI.
