from fastapi import APIRouter, HTTPException
from pydantic import HttpUrl, BaseModel, model_validator
import base64
import binascii
from typing import Optional

from app.utils import image_validator
from app.services.mobilenet import identify_image

class ImagePayload(BaseModel):
    image: Optional[str] = None
    url: Optional[HttpUrl] = None
    
    @model_validator(mode='before')
    @classmethod
    def check_exclusive_fields(cls, data):
        if ('image' in data and data['image']) and ('url' in data and data['url']):
            raise ValueError('Forneça apenas "image" (Base64) ou "url", não ambos.')
        if not ('image' in data and data['image']) and not ('url' in data and data['url']):
            raise ValueError('Você deve fornecer "image" (Base64) ou "url".')
        return data

router = APIRouter()

@router.post("/identify")
async def identify_insect_endpoint(payload: ImagePayload):
    """
    Recebe uma imagem (via URL ou Base64) e a envia para o serviço de identificação.
    Retorna o invertebrado identificado, se o inseto for da ordem Coleoptera.
    """
    image_bytes: bytes
    
    if payload.url:
        image_bytes = await image_validator.download_and_validate_image(str(payload.url))
    
    elif payload.image:
        image_data = payload.image
        if "," in image_data:
            _, image_data = image_data.split(",", 1)
    
        try:
            image_bytes = base64.b64decode(image_data)
            if not image_validator.validate_file_signature(image_bytes) or not image_validator.validate_file_type(image_bytes):
                raise HTTPException(status_code=400, detail="A imagem fornecida em Base64 não é um formato válido ou suportado.")
        except (binascii.Error, TypeError):
            raise HTTPException(status_code=400, detail="A string da imagem não é um Base64 válido.")
    
    result = await identify_image(image_bytes)
    
    if "error" in result:
        raise HTTPException(status_code=500, detail=result["error"])
    
    return result
