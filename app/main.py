"""FastAPI application for the Coleoptera identification prototype."""

from fastapi import FastAPI

from app.routers import identification
from app.services.bioclip import identifier

app = FastAPI(
    title="Coleoptera Identifier",
    description=(
        "Server-side BioCLIP suggestions for organisms in the order Coleoptera. "
        "Results are visual hypotheses and do not replace expert determination."
    ),
    version="0.1.0",
)

app.include_router(identification.router, prefix="/api", tags=["Identification"])


@app.get("/", tags=["Root"])
async def read_root():
    return {
        "name": "Coleoptera Identifier",
        "version": app.version,
        "docs": "/docs",
    }


@app.get("/health", tags=["Health"])
async def health():
    """Report process health without triggering a model download."""

    return {"status": "ok", "model_loaded": identifier.is_loaded}
