import pandas as pd
import json
from datetime import datetime, timezone
from psycopg2.extras import execute_values
from sqlalchemy import create_engine, text

class DataRepository:
    def __init__(self, db_url: str):
        self.engine = create_engine(db_url)

    def save_daily_market_data(self, rows_to_insert: list, data_lifetime: str):
        with self.engine.begin() as conn:
            cursor = conn.connection.cursor()
            execute_values(
                cursor,
                """
                    INSERT INTO daily_market_data (ticker, date, close_price, created_at)
                    VALUES %s
                    ON CONFLICT (ticker, date)
                    DO UPDATE SET close_price = EXCLUDED.close_price
                """,
                [(r['ticker'], r['date'], r['close_price']) for r in rows_to_insert],
                template="(%s, %s, %s, NOW())",
                page_size=1000
            )
            cursor.close()
            conn.execute(text(f"""
                DELETE FROM daily_market_data
                WHERE date < NOW() - INTERVAL '{data_lifetime}'
            """))
            
    def save_intraday_market_data(self, rows_to_insert: list, data_lifetime: str):
        with self.engine.begin() as conn:
            cursor = conn.connection.cursor()
            execute_values(
                cursor,
                """
                    INSERT INTO intraday_market_data (ticker, timestamp, price, created_at)
                    VALUES %s
                    ON CONFLICT (ticker, timestamp)
                    DO UPDATE SET price = EXCLUDED.price
                """,
                [(r['ticker'], r['timestamp'], r['price']) for r in rows_to_insert],
                template="(%s, %s, %s, NOW())",
                page_size=1000
            )
            cursor.close()
            conn.execute(text(f"""
                DELETE FROM intraday_market_data
                WHERE timestamp < NOW() - INTERVAL '{data_lifetime}'
            """))

    def get_historical_prices_tall(self) -> pd.DataFrame:
        query = "SELECT ticker, date, close_price FROM daily_market_data ORDER BY date ASC"
        return pd.read_sql(query, self.engine)

    def save_model_portfolios(self, all_buckets: dict):
        with self.engine.begin() as conn:
            for bucket_key, weights in all_buckets.items():
                conn.execute(
                    text("""
                    INSERT INTO model_portfolios (bucket_key, weights, computed_at, created_at)
                    VALUES (:key, :w, :now, :now)
                    """),
                    {"key": bucket_key, "w": json.dumps(weights), "now": datetime.now(timezone.utc)}
                )

    def update_forecast_status(self, task_id: str, status: str, payload: dict | None = None):
        with self.engine.begin() as conn:
            if payload:
                conn.execute(
                text("""
                UPDATE forecast_results 
                SET status = 'complete', payload = :payload, updated_at = :now
                WHERE task_id = :task_id
                """),
                {"status": status, "payload": json.dumps(payload), "task_id": task_id, "now": datetime.now(timezone.utc)}
                )
            else:
                conn.execute(
                    text("""
                    UPDATE forecast_results
                    SET status = :status, updated_at = :now
                    WHERE task_id = :task_id
                    """),
                    {"status": status, "task_id": task_id, "now": datetime.now(timezone.utc)}
                )