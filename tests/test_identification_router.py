"""Endpoint tests that replace BioCLIP and GBIF with deterministic fakes."""

import base64
import io
import unittest
from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient
from PIL import Image

from app.main import app
from app.schemas.identification import IdentificationResponse, Taxonomy


def encoded_image() -> str:
    buffer = io.BytesIO()
    Image.new("RGB", (16, 16), "green").save(buffer, format="PNG")
    return base64.b64encode(buffer.getvalue()).decode("ascii")


class IdentificationRouterTests(unittest.TestCase):
    def setUp(self) -> None:
        self.client = TestClient(app)

    def test_identify_preserves_base64_contract(self) -> None:
        response_value = IdentificationResponse(
            status="suggestion",
            identification="Scarabaeidae",
            prediction="Scarabaeidae",
            confidence=0.8,
            order="Coleoptera",
            reached_rank="family",
            taxonomy=Taxonomy(
                kingdom="Animalia",
                phylum="Arthropoda",
                class_="Insecta",
                order="Coleoptera",
                family="Scarabaeidae",
            ),
            requires_expert_confirmation=True,
            model_version="fake-bioclip",
            score_scope="Fake scores normalized among represented Coleoptera taxa.",
        )

        with (
            patch("app.routers.identification.identifier.identify", return_value=response_value),
            patch(
                "app.routers.identification.image_validator.validate_file_type",
                return_value=True,
            ),
            patch(
                "app.routers.identification.gbif_client.match_species",
                new=AsyncMock(return_value=None),
            ),
        ):
            response = self.client.post("/api/identify", json={"image": encoded_image()})

        self.assertEqual(response.status_code, 200)
        body = response.json()
        self.assertEqual(body["status"], "suggestion")
        self.assertEqual(body["reached_rank"], "family")
        self.assertFalse(body["confidence_calibrated"])

    def test_rejects_non_strict_base64(self) -> None:
        response = self.client.post("/api/identify", json={"image": "%%%invalid%%%"})
        self.assertEqual(response.status_code, 400)

    def test_requires_exactly_one_source(self) -> None:
        response = self.client.post("/api/identify", json={})
        self.assertEqual(response.status_code, 422)


if __name__ == "__main__":
    unittest.main()
