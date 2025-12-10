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

app = FastAPI(title="Email Writer Service", version="1.0.0")

LLAMA_MODEL = os.getenv("LLAMA_MODEL", "llama-3.3-70b-versatile")


class EmailRequest(BaseModel):
    job_title: str
    company: str
    manager_name: str | None = None
    manager_email: str
    user_name: str
    user_skills: list[str] = []
    user_experience: str | None = None


class EmailResponse(BaseModel):
    subject: str
    body: str
    tone: str = "professional"


@app.get("/health")
async def health_check() -> dict[str, str]:
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "email-writer",
        "version": "1.0.0",
        "model": LLAMA_MODEL
    }


@app.post("/api/v1/generate", response_model=EmailResponse)
async def generate_email(request: EmailRequest) -> EmailResponse:
    """
    Generate a personalized cold email for job outreach.
    This is a placeholder implementation that should be enhanced with actual LLM integration.
    """
    try:
        logger.info(
            "Generating email for %s to %s", request.user_name, request.company
        )

        # TODO: Implement actual email generation using LLAMA_API_KEY
        # This is a placeholder response
        manager_greeting = f"Dear {request.manager_name}" if request.manager_name else "Dear Hiring Manager"

        subject = f"Application for {request.job_title} at {request.company}"
        body = f"""{manager_greeting},

I hope this email finds you well. I am writing to express my interest in the {request.job_title} position at {request.company}.

With expertise in {', '.join(request.user_skills[:3]) if request.user_skills else 'various technologies'}, I believe I would be a strong fit for this role.

{request.user_experience if request.user_experience else 'I have significant experience in the field and am excited about the opportunity to contribute to your team.'}

I would appreciate the opportunity to discuss how my skills and experience align with your needs.

Best regards,
{request.user_name}"""

        return EmailResponse(
            subject=subject,
            body=body,
            tone="professional"
        )
    except Exception as e:
        logger.exception("Error generating email")
        raise HTTPException(status_code=500, detail=f"Failed to generate email: {e!s}") from e


def main() -> None:
    pass


if __name__ == "__main__":
    main()
