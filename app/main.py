"""FastAPI application for the Coleoptera identification prototype.

SPDX-License-Identifier: GPL-3.0-only
"""

from fastapi import FastAPI

from app.routers import identification
from app.services.bioclip import identifier

LICENSE_NAME = "GNU General Public License v3.0"
LICENSE_IDENTIFIER = "GPL-3.0-only"

app = FastAPI(
    title="Coleoptera Identifier",
    description=(
        "Server-side BioCLIP suggestions for organisms in the order Coleoptera. "
        "Results are visual hypotheses and do not replace expert determination."
    ),
    version="0.1.0",
    license_info={
        "name": LICENSE_NAME,
        "identifier": LICENSE_IDENTIFIER,
    },
)

app.include_router(identification.router, prefix="/api", tags=["Identification"])


@app.get("/", tags=["Root"])
async def read_root():
    return {
        "name": "Coleoptera Identifier",
        "version": app.version,
        "license": LICENSE_IDENTIFIER,
        "docs": "/docs",
    }


@app.get("/health", tags=["Health"])
async def health():
    """Report process health without triggering a model download."""

    return {"status": "ok", "model_loaded": identifier.is_loaded}
