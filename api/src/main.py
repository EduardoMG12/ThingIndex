from fastapi import FastAPI
from fastapi.responses import HTMLResponse
from src.api.routes import router

app = FastAPI(title="Crawler API", description="API to interact with the LegisGraph Crawler", version="1.0.0", docs_url=None)

app.include_router(router)

@app.get("/scalar", include_in_schema=False)
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

@app.get("/", include_in_schema=False)
async def root():
    return HTMLResponse('<h1>Crawler API</h1><p>Visit <a href="/scalar">/scalar</a> for documentation.</p>')
