# 🪲 coleoptera identifier experiment

![logo](./assets/coleoptera_logo.png)

Identificador de invertebrados terrestres da ordem Coleoptera (besouros) utilizando PyTorch e o modelo pré-treinado MobileNetV2.

## setup

```bash
python3 -m venv .venv # crie um ambiente virtual
pip3 install -m ./requirements.txt # instale as dependências
```

## API Contract

se for um coleoptéro:

```json
{
  "identification": "",
  "confidence": "",
  "order": "Coleoptera"
}
```

se não for:

```json
{
  "identification": "",
  "confidence": "",
  "detected_class": ""
}
```

## Routes 