"""HTTP endpoint for server-side taxonomic image suggestions."""

import base64
import binascii
from typing import Annotated

from fastapi import APIRouter, HTTPException
from fastapi.concurrency import run_in_threadpool
from pydantic import BaseModel, Field, HttpUrl, model_validator

from app.schemas.identification import IdentificationResponse
from app.services.bioclip import (
    IdentificationError,
    InvalidImageError,
    ModelUnavailableError,
    identifier,
)
from app.services.gbif import gbif_client
from app.utils import image_validator

MAX_IMAGE_SIZE = 10 * 1024 * 1024
MAX_BASE64_LENGTH = 4 * ((MAX_IMAGE_SIZE + 2) // 3)


class ImagePayload(BaseModel):
    """Exactly one supported image source."""

    image: Annotated[str | None, Field(default=None, max_length=MAX_BASE64_LENGTH + 128)]
    url: HttpUrl | None = None

    @model_validator(mode="after")
    def check_exclusive_fields(self) -> "ImagePayload":
        if bool(self.image) == bool(self.url):
            raise ValueError('Provide exactly one of "image" (Base64) or "url".')
        return self


router = APIRouter()


@router.post(
    "/identify",
    response_model=IdentificationResponse,
    summary="Suggest a taxonomic identification for an image",
)
async def identify_insect_endpoint(payload: ImagePayload) -> IdentificationResponse:
    """Return the most specific defensible rank and species candidates."""

    if payload.url:
        image_bytes = await image_validator.download_and_validate_image(str(payload.url))
    else:
        image_bytes = _decode_base64_image(payload.image or "")

    try:
        response = await run_in_threadpool(identifier.identify, image_bytes)
    except InvalidImageError as exc:
        raise HTTPException(status_code=400, detail="The submitted image is invalid.") from exc
    except ModelUnavailableError as exc:
        raise HTTPException(
            status_code=503,
            detail=(
                "The taxonomic model is unavailable. "
                "Check the BioCLIP artifacts on the server."
            ),
        ) from exc
    except IdentificationError as exc:
        raise HTTPException(
            status_code=500,
            detail="The identification could not be completed.",
        ) from exc

    await _enrich_species_with_gbif(response)
    return response


def _decode_base64_image(encoded: str) -> bytes:
    """Strictly decode Base64 while enforcing the same limit as URL downloads."""

    image_data = encoded.split(",", 1)[1] if "," in encoded else encoded
    if len(image_data) > MAX_BASE64_LENGTH:
        raise HTTPException(status_code=413, detail="The image exceeds the 10 MB limit.")

    try:
        image_bytes = base64.b64decode(image_data, validate=True)
    except (binascii.Error, ValueError, TypeError) as exc:
        raise HTTPException(status_code=400, detail="The image does not contain valid Base64.") from exc

    if len(image_bytes) > MAX_IMAGE_SIZE:
        raise HTTPException(status_code=413, detail="The image exceeds the 10 MB limit.")
    if not image_validator.validate_file_signature(image_bytes):
        raise HTTPException(status_code=400, detail="The image format is not supported.")
    if not image_validator.validate_file_type(image_bytes):
        raise HTTPException(status_code=400, detail="The submitted content is not a valid image.")
    return image_bytes


async def _enrich_species_with_gbif(response: IdentificationResponse) -> None:
    """Attach accepted-name metadata without changing the visual prediction."""

    if response.reached_rank != "species" or not response.candidates:
        return

    top = response.candidates[0]
    match = await gbif_client.match_species(top.scientific_name)
    if match is None:
        return

    top.gbif_taxon_key = match.taxon_key
    top.accepted_scientific_name = match.scientific_name
    top.taxonomic_status = match.status
