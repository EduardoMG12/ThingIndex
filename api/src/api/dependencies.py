from src.services.neo4j_client import neo4j_client
from src.services.rabbitmq import rabbitmq_service
from src.services.telemetry_consumer import telemetry_consumer
from src.core.protocols import LegalDataGateway, TaskQueueGateway, TelemetryGateway

def get_neo4j_client() -> LegalDataGateway:
    return neo4j_client

def get_rabbitmq_service() -> TaskQueueGateway:
    return rabbitmq_service

def get_telemetry_consumer() -> TelemetryGateway:
    return telemetry_consumer
