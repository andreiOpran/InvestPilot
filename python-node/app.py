from fastapi import FastAPI
from yfinance import download
from sqlalchemy import create_engine, text
import os

app = FastAPI(title="Robo-Advisory Python Engine", version="1.0")

DATABASE_URL = os.environ.get("DATABASE_URL", "postgresql://admin:pass@db:5432/robo_advisory")
engine = create_engine(DATABASE_URL)

TICKERS = ["SPY", "QQQ", "BND", "GLD", "VNQ"]

def fetch_and_store_prices():
    df = download(TICKERS, period="2y", interval="1d")["Close"]
    
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


@app.post('/generate-models')
def generate_models():
    # TODO: replace with real Markowitz optimizer reading from historical_market_data table
    # 15 buckets (risk 1-5 x horizon short/medium/long)
    # lower risk = more BND/GLD, higher risk = more SPY/QQQ
    # shorter horizon = more conservative allocation regardless of risk level
    mock_prices = {
        "SPY": 510.00,
        "QQQ": 430.00,
        "BND": 72.50,
        "GLD": 185.00,
        "VNQ": 85.00
    }

    mock_weights = {
        "risk_1_horizon_short":  {"BND": 0.70, "GLD": 0.20, "VNQ": 0.05, "SPY": 0.05, "QQQ": 0.00},
        # "risk_1_horizon_medium": {"BND": 0.65, "GLD": 0.20, "VNQ": 0.10, "SPY": 0.05, "QQQ": 0.00},
        # "risk_1_horizon_long":   {"BND": 0.55, "GLD": 0.20, "VNQ": 0.15, "SPY": 0.10, "QQQ": 0.00},

        "risk_2_horizon_short":  {"BND": 0.55, "GLD": 0.20, "VNQ": 0.10, "SPY": 0.15, "QQQ": 0.00},
        # "risk_2_horizon_medium": {"BND": 0.45, "GLD": 0.20, "VNQ": 0.15, "SPY": 0.20, "QQQ": 0.00},
        # "risk_2_horizon_long":   {"BND": 0.35, "GLD": 0.15, "VNQ": 0.15, "SPY": 0.30, "QQQ": 0.05},

        "risk_3_horizon_short":  {"BND": 0.40, "GLD": 0.15, "VNQ": 0.10, "SPY": 0.30, "QQQ": 0.05},
        # "risk_3_horizon_medium": {"BND": 0.30, "GLD": 0.10, "VNQ": 0.15, "SPY": 0.35, "QQQ": 0.10},
        # "risk_3_horizon_long":   {"BND": 0.20, "GLD": 0.10, "VNQ": 0.15, "SPY": 0.40, "QQQ": 0.15},

        "risk_4_horizon_short":  {"BND": 0.25, "GLD": 0.10, "VNQ": 0.10, "SPY": 0.40, "QQQ": 0.15},
        # "risk_4_horizon_medium": {"BND": 0.15, "GLD": 0.05, "VNQ": 0.10, "SPY": 0.45, "QQQ": 0.25},
        # "risk_4_horizon_long":   {"BND": 0.10, "GLD": 0.05, "VNQ": 0.05, "SPY": 0.50, "QQQ": 0.30},

        "risk_5_horizon_short":  {"BND": 0.15, "GLD": 0.05, "VNQ": 0.10, "SPY": 0.45, "QQQ": 0.25},
        # "risk_5_horizon_medium": {"BND": 0.05, "GLD": 0.05, "VNQ": 0.10, "SPY": 0.45, "QQQ": 0.35},
        # "risk_5_horizon_long":   {"BND": 0.00, "GLD": 0.05, "VNQ": 0.10, "SPY": 0.45, "QQQ": 0.40},
    }

    result = {}
    for bucket, weights in mock_weights.items():
        result[bucket] = {
            "weights": weights,
            "prices": mock_prices
        }

    return result

@app.post('/sync')
def sync():
    try:
        row_count = fetch_and_store_prices()
        return {
            "message": "Data synced successfully",
            "rows_inserted": row_count,
            "tickers": TICKERS
        }
    except Exception as e:
        return {"error": str(e)}

if __name__ == '__main__':
    import uvicorn
    uvicorn.run(app, host='0.0.0.0', port=5000)
