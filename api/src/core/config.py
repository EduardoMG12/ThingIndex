from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    rabbitmq_url: str
    rabbitmq_queue: str = "crawler_tasks"
    telemetry_exchange: str = "pipeline_telemetry"
    neo4j_uri: str
    neo4j_auth: str

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

settings = Settings()
