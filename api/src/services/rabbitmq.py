import pika
import json
import logging
from src.core.config import settings

logger = logging.getLogger(__name__)

class RabbitMQService:
    def __init__(self):
        self.url = settings.rabbitmq_url
        self.queue_name = settings.rabbitmq_queue

    def _get_connection(self):
        parameters = pika.URLParameters(self.url)
        return pika.BlockingConnection(parameters)

    def publish_task(self, url: str, depth: int, dir_name: str = "", parent: str = "", description: str = "", priority: int = 0):
        connection = self._get_connection()
        channel = connection.channel()
        channel.queue_declare(queue=self.queue_name, durable=True, arguments={"x-max-priority": 10})

        payload = {
            "url": url,
            "parent": parent,
            "parent_context": {},
            "description": description,
            "depth": 0,
            "max_depth": depth,
            "dir_name": dir_name
        }

        channel.basic_publish(
            exchange='',
            routing_key=self.queue_name,
            body=json.dumps(payload),
            properties=pika.BasicProperties(
                delivery_mode=pika.spec.PERSISTENT_DELIVERY_MODE,
                content_type="application/json",
                priority=priority
            )
        )
        connection.close()
        logger.info(f"Published task for URL: {url} at depth {depth}")

    def purge_queue(self):
        # We send the magic purge command with priority 10
        self.publish_task("__PURGE__", 0, priority=10)
        logger.info("Published PURGE command to the queue.")

rabbitmq_service = RabbitMQService()
