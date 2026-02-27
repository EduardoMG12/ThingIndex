import logging
from fastapi import Request, status
from fastapi.responses import JSONResponse
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException

logger = logging.getLogger("GlobalExceptionHandler")

async def global_exception_handler(request: Request, exc: Exception):
    """
    Catches all unhandled exceptions and returns a structured JSON response.
    """
    logger.error(f"Unhandled exception: {str(exc)}", exc_info=True)
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "status": "error",
            "message": "Internal Server Error",
            "detail": str(exc) if "dev" in str(request.url) else "An unexpected error occurred."
        }
    )

async def http_exception_handler(request: Request, exc: StarletteHTTPException):
    """
    Passes through HTTPExceptions but ensuring standardized format if desired.
    """
    return JSONResponse(
        status_code=exc.status_code,
        content={
            "status": "error",
            "message": exc.detail
        }
    )

async def validation_exception_handler(request: Request, exc: RequestValidationError):
    """
    Handles Pydantic validation errors (422).
    """
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content={
            "status": "error",
            "message": "Validation Error",
            "errors": exc.errors()
        }
    )

def setup_exception_handlers(app):
    """Register all global handlers to the FastAPI app."""
    app.add_exception_handler(StarletteHTTPException, http_exception_handler)
    app.add_exception_handler(RequestValidationError, validation_exception_handler)
    app.add_exception_handler(Exception, global_exception_handler)
