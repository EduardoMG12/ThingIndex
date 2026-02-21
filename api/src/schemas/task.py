from pydantic import BaseModel, Field

class CrawlRequest(BaseModel):
    url: str = Field(..., description="The URL to start crawling from")
    depth: int = Field(0, description="The depth of the crawl (0 means only this page, 1 means this page and its children, etc.)")
    dir_name: str = Field(default="", description="The target directory or sub-folder inside the Minio Bucket to save documents")

class TaskResponse(BaseModel):
    status: str = Field(..., description="The status of the request (e.g., 'success', 'error')")
    message: str = Field(..., description="Details regarding the operation result")
