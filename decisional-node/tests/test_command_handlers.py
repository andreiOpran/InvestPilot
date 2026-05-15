import numpy as np
import pandas as pd
from unittest.mock import MagicMock, patch

from handlers.command_handlers import (
    process_forecast,
    process_rebalance_batch,
    process_rebalance_user,
    process_sync_daily,
    process_sync_intraday,
)


def _make_repo():
    repo = MagicMock()
    repo.save_daily_market_data.return_value = None
    repo.save_intraday_market_data.return_value = None
    repo.save_model_portfolios.return_value = None
    repo.update_forecast_status.return_value = None
    return repo


def _make_tall_price_df(tickers=("SPY", "IEF"), n_dates=120):
    rng = np.random.default_rng(0)
    rows = []
    for ticker in tickers:
        prices = 100 * np.cumprod(1 + rng.normal(0.0003, 0.01, n_dates))
        for date, price in zip(pd.bdate_range("2020-01-01", periods=n_dates), prices):
            rows.append({"ticker": ticker, "date": date.date(), "close_price": float(price)})
    return pd.DataFrame(rows)


def _make_yfinance_close(tickers, n_days=5, tz=None):
    rng = np.random.default_rng(99)
    idx = pd.date_range("2024-01-02", periods=n_days, freq="B", tz=tz)
    return pd.DataFrame({t: 100 + rng.normal(0, 1, n_days) for t in tickers}, index=idx)


# process_rebalance_user

class TestProcessRebalanceUser:
    def test_valid_payload_returns_correct_structure(self):
        payload = {
            "request_id": "req-001",
            "current_allocation": {"AAPL": 0.5, "MSFT": 0.5},
            "target_weights": {"AAPL": 0.6, "MSFT": 0.4},
            "threshold": 0.05,
            "cash_first": False,
        }
        result = process_rebalance_user(payload, _make_repo())
        assert result["request_id"] == "req-001"
        assert isinstance(result["adjusted_targets"], dict)
        assert isinstance(result["skipped"], list)

    def test_adjusted_targets_sum_to_one(self):
        payload = {
            "request_id": "req-002",
            "current_allocation": {"AAPL": 0.4, "MSFT": 0.3, "SPY": 0.3},
            "target_weights": {"AAPL": 0.5, "MSFT": 0.3, "SPY": 0.2},
            "threshold": 0.05,
            "cash_first": False,
        }
        result = process_rebalance_user(payload, _make_repo())
        assert abs(sum(result["adjusted_targets"].values()) - 1.0) < 1e-3  # type: ignore[union-attr]

    def test_missing_request_id_returns_error(self):
        payload = {"target_weights": {"AAPL": 1.0}}
        result = process_rebalance_user(payload, _make_repo())
        assert "error" in result

    def test_missing_target_weights_returns_error(self):
        payload = {"request_id": "req-003", "current_allocation": {"AAPL": 1.0}}
        result = process_rebalance_user(payload, _make_repo())
        assert "error" in result

    def test_empty_target_weights_returns_error(self):
        payload = {"request_id": "req-004", "target_weights": {}}
        result = process_rebalance_user(payload, _make_repo())
        assert "error" in result

    def test_repo_never_called(self):
        # rebalance user is pure logic, no DB access
        repo = _make_repo()
        payload = {
            "request_id": "req-005",
            "current_allocation": {"AAPL": 1.0},
            "target_weights": {"AAPL": 0.7, "MSFT": 0.3},
        }
        process_rebalance_user(payload, repo)
        repo.save_daily_market_data.assert_not_called()
        repo.get_historical_prices_tall.assert_not_called()


# process_rebalance_batch

class TestProcessRebalanceBatch:
    def test_empty_users_returns_empty_results(self):
        result = process_rebalance_batch({"users": []}, _make_repo())
        assert result == {"results": []}

    def test_all_users_processed_in_order(self):
        payload = {
            "threshold": 0.05,
            "cash_first": False,
            "users": [
                {
                    "request_id": "u1",
                    "current_allocation": {"AAPL": 0.5, "MSFT": 0.5},
                    "target_weights": {"AAPL": 0.6, "MSFT": 0.4},
                },
                {
                    "request_id": "u2",
                    "current_allocation": {"SPY": 0.7, "IEF": 0.3},
                    "target_weights": {"SPY": 0.5, "IEF": 0.5},
                },
            ],
        }
        result = process_rebalance_batch(payload, _make_repo())
        assert len(result["results"]) == 2
        assert result["results"][0]["request_id"] == "u1"
        assert result["results"][1]["request_id"] == "u2"

    def test_user_with_empty_target_weights_skipped(self):
        payload = {
            "users": [
                {"request_id": "u1", "current_allocation": {"AAPL": 1.0}, "target_weights": {}},
                {"request_id": "u2", "current_allocation": {}, "target_weights": {"AAPL": 1.0}},
            ]
        }
        result = process_rebalance_batch(payload, _make_repo())
        assert len(result["results"]) == 1
        assert result["results"][0]["request_id"] == "u2"

    def test_each_result_has_required_keys(self):
        payload = {
            "users": [
                {
                    "request_id": "u1",
                    "current_allocation": {"AAPL": 1.0},
                    "target_weights": {"AAPL": 0.7, "MSFT": 0.3},
                }
            ]
        }
        result = process_rebalance_batch(payload, _make_repo())
        r = result["results"][0]
        assert "request_id" in r
        assert "adjusted_targets" in r
        assert "skipped" in r

    def test_missing_users_key_returns_empty_results(self):
        result = process_rebalance_batch({}, _make_repo())
        assert result == {"results": []}


# process_sync_daily

class TestProcessSyncDaily:
    def test_success_returns_correct_message_and_counts(self):
        tickers = ["AAPL", "IEF"]
        mock_close = _make_yfinance_close(tickers, n_days=5)
        repo = _make_repo()

        with patch("handlers.command_handlers.yf.download", return_value={"Close": mock_close}):
            result = process_sync_daily(
                {"equity_tickers": ["AAPL"], "bond_tickers": ["IEF"], "data_lifetime": "5 years"},
                repo,
            )

        assert result["message"] == "Daily sync complete"
        assert result["rows_inserted"] == len(tickers) * 5
        assert result["tickers"] == len(tickers)
        repo.save_daily_market_data.assert_called_once()

    def test_exception_during_download_returns_error_dict(self):
        repo = _make_repo()
        with patch("handlers.command_handlers.yf.download", side_effect=RuntimeError("timeout")):
            result = process_sync_daily(
                {"equity_tickers": [], "bond_tickers": [], "data_lifetime": "5 years"}, repo
            )
        assert "error" in result
        assert "timeout" in result["error"]

    def test_unknown_ticker_excluded_from_rows(self):
        # only "AAPL" in mock columns; "FAKE" will be skipped
        mock_close = _make_yfinance_close(["AAPL"], n_days=3)
        repo = _make_repo()

        with patch("handlers.command_handlers.yf.download", return_value={"Close": mock_close}):
            result = process_sync_daily(
                {"equity_tickers": ["AAPL", "FAKE"], "bond_tickers": [], "data_lifetime": "5 years"},
                repo,
            )

        assert result["rows_inserted"] == 3  # only AAPL's 3 rows


# process_sync_intraday

class TestProcessSyncIntraday:
    def test_success_returns_correct_message(self):
        tickers = ["AAPL", "IEF"]
        mock_close = _make_yfinance_close(tickers, n_days=8, tz="UTC")
        repo = _make_repo()

        with patch("handlers.command_handlers.yf.download", return_value={"Close": mock_close}):
            result = process_sync_intraday(
                {"equity_tickers": ["AAPL"], "bond_tickers": ["IEF"], "data_lifetime": "14 days"},
                repo,
            )

        assert result["message"] == "Intraday sync complete"
        assert result["rows_inserted"] == len(tickers) * 8
        repo.save_intraday_market_data.assert_called_once()

    def test_exception_returns_error_dict(self):
        repo = _make_repo()
        with patch("handlers.command_handlers.yf.download", side_effect=ConnectionError("network")):
            result = process_sync_intraday(
                {"equity_tickers": [], "bond_tickers": [], "data_lifetime": "14 days"}, repo
            )
        assert "error" in result


# process_forecast

class TestProcessForecast:
    def test_valid_payload_calls_update_with_complete_status(self):
        repo = _make_repo()
        repo.get_historical_prices_tall.return_value = _make_tall_price_df(["SPY", "IEF"])

        np.random.seed(42)
        process_forecast(
            {
                "task_id": "task-001",
                "weights": {"SPY": 0.6, "IEF": 0.4},
                "years": 5,
                "initial_investment": 10_000.0,
                "monthly_contribution": 200.0,
                "verbose": False,
            },
            repo,
        )

        repo.update_forecast_status.assert_called_once()
        task_id, status, payload = repo.update_forecast_status.call_args[0]
        assert task_id == "task-001"
        assert status == "complete"

    def test_result_payload_has_required_keys(self):
        repo = _make_repo()
        repo.get_historical_prices_tall.return_value = _make_tall_price_df(["SPY", "IEF"])

        np.random.seed(0)
        process_forecast(
            {
                "task_id": "task-002",
                "weights": {"SPY": 0.6, "IEF": 0.4},
                "years": 3,
                "initial_investment": 5_000.0,
                "monthly_contribution": 0.0,
            },
            repo,
        )

        _, _, result_payload = repo.update_forecast_status.call_args[0]
        assert "pessimistic_5th_percentile" in result_payload
        assert "expected_50th_percentile" in result_payload
        assert "optimistic_95th_percentile" in result_payload
        assert len(result_payload["years"]) == 4  # 0..3

    def test_missing_task_id_returns_early_without_db_call(self):
        repo = _make_repo()
        process_forecast({}, repo)
        repo.update_forecast_status.assert_not_called()

    def test_empty_price_data_updates_error_status(self):
        repo = _make_repo()
        repo.get_historical_prices_tall.return_value = pd.DataFrame()

        process_forecast({"task_id": "task-003", "weights": {"SPY": 1.0}}, repo)

        repo.update_forecast_status.assert_called_once_with("task-003", "error")

    def test_forecast_percentile_ordering(self):
        repo = _make_repo()
        repo.get_historical_prices_tall.return_value = _make_tall_price_df(["SPY", "IEF"])

        np.random.seed(1)
        process_forecast(
            {
                "task_id": "task-004",
                "weights": {"SPY": 0.7, "IEF": 0.3},
                "years": 10,
                "initial_investment": 20_000.0,
                "monthly_contribution": 500.0,
            },
            repo,
        )

        _, _, result_payload = repo.update_forecast_status.call_args[0]
        pessimistic = result_payload["pessimistic_5th_percentile"]
        expected = result_payload["expected_50th_percentile"]
        optimistic = result_payload["optimistic_95th_percentile"]
        for i in range(len(pessimistic)):
            assert pessimistic[i] <= expected[i] <= optimistic[i], f"ordering violated at year {i}"
