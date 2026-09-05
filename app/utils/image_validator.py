from fastapi import HTTPException
import httpx
import magic
from urllib.parse import urlparse

def validate_url(url: str) -> bool:
    """VValida se a URL é segura e permitida."""
    try:
        parsed = urlparse(url)
        if parsed.scheme not in ['http', 'https']:
            return False
        
        # Restringir domínios
        # ALLOWED_DOMAINS = {'i.imgur.com', 'images.unsplash.com'}
        # if parsed.netloc not in ALLOWED_DOMAINS:
        #     return False
        
        return True
    except Exception:
        return False
    
def validate_file_signature(content: bytes) -> bool:
    """Verifica a assinatura do arquivo (magic numbers) para tipos de imagem comuns."""
    SIGNATURES = {
        b'\xff\xd8\xff': 'image/jpeg',
        b'\x89PNG\r\n\x1a\n': 'image/png',
        b'GIF8': 'image/gif',
        b'BM': 'image/bmp',
        b'RIFF': 'image/webp'
     }
    for signature in SIGNATURES:
        if content.startswith(signature):
            return True
    return False

def validate_file_type(content: bytes) -> bool:
    """Verifica o MIME type do conteúdo usando python-magic."""
    ALLOWED_MIME_TYPES = {'image/jpeg', 'image/png', 'image/gif', 'image/bmp', 'image/webp'}
    try:
        file_type = magic.from_buffer(content, mime=True)
        return file_type in ALLOWED_MIME_TYPES
    except Exception:
        return False
    
async def download_and_validate_image(url: str) -> bytes:
    """Faz o download da imagem e aplica múltiplas validações de segurança."""
    if not validate_url(url):
        raise HTTPException(status_code=400, detail="URL fornecida é inválida ou não permitida.")

    MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB
    TIMEOUT = 30
    headers = {'User-Agent': 'ColeopteraIdentificator/0.0.1'}

    async with httpx.AsyncClient(timeout=TIMEOUT, follow_redirects=True) as client:
        try:
            async with client.stream('GET', url, headers=headers) as response:
                response.raise_for_status()

                content = b''
                async for chunk in response.aiter_bytes():
                    content += chunk
                    if len(content) > MAX_FILE_SIZE:
                        raise HTTPException(status_code=413, detail="Arquivo de imagem excede o tamanho máximo de 10MB.")
        except httpx.TimeoutException:
            raise HTTPException(status_code=408, detail="Timeout ao baixar a imagem.")
        except httpx.RequestError as e:
            raise HTTPException(status_code=400, detail=f"Erro ao requisitar a imagem da URL: {e}")

    # Validações pós-download
    if not validate_file_signature(content) or not validate_file_type(content):
        raise HTTPException(status_code=400, detail="O arquivo baixado não é um formato de imagem válido ou suportado.")
    
    return content