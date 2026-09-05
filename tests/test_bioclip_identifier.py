"""Unit tests for confidence policy without loading model artifacts."""

import io
import unittest

from PIL import Image

from app.services.bioclip import (
    BioCLIPIdentifier,
    IdentifierSettings,
    InvalidImageError,
    RawCandidate,
    RawPrediction,
)


def image_bytes() -> bytes:
    buffer = io.BytesIO()
    Image.new("RGB", (16, 16), "green").save(buffer, format="PNG")
    return buffer.getvalue()


def candidate(
    species: str = "Phanaeus splendidulus",
    confidence: float = 0.8,
) -> RawCandidate:
    return RawCandidate(
        taxonomy={
            "kingdom": "Animalia",
            "phylum": "Arthropoda",
            "class": "Insecta",
            "order": "Coleoptera",
            "family": "Scarabaeidae",
            "genus": "Phanaeus",
            "species": species,
        },
        common_name=None,
        confidence=confidence,
    )


class FakeRuntime:
    model_version = "fake-bioclip"

    def __init__(self, prediction: RawPrediction) -> None:
        self.prediction = prediction

    def predict(self, _image: Image.Image, _top_k: int) -> RawPrediction:
        return self.prediction


class BioCLIPIdentifierTests(unittest.TestCase):
    def setUp(self) -> None:
        self.settings = IdentifierSettings()

    def identify(self, prediction: RawPrediction):
        service = BioCLIPIdentifier(
            settings=self.settings,
            runtime=FakeRuntime(prediction),
        )
        return service.identify(image_bytes())

    def test_rejects_non_coleoptera_order(self) -> None:
        response = self.identify(RawPrediction(detected_order="Lepidoptera", order_confidence=0.88))

        self.assertEqual(response.status, "not_coleoptera")
        self.assertEqual(
            response.identification,
            "the model suggests it does not belong to the order Coleoptera",
        )
        self.assertEqual(response.reached_rank, "order")
        self.assertTrue(response.requires_expert_confirmation)
        self.assertFalse(response.confidence_calibrated)

    def test_non_coleoptera_is_uncertain_when_order_score_is_low(self) -> None:
        response = self.identify(RawPrediction(detected_order="Lepidoptera", order_confidence=0.2))

        self.assertEqual(response.status, "uncertain")
        self.assertEqual(response.identification, "inconclusive visual classification")
        self.assertEqual(response.detected_order, "Lepidoptera")
        self.assertTrue(response.requires_expert_confirmation)

    def test_returns_uncertain_when_order_score_is_too_low(self) -> None:
        response = self.identify(
            RawPrediction(
                detected_order="Coleoptera",
                order_confidence=0.2,
                candidates=(candidate(),),
            )
        )

        self.assertEqual(response.status, "uncertain")
        self.assertEqual(
            response.identification,
            "possible organism of the order Coleoptera",
        )
        self.assertEqual(response.reached_rank, "order")
        self.assertEqual(response.candidates, [])

    def test_allows_species_only_with_score_and_margin(self) -> None:
        response = self.identify(
            RawPrediction(
                detected_order="Coleoptera",
                order_confidence=0.9,
                candidates=(candidate(), candidate("Phanaeus kirbyi", 0.1)),
                family_confidence=0.95,
                genus_confidence=0.91,
                species_margin=0.7,
            )
        )

        self.assertEqual(response.status, "suggestion")
        self.assertEqual(response.reached_rank, "species")
        self.assertEqual(response.prediction, "Phanaeus splendidulus")
        self.assertTrue(response.requires_expert_confirmation)
        self.assertIn("not a taxonomic determination", response.disclaimer)

    def test_falls_back_to_genus_without_species_evidence(self) -> None:
        response = self.identify(
            RawPrediction(
                detected_order="Coleoptera",
                order_confidence=0.9,
                candidates=(candidate(confidence=0.4), candidate("Phanaeus kirbyi", 0.35)),
                family_confidence=0.93,
                genus_confidence=0.78,
                species_margin=0.05,
            )
        )

        self.assertEqual(response.reached_rank, "genus")
        self.assertEqual(response.prediction, "Phanaeus")
        self.assertIsNone(response.taxonomy.species)
        self.assertEqual(len(response.candidates), 2)

    def test_never_invents_a_species_name_for_incomplete_taxonomy(self) -> None:
        incomplete = candidate()
        incomplete.taxonomy.pop("species")
        response = self.identify(
            RawPrediction(
                detected_order="Coleoptera",
                order_confidence=0.9,
                candidates=(incomplete,),
                family_confidence=0.95,
                genus_confidence=0.9,
                species_margin=0.8,
            )
        )

        self.assertEqual(response.reached_rank, "genus")
        self.assertEqual(response.candidates, [])

    def test_rejects_invalid_image_bytes(self) -> None:
        service = BioCLIPIdentifier(
            settings=self.settings,
            runtime=FakeRuntime(RawPrediction("Coleoptera", 1.0)),
        )
        with self.assertRaises(InvalidImageError):
            service.identify(b"not an image")


if __name__ == "__main__":
    unittest.main()
