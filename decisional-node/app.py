import json
import logging
import time

import pika

from config import settings
from handlers.command_handlers import (
    process_forecast,
    process_generate_models,
    process_rebalance_user,
    process_rebalance_batch,
    process_sync_daily,
    process_sync_intraday,
)
from repositories.db_repository import DataRepository

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')


def main():
    repo = DataRepository(settings.DATABASE_URL)
    
    max_retries = 10
    connection = None
    params = pika.URLParameters(settings.RABBITMQ_URL)
    params.heartbeat = 0
    
    # wait for rabbitmq to start with backoff
    for i in range(1, max_retries + 1):
        try:
            connection = pika.BlockingConnection(params)
            break
        except Exception as e:
            logging.warning(f"RabbitMQ connection failed (attempt {i}/{max_retries}): {e}")
            time.sleep(i * 2)
            
    if not connection:
        logging.error(f"Could not connect to RabbitMQ after {max_retries} attempts.")
        return
        
    channel = connection.channel()
    channel.queue_declare(queue="cmd_queue", durable=True)
    
    def callback(ch, method, properties, body):
        try:
            message = json.loads(body)
            command = message.get("command")
            payload = message.get("payload")
            
            logging.info(f"Received command: {command}")
            
            # capture result dict
            response = None
            
            if command == "CMD_SYNC_DAILY":
                response = process_sync_daily(payload, repo)
            elif command == "CMD_SYNC_INTRADAY":
                response = process_sync_intraday(payload, repo)
            elif command == "CMD_GENERATE":
                response = process_generate_models(payload, repo)
            elif command == "CMD_REBALANCE_USER":
                response = process_rebalance_user(payload, repo)
            elif command == "CMD_REBALANCE_BATCH":
                response = process_rebalance_batch(payload, repo)
            elif command == "CMD_FORECAST":
                response = process_forecast(payload, repo)
            else:
                logging.warning(f"Unknown command: {command}")
                response = {"error": f"Unknown command: {command}"}
        
            # check if operational-node expects an RPC reply
            if properties.reply_to and response is not None:
                ch.basic_publish(
                    exchange='',
                    routing_key = properties.reply_to,
                    properties=pika.BasicProperties(
                        correlation_id=properties.correlation_id,
                        content_type="application/json"
                    ),
                    body = json.dumps(response)
                )
                logging.info(f"RPC Reply sent to {properties.reply_to}")
        
        except Exception as e:
            logging.error(f"Error processing message: {e}")
        finally:
            # acknowledge message succesfully processed
            ch.basic_ack(delivery_tag=method.delivery_tag)
        
    # process one message at a time per worker
    channel.basic_qos(prefetch_count=1)
    channel.basic_consume(queue="cmd_queue", on_message_callback=callback)
    
    logging.info("Python Decisional Node started. Waiting for messages...")
    channel.start_consuming()

if __name__ == '__main__':
    main()