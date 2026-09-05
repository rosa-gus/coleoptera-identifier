"""Optional GBIF name normalization with an in-memory cache."""

from __future__ import annotations

import asyncio
import logging
import time
from dataclasses import dataclass

import httpx

logger = logging.getLogger(__name__)

GBIF_MATCH_URL = "https://api.gbif.org/v2/species/match"
CACHE_TTL_SECONDS = 24 * 60 * 60


@dataclass(frozen=True)
class GBIFMatch:
    """Accepted taxonomic information returned by GBIF."""

    taxon_key: str | None
    scientific_name: str
    status: str | None


class GBIFClient:
    """Best-effort async client; GBIF failures never invalidate inference."""

    def __init__(self, timeout: float = 3.0) -> None:
        self._timeout = timeout
        self._cache: dict[str, tuple[float, GBIFMatch | None]] = {}
        self._lock = asyncio.Lock()

    async def match_species(self, scientific_name: str) -> GBIFMatch | None:
        """Normalize one scientific name against the GBIF backbone."""

        key = scientific_name.casefold().strip()
        if not key:
            return None

        now = time.monotonic()
        cached = self._cache.get(key)
        if cached and now - cached[0] < CACHE_TTL_SECONDS:
            return cached[1]

        async with self._lock:
            cached = self._cache.get(key)
            if cached and now - cached[0] < CACHE_TTL_SECONDS:
                return cached[1]

            match = await self._fetch_match(scientific_name)
            self._cache[key] = (time.monotonic(), match)
            return match

    async def _fetch_match(self, scientific_name: str) -> GBIFMatch | None:
        try:
            async with httpx.AsyncClient(timeout=self._timeout) as client:
                response = await client.get(
                    GBIF_MATCH_URL,
                    params={
                        "scientificName": scientific_name,
                        "kingdom": "Animalia",
                    },
                )
                response.raise_for_status()
        except (httpx.HTTPError, ValueError):
            logger.warning("GBIF normalization failed", exc_info=True)
            return None

        try:
            data = response.json()
        except ValueError:
            logger.warning("GBIF returned invalid JSON", exc_info=True)
            return None
        usage = data.get("usage") or {}
        accepted = data.get("acceptedUsage") or usage
        canonical_name = (
            accepted.get("canonicalName")
            or usage.get("canonicalName")
            or accepted.get("name")
            or usage.get("name")
        )
        if not canonical_name:
            return None

        return GBIFMatch(
            taxon_key=_as_optional_string(accepted.get("key") or usage.get("key")),
            scientific_name=str(canonical_name),
            status=_as_optional_string(accepted.get("status") or usage.get("status")),
        )


def _as_optional_string(value: object) -> str | None:
    return str(value) if value is not None else None


gbif_client = GBIFClient()
