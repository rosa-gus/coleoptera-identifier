"""Schemas returned by the taxonomic identification endpoint."""

from typing import Literal

from pydantic import BaseModel, Field


class Taxonomy(BaseModel):
    """Linnaean path associated with a model suggestion."""

    kingdom: str | None = None
    phylum: str | None = None
    class_: str | None = Field(default=None, alias="class")
    order: str | None = None
    family: str | None = None
    genus: str | None = None
    species: str | None = None

    model_config = {"populate_by_name": True}


class Candidate(BaseModel):
    """A species-level visual suggestion produced by BioCLIP."""

    scientific_name: str
    common_name: str | None = None
    confidence: float = Field(
        ge=0.0,
        le=1.0,
        description=(
            "Uncalibrated model score normalized among represented Coleoptera taxa; "
            "it is not the probability that the determination is correct."
        ),
    )
    taxonomy: Taxonomy
    gbif_taxon_key: str | None = None
    accepted_scientific_name: str | None = None
    taxonomic_status: str | None = None


class IdentificationResponse(BaseModel):
    """Stable response for positive, negative, and uncertain predictions."""

    status: Literal["suggestion", "not_coleoptera", "uncertain"]
    identification: str
    prediction: str | None = None
    confidence: float = Field(
        ge=0.0,
        le=1.0,
        description="Uncalibrated model score, not an empirical probability of correctness.",
    )
    confidence_calibrated: bool = False
    confidence_interpretation: str = (
        "Model score used for conservative rank selection; not a scientific certainty."
    )
    order: str | None = None
    detected_class: str | None = None
    detected_order: str | None = None
    reached_rank: Literal["order", "family", "genus", "species"] = Field(
        description=(
            "Most specific rank allowed by configurable heuristic thresholds, "
            "not a verified taxonomic determination."
        )
    )
    taxonomy: Taxonomy | None = None
    candidates: list[Candidate] = Field(default_factory=list)
    requires_expert_confirmation: bool
    model_version: str
    score_scope: str
    candidate_scope: str = (
        "Coleoptera taxa represented by the configured BioCLIP Tree of Life embedding set."
    )
    disclaimer: str = (
        "Automated visual hypothesis, not a taxonomic determination. Species-level "
        "work may require additional views, collection metadata, microscopy, or a specialist."
    )
