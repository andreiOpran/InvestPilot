import pandas as pd
import json
from datetime import datetime, timezone
from sqlalchemy import create_engine, text

class DataRepository:
    def __init__(self, db_url: str):
        self.engine = create_engine(db_url)

    def save_daily_market_data(self, rows_to_insert: list, data_lifetime: str):
        # open transaction with auto-commit/rollback
        with self.engine.begin() as conn:
            for row in rows_to_insert:
                conn.execute(
                    text("""
                        INSERT INTO daily_market_data (ticker, date, close_price, created_at)
                        VALUES (:ticker, :date, :close_price, NOW())
                        ON CONFLICT (ticker, date)
                        DO UPDATE SET close_price = EXCLUDED.close_price
                    """),
                    # ON CONFLICT: if a row for this (ticker, date) already exists,
                    # update its price instead of throwing a duplicate key error
                    row
                )
            # delete rows for data older than 5 years
            conn.execute(text(f"""
                DELETE FROM daily_market_data
                WHERE date < NOW() - INTERVAL '{data_lifetime}'
            """))
            
    def save_intraday_market_data(self, rows_to_insert: list, data_lifetime: str):
        # open transaction with auto-commit/rollback
        with self.engine.begin() as conn:
            for row in rows_to_insert:
                conn.execute(
                    text("""
                        INSERT INTO intraday_market_data (ticker, timestamp, price, created_at)
                        VALUES (:ticker, :timestamp, :price, NOW())
                        ON CONFLICT (ticker, timestamp)
                        DO UPDATE SET price = EXCLUDED.price
                    """),
                    # ON CONFLICT: if a row for this (ticker, timestamp) already exists,
                    # update its price instead of throwing a duplicate key error
                    row
                )
            
            # delete rows for data older than 14 days
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