"""Lazy, server-side taxonomic identification powered by BioCLIP."""

from __future__ import annotations

import io
import logging
import os
import threading
from collections.abc import Sequence
from dataclasses import dataclass
from typing import Any, Protocol

from PIL import Image, ImageOps, UnidentifiedImageError

from app.schemas.identification import Candidate, IdentificationResponse, Taxonomy

logger = logging.getLogger(__name__)

COLEOPTERA = "Coleoptera"
DEFAULT_MODEL = "hf-hub:imageomics/bioclip"
MAX_IMAGE_PIXELS = 25_000_000
MIN_IMAGE_DIMENSION = 32
MAX_ASPECT_RATIO = 20.0


class ModelUnavailableError(RuntimeError):
    """Raised when the BioCLIP runtime cannot be loaded."""


class InvalidImageError(ValueError):
    """Raised when submitted bytes cannot be decoded as a safe image."""


class IdentificationError(RuntimeError):
    """Raised when inference fails after the model has loaded."""


@dataclass(frozen=True)
class IdentifierSettings:
    """Runtime and confidence settings for hierarchical identification."""

    model_name: str = DEFAULT_MODEL
    device: str = "cpu"
    top_k: int = 5
    order_threshold: float = 0.35
    family_threshold: float = 0.70
    genus_threshold: float = 0.65
    species_threshold: float = 0.60
    species_margin: float = 0.15

    @classmethod
    def from_environment(cls) -> IdentifierSettings:
        """Build settings from environment variables with safe defaults."""

        return cls(
            model_name=os.getenv("BIOCLIP_MODEL", DEFAULT_MODEL),
            device=os.getenv("BIOCLIP_DEVICE", "cpu"),
            top_k=_read_int("BIOCLIP_TOP_K", 5, minimum=1, maximum=20),
            order_threshold=_read_probability("BIOCLIP_ORDER_THRESHOLD", 0.35),
            family_threshold=_read_probability("BIOCLIP_FAMILY_THRESHOLD", 0.70),
            genus_threshold=_read_probability("BIOCLIP_GENUS_THRESHOLD", 0.65),
            species_threshold=_read_probability("BIOCLIP_SPECIES_THRESHOLD", 0.60),
            species_margin=_read_probability("BIOCLIP_SPECIES_MARGIN", 0.15),
        )


@dataclass(frozen=True)
class RawCandidate:
    """Internal candidate before confidence policy and GBIF enrichment."""

    taxonomy: dict[str, str]
    common_name: str | None
    confidence: float


@dataclass(frozen=True)
class RawPrediction:
    """Probabilities needed to select the most reliable taxonomic rank."""

    detected_order: str
    order_confidence: float
    candidates: tuple[RawCandidate, ...] = ()
    family_confidence: float = 0.0
    genus_confidence: float = 0.0
    species_margin: float = 0.0


class PredictionRuntime(Protocol):
    """Minimal runtime interface used by the service and its tests."""

    model_version: str

    def predict(self, image: Image.Image, top_k: int) -> RawPrediction:
        """Return hierarchical probabilities for one RGB image."""


class BioCLIPIdentifier:
    """Thread-safe façade that lazily loads and reuses the BioCLIP model."""

    def __init__(
        self,
        settings: IdentifierSettings | None = None,
        runtime: PredictionRuntime | None = None,
    ) -> None:
        self.settings = settings or IdentifierSettings.from_environment()
        self._runtime = runtime
        self._load_lock = threading.Lock()
        self._inference_lock = threading.Lock()

    @property
    def is_loaded(self) -> bool:
        """Report whether model artifacts are resident in memory."""

        return self._runtime is not None

    def load(self) -> None:
        """Load model artifacts explicitly for image builds or warm-up jobs."""

        self._get_runtime()

    def identify(self, image_bytes: bytes) -> IdentificationResponse:
        """Decode an image, execute inference, and apply confidence policy."""

        image = _decode_image(image_bytes)
        runtime = self._get_runtime()

        try:
            with self._inference_lock:
                raw = runtime.predict(image, self.settings.top_k)
        except Exception as exc:
            logger.exception("BioCLIP inference failed")
            raise IdentificationError("BioCLIP inference failed") from exc

        return self._build_response(raw, runtime.model_version)

    def _get_runtime(self) -> PredictionRuntime:
        if self._runtime is not None:
            return self._runtime

        with self._load_lock:
            if self._runtime is not None:
                return self._runtime
            try:
                self._runtime = _BioCLIPRuntime(self.settings)
            except Exception as exc:
                logger.exception("BioCLIP model loading failed")
                raise ModelUnavailableError("BioCLIP model is unavailable") from exc
        return self._runtime

    def _build_response(
        self,
        raw: RawPrediction,
        model_version: str,
    ) -> IdentificationResponse:
        order_confidence = _probability(raw.order_confidence)

        if (
            raw.detected_order.casefold() != COLEOPTERA.casefold()
            and order_confidence < self.settings.order_threshold
        ):
            detected = raw.detected_order or "unknown"
            return IdentificationResponse(
                status="uncertain",
                identification="inconclusive visual classification",
                prediction=detected,
                confidence=order_confidence,
                detected_order=detected,
                reached_rank="order",
                taxonomy=Taxonomy(order=detected),
                requires_expert_confirmation=True,
                model_version=model_version,
                score_scope=(
                    "Species-level scores aggregated by order across the full "
                    "represented Tree of Life."
                ),
            )

        if raw.detected_order.casefold() != COLEOPTERA.casefold():
            detected = raw.detected_order or "unknown"
            return IdentificationResponse(
                status="not_coleoptera",
                identification="the model suggests it does not belong to the order Coleoptera",
                prediction=detected,
                confidence=order_confidence,
                detected_order=detected,
                reached_rank="order",
                taxonomy=Taxonomy(order=detected),
                requires_expert_confirmation=True,
                model_version=model_version,
                score_scope=(
                    "Species-level scores aggregated by order across the full "
                    "represented Tree of Life."
                ),
            )

        if order_confidence < self.settings.order_threshold or not raw.candidates:
            return IdentificationResponse(
                status="uncertain",
                identification="possible organism of the order Coleoptera",
                prediction=COLEOPTERA,
                confidence=order_confidence,
                order=COLEOPTERA,
                reached_rank="order",
                taxonomy=Taxonomy(order=COLEOPTERA),
                requires_expert_confirmation=True,
                model_version=model_version,
                score_scope=(
                    "Species-level scores aggregated by order across the full "
                    "represented Tree of Life."
                ),
            )

        top = raw.candidates[0]
        reached_rank, confidence = self._select_rank(raw)
        taxonomy = _taxonomy_to_rank(top.taxonomy, reached_rank)
        prediction = top.taxonomy.get(reached_rank) or COLEOPTERA
        candidates = [
            Candidate(
                scientific_name=species,
                common_name=item.common_name,
                confidence=_probability(item.confidence),
                taxonomy=_taxonomy_to_rank(item.taxonomy, "species"),
            )
            for item in raw.candidates
            if (species := item.taxonomy.get("species"))
        ]

        return IdentificationResponse(
            status="suggestion",
            identification=prediction,
            prediction=prediction,
            confidence=_probability(confidence),
            order=COLEOPTERA,
            reached_rank=reached_rank,
            taxonomy=taxonomy,
            candidates=candidates,
            requires_expert_confirmation=True,
            model_version=model_version,
            score_scope=(
                f"Score normalized among represented Coleoptera taxa at the {reached_rank} rank."
                if reached_rank != "order"
                else (
                    "Species-level scores aggregated by order across the full "
                    "represented Tree of Life."
                )
            ),
        )

    def _select_rank(self, raw: RawPrediction) -> tuple[str, float]:
        top = raw.candidates[0]
        if (
            top.taxonomy.get("species")
            and top.taxonomy.get("genus")
            and top.confidence >= self.settings.species_threshold
            and raw.species_margin >= self.settings.species_margin
        ):
            return "species", top.confidence
        if top.taxonomy.get("genus") and raw.genus_confidence >= self.settings.genus_threshold:
            return "genus", raw.genus_confidence
        if top.taxonomy.get("family") and raw.family_confidence >= self.settings.family_threshold:
            return "family", raw.family_confidence
        return "order", raw.order_confidence


class _BioCLIPRuntime:
    """Adapter around pybioclip that computes image features only once."""

    def __init__(self, settings: IdentifierSettings) -> None:
        import torch
        from bioclip import TreeOfLifeClassifier

        self._torch = torch
        self._classifier = TreeOfLifeClassifier(
            model_str=settings.model_name,
            device=settings.device,
        )
        self.model_version = _model_version(settings.model_name)

        label_data = self._classifier.get_label_data()
        order_values = [_clean_taxon(value) or "unknown" for value in label_data["order"]]
        order_codes, self._order_names = _encode_groups(order_values)
        self._order_codes = torch.tensor(order_codes, dtype=torch.long)

        coleoptera_positions = [
            index
            for index, value in enumerate(order_values)
            if value.casefold() == COLEOPTERA.casefold()
        ]
        if not coleoptera_positions:
            raise RuntimeError("Tree of Life labels do not contain Coleoptera")

        self._coleoptera_positions = torch.tensor(coleoptera_positions, dtype=torch.long)
        coleoptera_data = label_data.iloc[coleoptera_positions]
        self._taxonomies = [
            {
                rank: _clean_taxon(record.get(rank))
                for rank in (
                    "kingdom",
                    "phylum",
                    "class",
                    "order",
                    "family",
                    "genus",
                    "species",
                )
            }
            | {"common_name": _clean_taxon(record.get("common_name"))}
            for record in coleoptera_data.to_dict(orient="records")
        ]

        family_codes, self._family_names = _encode_groups(
            [item["family"] or "unknown" for item in self._taxonomies]
        )
        genus_codes, self._genus_names = _encode_groups(
            [item["genus"] or "unknown" for item in self._taxonomies]
        )
        self._family_codes = torch.tensor(family_codes, dtype=torch.long)
        self._genus_codes = torch.tensor(genus_codes, dtype=torch.long)

    def predict(self, image: Image.Image, top_k: int) -> RawPrediction:
        probabilities = self._classifier.create_batched_probabilities_for_images(
            [image],
            self._classifier.get_txt_embeddings(),
            batch_size=1,
            callback=lambda _processed, _total: None,
        )["0"].cpu()

        order_scores = self._aggregate(probabilities, self._order_codes, self._order_names)
        detected_order, order_confidence = max(order_scores.items(), key=lambda item: item[1])
        if detected_order.casefold() != COLEOPTERA.casefold():
            return RawPrediction(
                detected_order=detected_order,
                order_confidence=order_confidence,
            )

        coleoptera_scores = probabilities.index_select(0, self._coleoptera_positions)
        total = float(coleoptera_scores.sum().item())
        if total <= 0:
            return RawPrediction(
                detected_order=COLEOPTERA,
                order_confidence=order_confidence,
            )
        coleoptera_scores = coleoptera_scores / total

        candidate_count = min(top_k, len(self._taxonomies))
        top = coleoptera_scores.topk(candidate_count)
        candidates = tuple(
            RawCandidate(
                taxonomy={
                    rank: value
                    for rank, value in self._taxonomies[int(index)].items()
                    if rank != "common_name" and value
                },
                common_name=self._taxonomies[int(index)].get("common_name") or None,
                confidence=float(score),
            )
            for index, score in zip(top.indices.tolist(), top.values.tolist(), strict=True)
        )

        top_index = int(top.indices[0])
        top_taxonomy = self._taxonomies[top_index]
        family_scores = self._aggregate(
            coleoptera_scores,
            self._family_codes,
            self._family_names,
        )
        genus_scores = self._aggregate(
            coleoptera_scores,
            self._genus_codes,
            self._genus_names,
        )
        margin = float(top.values[0])
        if len(top.values) > 1:
            margin -= float(top.values[1])

        return RawPrediction(
            detected_order=COLEOPTERA,
            order_confidence=order_confidence,
            candidates=candidates,
            family_confidence=family_scores.get(top_taxonomy["family"] or "unknown", 0.0),
            genus_confidence=genus_scores.get(top_taxonomy["genus"] or "unknown", 0.0),
            species_margin=margin,
        )

    def _aggregate(
        self,
        probabilities: Any,
        group_codes: Any,
        group_names: Sequence[str],
    ) -> dict[str, float]:
        scores = self._torch.zeros(len(group_names), dtype=probabilities.dtype)
        scores.scatter_add_(0, group_codes, probabilities)
        return dict(
            zip(
                group_names,
                (float(value) for value in scores.tolist()),
                strict=True,
            )
        )


def _decode_image(image_bytes: bytes) -> Image.Image:
    if not image_bytes:
        raise InvalidImageError("The image is empty")

    try:
        with Image.open(io.BytesIO(image_bytes)) as source:
            width, height = source.size
            if width <= 0 or height <= 0:
                raise InvalidImageError("The image dimensions are invalid")
            if min(width, height) < MIN_IMAGE_DIMENSION:
                raise InvalidImageError(
                    f"The image is too small; both dimensions must be at least {MIN_IMAGE_DIMENSION} pixels"
                )
            if width * height > MAX_IMAGE_PIXELS:
                raise InvalidImageError("The image dimensions exceed the safety limit")
            aspect_ratio = max(width, height) / min(width, height)
            if aspect_ratio > MAX_ASPECT_RATIO:
                raise InvalidImageError("The image aspect ratio is not supported")
            source.seek(0)
            image = ImageOps.exif_transpose(source).convert("RGB")
            image.load()
            return image
    except InvalidImageError:
        raise
    except (UnidentifiedImageError, OSError, Image.DecompressionBombError) as exc:
        raise InvalidImageError("The submitted content is not a valid image") from exc


def _taxonomy_to_rank(values: dict[str, str], reached_rank: str) -> Taxonomy:
    ranks = ("kingdom", "phylum", "class", "order", "family", "genus", "species")
    allowed: dict[str, str | None] = {}
    for rank in ranks:
        allowed["class_" if rank == "class" else rank] = values.get(rank) or None
        if rank == reached_rank:
            break
    return Taxonomy(**allowed)


def _encode_groups(values: Sequence[str]) -> tuple[list[int], list[str]]:
    names: list[str] = []
    indexes: dict[str, int] = {}
    codes: list[int] = []
    for value in values:
        if value not in indexes:
            indexes[value] = len(names)
            names.append(value)
        codes.append(indexes[value])
    return codes, names


def _clean_taxon(value: Any) -> str:
    if value is None:
        return ""
    cleaned = str(value).strip()
    if cleaned.casefold() in {"nan", "none", "<na>"}:
        return ""
    return cleaned


def _model_version(model_name: str) -> str:
    if model_name == DEFAULT_MODEL:
        return "bioclip-v1-tol10m"
    if model_name == "hf-hub:imageomics/bioclip-2":
        return "bioclip-v2-tol200m"
    return model_name


def _probability(value: float) -> float:
    return min(max(float(value), 0.0), 1.0)


def _read_probability(name: str, default: float) -> float:
    value = float(os.getenv(name, str(default)))
    if not 0.0 <= value <= 1.0:
        raise ValueError(f"{name} must be between 0 and 1")
    return value


def _read_int(name: str, default: int, minimum: int, maximum: int) -> int:
    value = int(os.getenv(name, str(default)))
    if not minimum <= value <= maximum:
        raise ValueError(f"{name} must be between {minimum} and {maximum}")
    return value


identifier = BioCLIPIdentifier()
