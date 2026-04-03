from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    APP_NAME: str = "Robo-Advisory Decisional Node"
    DEBUG: bool = False
    
    # Defaults align with your docker compose
    DATABASE_URL: str = "postgresql://admin:pass@db:5432/robo_advisory"
    RABBITMQ_URL: str = "amqp://guest:guest@rabbitmq:5672/"

    class Config:
        env_file = ".env-decisional-node"

settings = Settings()