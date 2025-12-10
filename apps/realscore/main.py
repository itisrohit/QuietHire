import logging
import os

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

# Configure logging
logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

app = FastAPI(title="RealScore Service", version="1.0.0")

MIN_THRESHOLD = int(os.getenv("MIN_REALSCORE_THRESHOLD", "70"))


class ScoreRequest(BaseModel):
    job_title: str
    job_description: str
    company: str
    url: str


class RealScoreResponse(BaseModel):
    score: int
    is_real: bool
    confidence: float
    reasons: list[str] = []
    flags: list[str] = []


@app.get("/health")
async def health_check() -> dict[str, str | int]:
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "realscore",
        "version": "1.0.0",
        "min_threshold": MIN_THRESHOLD
    }


@app.post("/api/v1/score", response_model=RealScoreResponse)
async def calculate_realscore(request: ScoreRequest) -> RealScoreResponse:
    """
    Calculate RealScore for a job posting to determine if it's a ghost job.
    This is a placeholder implementation that should be enhanced with actual ML/AI logic.
    """
    try:
        logger.info(
            "Calculating RealScore for job: %s at %s",
            request.job_title,
            request.company,
        )

        # TODO: Implement actual RealScore calculation using GROQ_API_KEY
        # This is a placeholder response
        score = 85
        is_real = score >= MIN_THRESHOLD

        return RealScoreResponse(
            score=score,
            is_real=is_real,
            confidence=0.92,
            reasons=[
                "Job posting has detailed requirements",
                "Company has active online presence",
                "Salary range is provided",
            ],
            flags=[] if is_real else ["Vague job description", "No contact information"],
        )
    except Exception as e:
        logger.exception("Error calculating RealScore")
        raise HTTPException(
            status_code=500, detail=f"Failed to calculate RealScore: {e!s}"
        ) from e


def main() -> None:
    pass


if __name__ == "__main__":
    main()
