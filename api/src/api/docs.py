from fastapi import APIRouter
from fastapi.responses import HTMLResponse

docs_router = APIRouter(tags=["Documentation"])

@docs_router.get("/scalar", include_in_schema=False)
async def scalar_html():
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
      <head>
        <title>Crawler API - Scalar</title>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </head>
      <body>
        <script id="api-reference" data-url="/openapi.json"></script>
        <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
      </body>
    </html>
    """)
