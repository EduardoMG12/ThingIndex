from fastapi import APIRouter, HTTPException
from src.schemas.task import CrawlRequest
from src.services.rabbitmq import rabbitmq_service
import pika

router = APIRouter(tags=["Crawler Operations"])

@router.post("/crawl", summary="Enqueue a URL for crawling")
async def enqueue_crawl(request: CrawlRequest):
    try:
        rabbitmq_service.publish_task(request.url, request.depth)
        return {"status": "success", "message": f"URL {request.url} enqueued successfully."}
    except pika.exceptions.AMQPError as e:
        raise HTTPException(status_code=500, detail=f"RabbitMQ connection error: {str(e)}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/purge", summary="Purge the crawler queue")
async def purge_queue():
    try:
        rabbitmq_service.purge_queue()
        return {"status": "success", "message": "PURGE command enqueued successfully."}
    except pika.exceptions.AMQPError as e:
        raise HTTPException(status_code=500, detail=f"RabbitMQ connection error: {str(e)}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
