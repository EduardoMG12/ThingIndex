FROM python:3.11-slim

WORKDIR /app

# Install PDM
RUN pip install -U pdm
ENV PDM_CHECK_UPDATE=false

COPY pyproject.toml README.md ./

# Generate lock and install dependencies system-wide
RUN pdm lock && pdm install --prod --no-editable

COPY src/ ./src/

CMD ["pdm", "run", "uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8000"]
