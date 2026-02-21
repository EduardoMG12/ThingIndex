import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI

logger = logging.getLogger("CrawlerAPI")

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Print beautiful route log
    logger.info("=========================================")
    logger.info("🚀 CRAWLER API COMPLETED STARTUP 🚀")
    logger.info("=========================================")
    logger.info("Available Routes:")
    for route in app.routes:
        if hasattr(route, "methods"):
            # Exclude default internal endpoints if desired, but let's show all
            methods = ", ".join(route.methods)
            logger.info(f"  {methods: <7} -> {route.path}")
    logger.info("=========================================")
    logger.info("Interactive Docs (Scalar): http://[host]:[port]/scalar")
    
    yield
    # Shutdown logic if any
    logger.info("API is shutting down...")
