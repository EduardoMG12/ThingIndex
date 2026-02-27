import logging
from fastapi import FastAPI
from fastapi.responses import HTMLResponse
from fastapi.middleware.cors import CORSMiddleware
from src.core.lifespan import lifespan
from src.core.exceptions import setup_exception_handlers
from src.api.routes import router as operations_router
from src.api.docs import docs_router
from src.api.monitoring import monitoring_router
from src.api.graph import router as graph_router

logging.basicConfig(level=logging.INFO)

app = FastAPI(
    title="LegisGraph API", 
    description="API to interact with the LegisGraph Crawler", 
    version="1.0.0", 
    docs_url=None,
    lifespan=lifespan
)

setup_exception_handlers(app)

app.include_router(operations_router)
app.include_router(docs_router)
app.include_router(monitoring_router)
app.include_router(graph_router, prefix="/api")

@app.get("/", include_in_schema=False)
async def root():
    return HTMLResponse('<h1>Crawler API</h1><p>Visit <a href="/scalar">/scalar</a> for documentation.</p>')

