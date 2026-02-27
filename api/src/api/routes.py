from fastapi import APIRouter, HTTPException, Depends
from src.schemas.task import CrawlRequest, TaskResponse
from src.core.protocols import TaskQueueGateway
from src.api.dependencies import get_rabbitmq_service
import pika

router = APIRouter(tags=["Crawler Operations"])

@router.post("/crawl", summary="Enqueue a URL for crawling", response_model=TaskResponse)
async def enqueue_crawl(
    request: CrawlRequest, 
    service: TaskQueueGateway = Depends(get_rabbitmq_service)
):
    service.publish_task(url=request.url, depth=request.depth, dir_name=request.dir_name)
    return TaskResponse(status="success", message=f"URL {request.url} enqueued successfully to directory '{request.dir_name}'.")

@router.post("/purge", summary="Purge the crawler queue", response_model=TaskResponse)
async def purge_queue(service: TaskQueueGateway = Depends(get_rabbitmq_service)):
    service.purge_queue()
    return TaskResponse(status="success", message="PURGE command enqueued successfully.")
