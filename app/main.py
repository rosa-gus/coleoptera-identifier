from fastapi import FastAPI
from app.routers import identification

app = FastAPI(
    title="Coleoptera identificator",
    description="An experiment to find invertebrates of the order Coleoptera based on an image.",
    version="0.0.1",
)

app.include_router(identification.router, prefix="/api", tags=["Identification"])

@app.get("/", tags=["Root"])
async def read_root():
    return {"message": "hello world!"}