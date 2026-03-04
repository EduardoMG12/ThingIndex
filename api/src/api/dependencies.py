from src.services.neo4j_connection import neo4j_connection
from src.services.neo4j_graph_service import Neo4jGraphService
from src.services.neo4j_assistant_service import Neo4jAssistantService
from src.services.rabbitmq import rabbitmq_service
from src.services.telemetry_consumer import telemetry_consumer
from src.core.protocols import GraphDataGateway, LawSearchGateway, TaskQueueGateway, TelemetryGateway

# Services are instantiated once sharing the same connection
_graph_service = Neo4jGraphService(neo4j_connection)
_assistant_service = Neo4jAssistantService(neo4j_connection)


def get_graph_service() -> GraphDataGateway:
    return _graph_service


def get_assistant_service() -> LawSearchGateway:
    return _assistant_service


def get_rabbitmq_service() -> TaskQueueGateway:
    return rabbitmq_service


def get_telemetry_consumer() -> TelemetryGateway:
    return telemetry_consumer
