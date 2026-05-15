import numpy as np
import pandas as pd
from services.hrp_service import (
    _bisect_and_allocate_weights,
    _compute_cluster_variance,
    compute_hrp_weights,
)


def _make_returns(tickers, n_dates=150, seed=42):
    rng = np.random.default_rng(seed)
    data = {t: rng.normal(0, 0.01, n_dates) for t in tickers}
    return pd.DataFrame(data, index=pd.bdate_range("2020-01-01", periods=n_dates))


class TestComputeHrpWeights:
    def test_empty_returns_gives_empty_dict(self):
        result = compute_hrp_weights(pd.DataFrame())
        assert result == {}

    def test_two_assets_weights_sum_to_one(self):
        returns = _make_returns(["AAPL", "MSFT"])
        result = compute_hrp_weights(returns)
        assert isinstance(result, dict)
        assert set(result.keys()) == {"AAPL", "MSFT"}
        assert abs(sum(result.values()) - 1.0) < 1e-6

    def test_four_assets_weights_sum_to_one(self):
        returns = _make_returns(["SPY", "QQQ", "IEF", "GLD"])
        result = compute_hrp_weights(returns)
        assert abs(sum(result.values()) - 1.0) < 1e-6

    def test_all_weights_strictly_positive(self):
        returns = _make_returns(["SPY", "QQQ", "IEF", "GLD"])
        result = compute_hrp_weights(returns)
        assert all(w > 0 for w in result.values())

    def test_returns_exactly_the_input_tickers(self):
        tickers = ["SPY", "QQQ", "IEF", "GLD"]
        result = compute_hrp_weights(_make_returns(tickers))
        assert set(result.keys()) == set(tickers)

    def test_higher_volatility_asset_gets_lower_weight(self):
        # LOW_VOL std=0.005, HIGH_VOL std=0.025 -> HRP gives less weight to riskier asset
        rng = np.random.default_rng(0)
        n = 300
        returns = pd.DataFrame(
            {
                "LOW_VOL": rng.normal(0, 0.005, n),
                "HIGH_VOL": rng.normal(0, 0.025, n),
            },
            index=pd.bdate_range("2020-01-01", periods=n),
        )
        result = compute_hrp_weights(returns)
        assert result["LOW_VOL"] > result["HIGH_VOL"]

    def test_six_assets_weights_sum_to_one(self):
        tickers = ["SPY", "QQQ", "IEF", "GLD", "VNQ", "TLT"]
        returns = _make_returns(tickers, seed=7)
        result = compute_hrp_weights(returns)
        assert abs(sum(result.values()) - 1.0) < 1e-6
        assert len(result) == 6


class TestComputeClusterVariance:
    def test_returns_positive_float(self):
        cov = pd.DataFrame(
            [[0.04, 0.01], [0.01, 0.09]],
            index=["A", "B"],
            columns=["A", "B"],
        )
        var = _compute_cluster_variance(cov, ["A", "B"])
        assert isinstance(var, float)
        assert var > 0

    def test_single_asset_equals_its_own_variance(self):
        cov = pd.DataFrame([[0.04]], index=["A"], columns=["A"])
        var = _compute_cluster_variance(cov, ["A"])
        # IVP weights = [1.0], variance = 1.0^2 * 0.04 = 0.04
        assert abs(var - 0.04) < 1e-9

    def test_higher_covariance_yields_higher_cluster_variance(self):
        low_cov = pd.DataFrame(
            [[0.01, 0.000], [0.000, 0.01]],
            index=["A", "B"],
            columns=["A", "B"],
        )
        high_cov = pd.DataFrame(
            [[0.01, 0.009], [0.009, 0.01]],
            index=["A", "B"],
            columns=["A", "B"],
        )
        var_low = _compute_cluster_variance(low_cov, ["A", "B"])
        var_high = _compute_cluster_variance(high_cov, ["A", "B"])
        assert var_high > var_low


class TestBisectAndAllocateWeights:
    def test_two_assets_weights_sum_to_one(self):
        rng = np.random.default_rng(1)
        returns = pd.DataFrame(
            {"A": rng.normal(0, 0.01, 100), "B": rng.normal(0, 0.02, 100)},
            index=pd.bdate_range("2020-01-01", periods=100),
        )
        cov = returns.cov()
        budgets = _bisect_and_allocate_weights(cov, ["A", "B"])
        assert abs(budgets.sum() - 1.0) < 1e-9

    def test_four_assets_weights_sum_to_one(self):
        returns = _make_returns(["A", "B", "C", "D"])
        cov = returns.cov()
        ordered = ["A", "B", "C", "D"]
        budgets = _bisect_and_allocate_weights(cov, ordered)
        assert abs(budgets.sum() - 1.0) < 1e-9

    def test_all_weights_positive(self):
        returns = _make_returns(["A", "B", "C", "D"])
        cov = returns.cov()
        budgets = _bisect_and_allocate_weights(cov, ["A", "B", "C", "D"])
        assert all(w > 0 for w in budgets)
