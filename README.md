# 🪲 Coleoptera Identifier

![Coleoptera Identifier logo](./assets/coleoptera_logo.png)

Protótipo de identificação visual taxonômica de besouros. A API usa o
[BioCLIP](https://github.com/Imageomics/BioCLIP) no servidor e devolve a
hipótese visual mais específica permitida por limiares heurísticos
configurados: ordem, família, gênero ou espécie.

O resultado **não é uma determinação taxonômica**. Algumas espécies exigem
outras vistas, dados de coleta, escala, microscopia ou avaliação de um
especialista.

## Arquitetura

```text
Go CLI/TUI
    │ imagem em Base64
    ▼
FastAPI ── validação da imagem
    │
    ▼
BioCLIP ── classificação geral por ordem
    │
    ├── não Coleoptera → sugestão fora da ordem
    │
    └── Coleoptera → candidatos taxonômicos + nível sugerido
                              │
                              └── GBIF: normalização opcional do nome
```

O modelo é carregado apenas na primeira identificação. O endpoint de saúde e
a inicialização da API não baixam pesos nem consultam serviços externos.

## Requisitos

- Python 3.10 ou mais recente, em uma versão suportada pelo PyTorch instalado.
- `libmagic` instalado no sistema.
- Memória e armazenamento suficientes para os artefatos do BioCLIP.

Em Debian/Ubuntu:

```bash
sudo apt-get install libmagic1
```

## Instalação

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install -r requirements.txt
```

O BioCLIP original, baseado em ViT-B/16, é o padrão por ser menor que o
BioCLIP 2. Para baixar e preparar os artefatos antes de iniciar a API:

```bash
python -m app.services.preload
```

## Executar

```bash
uvicorn app.main:app --host 127.0.0.1 --port 8000
```

- Documentação OpenAPI: <http://localhost:8000/docs>
- Saúde: <http://localhost:8000/health>
- Cliente Go: [cli/README.md](./cli/README.md)

## API

```bash
IMAGE_BASE64="$(base64 -w 0 foto.jpg)"
curl -X POST http://localhost:8000/api/identify \
  -H 'Content-Type: application/json' \
  -d "{\"image\":\"$IMAGE_BASE64\"}"
```

Resposta ilustrativa:

```json
{
  "status": "suggestion",
  "identification": "Scarabaeidae",
  "prediction": "Scarabaeidae",
  "confidence": 0.78,
  "confidence_calibrated": false,
  "confidence_interpretation": "Model score used for conservative rank selection; not a scientific certainty.",
  "order": "Coleoptera",
  "reached_rank": "family",
  "taxonomy": {
    "kingdom": "Animalia",
    "phylum": "Arthropoda",
    "class": "Insecta",
    "order": "Coleoptera",
    "family": "Scarabaeidae",
    "genus": null,
    "species": null
  },
  "candidates": [],
  "requires_expert_confirmation": true,
  "model_version": "bioclip-v1-tol10m",
  "score_scope": "Score normalized among represented Coleoptera taxa at the family rank.",
  "candidate_scope": "Coleoptera taxa represented by the configured BioCLIP Tree of Life embedding set.",
  "disclaimer": "Automated visual hypothesis, not a taxonomic determination."
}
```

`confidence` é um escore interno não calibrado. Ele não representa a
probabilidade empírica de a hipótese estar correta. Os escores dos candidatos
de espécie são condicionais aos táxons de Coleoptera representados pelo modelo.

## Configuração

| Variável | Padrão | Função |
|---|---:|---|
| `BIOCLIP_MODEL` | `hf-hub:imageomics/bioclip` | Modelo OpenCLIP utilizado |
| `BIOCLIP_DEVICE` | `cpu` | Dispositivo PyTorch, como `cpu` ou `cuda` |
| `BIOCLIP_TOP_K` | `5` | Número de candidatos de espécie |
| `BIOCLIP_ORDER_THRESHOLD` | `0.35` | Escore mínimo para aceitar a ordem sugerida |
| `BIOCLIP_FAMILY_THRESHOLD` | `0.70` | Escore mínimo para sugerir família |
| `BIOCLIP_GENUS_THRESHOLD` | `0.65` | Escore mínimo para sugerir gênero |
| `BIOCLIP_SPECIES_THRESHOLD` | `0.60` | Escore mínimo para sugerir espécie |
| `BIOCLIP_SPECIES_MARGIN` | `0.15` | Margem mínima entre as duas primeiras espécies |
| `ENABLE_REMOTE_IMAGE_URLS` | `false` | Habilita download de imagens por URL |
| `ALLOWED_IMAGE_HOSTS` | vazio | Lista de domínios permitidos, separada por vírgula |

Os limiares são heurísticos até que sejam calibrados em um conjunto de
avaliação representativo. Alterá-los não transforma o escore em probabilidade.

URLs remotas ficam desabilitadas por padrão. Quando habilitadas, a API bloqueia
destinos locais/privados e revalida redirecionamentos. Para reduzir a superfície
de ataque, prefira o envio da própria imagem.

## Testes

```bash
python -m pytest
cd cli && go test ./...
```

Os testes do backend substituem o runtime BioCLIP por resultados determinísticos;
eles não baixam pesos nem alegam medir acurácia científica. A avaliação científica
deve usar um conjunto rotulado e separado, incluindo espécies fora do catálogo e
imagens que não contenham Coleoptera.

## Limitações científicas

- A cobertura é limitada aos táxons presentes no Tree of Life usado pelo modelo.
- Escores de redes neurais podem ser altos mesmo para imagens fora da distribuição.
- A agregação por ordem pode refletir a quantidade desigual de táxons representados.
- O modelo sempre compara a imagem com seu catálogo; ele não possui uma classe aberta
  confiável para “organismo desconhecido” ou “imagem sem organismo”.
- Espécies crípticas podem ser visualmente indistinguíveis em uma única fotografia.
- A consulta ao GBIF normaliza nomes; ela não confirma a identificação da imagem.
- Localidade, data, estágio de vida, sexo e escala ainda não entram no ranqueamento.
- As revisões dos artefatos hospedados no Hugging Face ainda não são fixadas por hash.
- Todo resultado em espécie deve ser tratado como candidato para confirmação.

## Créditos do modelo

O projeto utiliza `pybioclip` e o BioCLIP original. Consulte as publicações e
licenças mantidas pelo projeto [Imageomics/BioCLIP](https://github.com/Imageomics/BioCLIP)
antes de redistribuir artefatos do modelo.
