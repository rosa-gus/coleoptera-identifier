import torch
import io
import json
import os

from PIL import Image
from torchvision import models, transforms
from app.services.gbif import get_coleoptera_taxa
from app.utils.coleoptera_validator import is_coleoptera

# initialize filter
coleoptera_taxa = get_coleoptera_taxa()

#load pre-trained model
model = models.mobilenet_v2(weights=models.MobileNet_V2_Weights.DEFAULT)
model.eval()

class_idx = None

try:
    json_path = os.path.join(os.path.dirname(__file__), "imagenet_class_index.json")
    with open(json_path) as f:
        class_idx = json.load(f)
    labels = {int(k): v[1] for k, v in class_idx.items()}
except FileNotFoundError:
    print("WARNING: 'image_class_index.json' não encontrado. A identificação irá retornar somente o índice da classe.")
    labels = {}

preprocess = transforms.Compose([
    transforms.Resize(256),
    transforms.CenterCrop(224),
    transforms.ToTensor(),
    transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
])

async def identify_image(image_bytes: bytes):
    """
    Identifica o conteúdo de uma imagem usando o modelo MobileNetV2.
    """
    try:
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
        input_tensor = preprocess(image)
        input_batch = input_tensor.unsqueeze(0)
        
        with torch.no_grad():
            output = model(input_batch)
        
        probabilities = torch.nn.functional.softmax(output[0], dim=0)
        
        top_prob, top_catid = torch.topk(probabilities, 1)
        prediction_idx = top_catid[0].item()
        
        if not labels:
            return {"error": "Arquivos de rótulos do ImageNet não encontrado."}
        
        predicted_class_name = labels.get(prediction_idx, "unknown")
        
        if is_coleoptera(predicted_class_name, coleoptera_taxa):
            return {
                "identification": predicted_class_name,
                "confidence": float(top_prob.item()),
                "order": "Coleoptera"
            }
        else:
            return {
                "identification": "não é um invertebrado da ordem Coleoptera",
                "confidence": float(top_prob.item()),
                "detected_class": predicted_class_name
            }
    except Exception as e:
        return {"error": f"Falha ao processar a imagem: {str(e)}"}