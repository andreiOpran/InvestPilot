import logging

import numpy as np
import pandas as pd
import yfinance as yf

from repositories.db_repository import DataRepository
from services.hrp_service import compute_hrp_weights
from services.monte_carlo_service import run_monte_carlo
from services.rebalance_service import compute_rebalance
from utils.debug import save_debug_csv


def process_sync_daily(payload: dict, repo: DataRepository):
    """
    DAILY DATA INGESTION PIPELINE - step 1 of daily background job
    
    Fetches 5 years of daily closing prices from Yahoo Finance
    for all tickers and puts them in daily_market_data
    table in the db
    """
    
    try:
        all_tickers = payload.get("equity_tickers", []) + payload.get("bond_tickers", [])

        # close gets the price at the end of trading day
        raw_download = yf.download(all_tickers, period="5y", interval="1d")
        raw_download_close = raw_download["Close"] if raw_download is not None else pd.DataFrame()
        
        rows_to_insert = []
        for ticker in all_tickers:
            if ticker not in raw_download_close.columns:
                continue
            
            for date_timestamp, close_price in raw_download_close[ticker].dropna().items():
                rows_to_insert.append({
                    "ticker":      ticker,
                    "date":        date_timestamp.date(),  # strip time, keep only date
                    "close_price": float(close_price)      # convert numpy float64 to python float
                })
        
        repo.save_daily_market_data(rows_to_insert, payload["data_lifetime"])
        
        return {
            "message":       "Daily sync complete",
            "rows_inserted": len(rows_to_insert),
            "tickers":       len(all_tickers)
        }
    except Exception as e:
        return {"error": str(e)}

def process_sync_intraday(payload: dict, repo: DataRepository):
    """
    INTRADAY PIPELINE
    
    Fetches 14 days of 15m interval prices from Yahoo Finance
    for all tickers and puts them in intraday_market_data
    table in the db
    """
    try:
        all_tickers = payload.get("equity_tickers", []) + payload.get("bond_tickers", [])
        
        # close gets the price at the end of the 15m interval
        raw_download = yf.download(all_tickers, period="14d", interval="15m")
        raw_download_close = raw_download["Close"] if raw_download is not None else pd.DataFrame()
        
        rows_to_insert = []
        for ticker in all_tickers:
            if ticker not in raw_download_close.columns:
                continue
            
            for timestamp, price in raw_download_close[ticker].dropna().items():
                rows_to_insert.append({
                    "ticker": ticker,
                    # remove timezone info for generic postgresql timestamp insertion
                    "timestamp": pd.to_datetime(timestamp).tz_localize(None),
                    "price": float(price)
                })

        repo.save_intraday_market_data(rows_to_insert, payload["data_lifetime"])
        
        return {
            "message":       "Intraday sync complete",
            "rows_inserted": len(rows_to_insert),
            "tickers":       len(all_tickers)
        }

    except Exception as e:
        return {"error": str(e)}

def process_generate_models(payload: dict, repo: DataRepository):
    """
    HRP MODEL GENERATION - step 2 of daily background job
    
    Reads historical prices from db, runs HRP separately on
    equities and bonds, combines them using the macro
    allocation table, and writes 15 pre-computed model
    portfolio buckets to the model_portfolios table
    (for various risk + investment horizon combinations).
    
    We cast to returns so we have absolute comparison,
    not regarding actual price of an asset
        * return = (today - yesterday) / yesterday
    
    Sharpe ratio = annualized return / annualized volatility
    (return earned per unit of risk taken, higher is better)
        * annualize return by multiplying daily mean with 252 (trading days)
        * annualize volatility by multiplying with sqrt(252) because
        volatility scales with square root of time (not linearly)
            (daily returns are approx independent, their variances
            add, and volatility is square root of variance)
    
    Go reads these buckets from the db on rebalance day
    """
    
    # load price data from the db, resulting in a tall table
    price_data_tall = repo.get_historical_prices_tall()
    
    if price_data_tall.empty:
        return {"error": "No data in the database. Run /sync first."}
    
    # pivot the table so we have a wide one,
    # with each day mapping to a singular row
    prices_wide = price_data_tall.pivot(
        index="date", columns="ticker", values="close_price"
    ).dropna(axis=1)
    
    # convert absolute pices to daily returns (percentages)
    daily_returns = prices_wide.pct_change().dropna()
    
    verbose = payload.get("verbose", False)
    if verbose:
        save_debug_csv(prices_wide, "0_prices_wide.csv")
        save_debug_csv(daily_returns, "1_daily_returns.csv")
        
    # filter out non existent tickers
    equity_tickers = payload.get("equity_tickers", [])
    bond_tickers = payload.get("bond_tickers", [])
    valid_equities = [t for t in equity_tickers if t in daily_returns.columns]
    valid_bonds    = [t for t in bond_tickers   if t in daily_returns.columns]
    
    # EQUITY SELECTION: keep only top N by sharpe ratio
    # running HRP on all equities prices tiny allocations which are useless
    equity_returns = daily_returns[valid_equities]
    if not equity_returns.empty:
        # sharpe ratio = annualized return / annualized volatility =
        # = return earned per unit of risk taken, higher is better
        sharpe_ratios = (
            (equity_returns.mean() * 252) /
            (equity_returns.std() * np.sqrt(252))
        )
        
        top_n_equities = payload.get("top_n_equities", 6)
        top_equity_tickers = sharpe_ratios.nlargest(top_n_equities).index.tolist()
        filtered_equity_retuns = daily_returns[top_equity_tickers]
    else:
        filtered_equity_retuns = pd.DataFrame()
        
    # compute HRP weights for top equity universe
    hrp_equity_weights = compute_hrp_weights(
        filtered_equity_retuns, verbose=verbose, prefix="equity"
    )
    
    # compute HRP weights for all bonds
    bond_returns = daily_returns[valid_bonds]
    hrp_bond_weights = compute_hrp_weights(
        bond_returns, verbose=verbose, prefix="bond"
    )
    
    # MACRO ALLOCATION: blend equity and bond weights using macro alloc table
    # {risk: equity allocation}, higher risk, more equities
    base_equity_allocation_raw = payload.get(
        "base_equity_allocation",
        {1: 0.20, 2: 0.40, 3: 0.60, 4: 0.80, 5: 0.90}
    )
    # JSON decoding casts ints to string, so we cast them back
    base_equity_allocation = {int(k): float(v) for k, v in base_equity_allocation_raw.items()}
    
    # horizon multiplier adjusts equity ration based on time horizon
    # longer horizon, more equities
    horizon_multipliers = payload.get("horizon_multipliers", {"short": 0.7, "medium": 1.0, "long": 1.1})
    
    all_buckets = {}
    
    for risk_level in range(1, 6):
        for horizon_name, horizon_mult in horizon_multipliers.items():
            bucket_key = f"risk_{risk_level}_horizon_{horizon_name}"
            
            # apply horizon multiplier to equities
            equity_ratio = base_equity_allocation[risk_level] * horizon_mult
            
            # cap equities at configured threshold
            max_equity_cap = payload.get("max_equity_cap", 0.95)
            equity_ratio = min(equity_ratio, max_equity_cap)
            
            # bond fills remainder
            bond_ratio = 1.0 - equity_ratio
            
            # scale each HRP weight by the macro allocation
            # after loop all weights combined sum to 1.0
            raw_weights = {}
            for ticker, hrp_weight in hrp_equity_weights.items():
                raw_weights[ticker] = float(hrp_weight * equity_ratio)
            for ticker, hrp_weight in hrp_bond_weights.items():
                raw_weights[ticker] = float(hrp_weight * bond_ratio)
                
            # weight cleanup: remove assets below minimum threshold
            weight_threshold = payload.get("weight_threshold", 0.02)
            clean_weights    = {}
            discarded_weight = 0.0
            
            for ticker, weight in raw_weights.items():
                if weight >= weight_threshold:
                    clean_weights[ticker] = weight
                else:
                    discarded_weight += weight
                    
            # redistribute discarded weight proportionally among survivors
            if len(clean_weights) > 0 and discarded_weight > 0:
                total_surviving_weight = sum(clean_weights.values())
                for ticker in clean_weights.keys():
                    # what fraction of surviving weight this ticker has
                    proportion = clean_weights[ticker] / total_surviving_weight
                    # give back same fraction of discarded weight
                    clean_weights[ticker] += (discarded_weight * proportion)
                    
            # round to 4 decimal for clean output
            final_weights = {k: round (v, 4) for k, v in clean_weights.items()}
            
            all_buckets[bucket_key] = final_weights # storing only weights
    
    # put buckets to model_portfolios table
    # go will read from db on rebalance day
    logging.info(f"Generated {len(all_buckets)} portfolio buckets, saving to DB...")
    repo.save_model_portfolios(all_buckets)
    
    return {"status": "success", "buckets_generated": len(all_buckets)}

def process_forecast(payload: dict, repo: DataRepository):
    """
    MONTE CARLO FORECAST: called by Go when a user requests a projection
    
    Reads historical returns for the user's specific portfolio tickers,
    computes portfolio's expected return and volatility, then runs
    simulated scenarios to produce the forecast
    """
    
    task_id = payload.get("task_id")
    if not task_id:
        logging.error("No task_id provided in CMD_FORECAST payload")
        return
    
    # load price data from the db, resulting in a tall table
    price_data_tall = repo.get_historical_prices_tall()
    
    if price_data_tall.empty:
        logging.error(f"Cannot run forecast for task {task_id}: No historical data.")
        repo.update_forecast_status(task_id, "error")
        return
    
    # pivot the table so we have a wide one,
    # with each day mapping to a singular row
    prices_wide = price_data_tall.pivot(
        index="date", columns="ticker", values="close_price"
    ).dropna(axis=1)
    
    # convert absolute pices to daily returns (percentages)
    # so we have do absolute comparison of assets
    daily_returns = prices_wide.pct_change().dropna()

    # filter to only the tickers in the user's portfolio
    weights_payload = payload.get("weights", {})
    tickers_in_portfolio = list(weights_payload.keys())
    portfolio_returns    = daily_returns[tickers_in_portfolio]
    
    # annualize mean daily return and covariance matrix
    mean_returns_annual = portfolio_returns.mean() * 252
    cov_matrix_annual   = portfolio_returns.cov()  * 252
    
    # build weight array in the same order as tickers_in_portfolio
    weight_array = np.array([weights_payload[t] for t in tickers_in_portfolio])
    
    # portfolio expected return = weighted average of inividual asset returns
    # np.sum(mean_returns_annual * weight_array) = sum(weight_i * return_i)
    portfolio_expected_return = np.sum(mean_returns_annual * weight_array)
    
    # portfolio volatility = sqrt(variance)
    # where variance = w' * Σ * w, w = weights array, Σ = covariance matrix
    # this accounts for individual asset variances and their co-movements
    portfolio_volatility = np.sqrt(
        np.dot(weight_array.T, np.dot(cov_matrix_annual, weight_array))
    )
    
    years_forecast = payload.get("years", 0)
    pessimistic, expected, optimistic = run_monte_carlo(
        mean_return     = portfolio_expected_return,
        volatility      = portfolio_volatility,
        initial_amount  = payload.get("initial_investment", 0),
        monthly_contrib = payload.get("monthly_contribution", 0),
        years           = years_forecast,
        verbose         = payload.get("verbose", False)
    )
    
    result_payload = {
        "years":                       list(range(years_forecast + 1)),
        "pessimistic_5th_percentile":  pessimistic,
        "expected_50th_percentile":    expected,
        "optimistic_95th_percentile":  optimistic,
        "stats": {
            # inputs that were given to monte carlo
            "historical_annual_return":     round(portfolio_expected_return, 4),
            "historical_annual_volatility": round(portfolio_volatility, 4)
        }
    }
    
    logging.info(f"Forecast complete for task {task_id}, saving to DB...")
    repo.update_forecast_status(task_id, "complete", result_payload)

def process_rebalance_user(payload: dict, repo: DataRepository):
    """
    REBALANCE SINGLE ANONYMOUS USER
    
    Rebalance anonymously using the model portfolio
    and the user portfolio (both have only weights)
    received from the payload, while applying business
    logic via service (threshold, cash_first)
    """
    
    req_id = payload.get("request_id")
    current_alloc = payload.get("current_allocation", {})
    target_weights = payload.get("target_weights", {})
    threshold = payload.get("threshold", 0.02)
    cash_first = payload.get("cash_first", True)
    
    if not req_id or not target_weights:
        logging.error("Missing required fields for Rebalance")
        return {"error": "request_id and target_weights required"}
    
    logging.info(f"Processing rebalance for {req_id}")
    
    # pass to service
    adjusted, skipped = compute_rebalance(
        current_alloc,
        target_weights,
        threshold,
        cash_first
    )
    
    reply = {
        "request_id": req_id,
        "adjusted_targets": adjusted,
        "skipped": skipped
    }
    
    logging.info(f"Rebalance {req_id} summary: {len(skipped)} assets skipped.")
    
    return reply

def process_rebalance_batch(payload: dict, repo: DataRepository):
    """
    REBALANCE BATCH OF ANONYMOUS USERS
    
    This is called monthly by the operational node.
    Parses an array of user portfolios and runs
    rebalance from service file on each portfolio
    """
    
    threshold = payload.get("threshold", 0.02)
    cash_first = payload.get("cash_first", True)
    users = payload.get("users", [])
    
    logging.info(f"Processing batch rebalance for {len(users)} users.")
    
    results = []
    
    for u_req in users:
        request_id = u_req.get("request_id")
        current_alloc = u_req.get("current_allocation", {})
        target_weights = u_req.get("target_weights", {})
        
        if not target_weights:
            continue
        
        # pass to service
        adjusted, skipped = compute_rebalance(
            current_alloc,
            target_weights,
            threshold,
            cash_first
        )
        
        results.append({
            "request_id": request_id,
            "adjusted_targets": adjusted,
            "skipped": skipped
        })
        
    logging.info(f"Batch rebalance complete. Returning {len(results)} results.")
    return {"results": results}
