from pydantic import Field, field_validator
from typing import Optional
import re
from src.schemas.base import ProtectedSchema

class CrawlRequest(ProtectedSchema):
    url: str = Field(..., description="The URL to start crawling from")
    depth: int = Field(0, description="The depth of the crawl (0 means only this page, 1 means this page and its children, etc.)")
    dir_name: str = Field(default="", description="The target directory or sub-folder inside the Minio Bucket to save documents")

    @field_validator("url")
    @classmethod
    def validate_url(cls, v: str) -> str:
        if not re.match(r'^https?://', v):
            raise ValueError("URL must start with http:// or https://")
        return v

    @field_validator("depth")
    @classmethod
    def validate_depth(cls, v: int) -> int:
        if v < 0:
            raise ValueError("Depth must be a non-negative integer")
        return v

class TaskResponse(ProtectedSchema):
    status: str = Field(..., description="The status of the request (e.g., 'success', 'error')")
    message: str = Field(..., description="Details regarding the operation result")
