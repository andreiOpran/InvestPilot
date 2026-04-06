from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    DATABASE_URL: str = "postgresql://admin:pass@db:5432/robo_advisory"
    RABBITMQ_URL: str = "amqp://guest:guest@rabbitmq:5672/"

    class Config:
        env_file = ".env-decisional-node"

settings = Settings()