# Use Python 3.12 slim image
FROM python:3.12-slim

WORKDIR /app

# Install system dependencies for Playwright
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    wget \
    gnupg \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Copy dependency files
COPY pyproject.toml ./
COPY uv.lock ./

# Sync dependencies and install Playwright browsers
RUN uv sync --frozen --no-dev && \
    uv run playwright install --with-deps chromium

# Copy application code
COPY . .

# Expose port
EXPOSE 8000

# Run the application using uv
CMD ["uv", "run", "python", "main.py"]
