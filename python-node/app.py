from flask import Flask, request, jsonify
import yfinance as yf
import pandas as pd
import numpy as np
from scipy.optimize import minimize
from sqlalchemy import create_engine, text, bindparam
import os

app = Flask(__name__)

# same connection string format as Go, reads from environment
DATABASE_URL = os.environ.get("DATABASE_URL", "postgresql://admin:pass@db:5432/robo_advisory")
engine = create_engine(DATABASE_URL)

# ETF universe
TICKERS = ["SPY", "QQQ", "BND", "GLD", "VNQ"]

def fetch_and_store_prices():
    df = yf.download(TICKERS, period="2y", interval="1d")["Close"]
    
    print("DataFrame shape:", df.shape)
    print("DataFrame head:", df.head())
    print("DataFrame columns:", df.columns.tolist())

    rows = []
    for ticker in TICKERS:
        for date, close_price in df[ticker].dropna().items():
            rows.append({
                "ticker": ticker,
                "date": date.date(),
                "close_price": float(close_price)
            })

    # insert into DB
    with engine.begin() as conn:
        for row in rows:
            conn.execute(
                text("""
                    INSERT INTO historical_market_data (ticker, date, close_price, created_at)
                    VALUES (:ticker, :date, :close_price, NOW())
                    ON CONFLICT (ticker, date) DO UPDATE SET close_price = EXCLUDED.close_price
                """),
                row
            )

    print("Rows inserted:", len(rows))
    return len(rows)


@app.route('/sync', methods=['POST'])
def sync():
    try:
        row_count = fetch_and_store_prices()
        return jsonify({
            "message": "Data synced successfully",
            "rows_inserted": row_count,
            "tickers": TICKERS
        }), 200
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=True)