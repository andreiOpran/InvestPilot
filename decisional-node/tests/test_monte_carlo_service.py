import numpy as np
from services.monte_carlo_service import run_monte_carlo


class TestRunMonteCarlo:
    def test_output_lengths_equal_years_plus_one(self):
        np.random.seed(0)
        p, e, o = run_monte_carlo(0.07, 0.15, 10_000, 0, 10)
        assert len(p) == 11
        assert len(e) == 11
        assert len(o) == 11

    def test_year_zero_is_initial_amount_for_all_percentiles(self):
        np.random.seed(0)
        p, e, o = run_monte_carlo(0.07, 0.15, 10_000, 0, 5)
        assert p[0] == 10_000.0
        assert e[0] == 10_000.0
        assert o[0] == 10_000.0

    def test_percentile_ordering_holds_each_year(self):
        np.random.seed(42)
        p, e, o = run_monte_carlo(0.07, 0.15, 10_000, 0, 10)
        for i in range(11):
            assert p[i] <= e[i] <= o[i], f"ordering violated at year {i}"

    def test_positive_mean_return_grows_expected_value(self):
        np.random.seed(42)
        _, e, _ = run_monte_carlo(0.07, 0.15, 10_000, 0, 10)
        assert e[-1] > 10_000.0

    def test_monthly_contribution_increases_final_value(self):
        np.random.seed(7)
        _, e_contrib, _ = run_monte_carlo(0.07, 0.15, 10_000, 500, 10)
        np.random.seed(7)
        _, e_none, _ = run_monte_carlo(0.07, 0.15, 10_000, 0, 10)
        assert e_contrib[-1] > e_none[-1]

    def test_zero_years_returns_single_element_list(self):
        p, e, o = run_monte_carlo(0.07, 0.15, 10_000, 0, 0)
        assert len(p) == 1
        assert p[0] == 10_000.0
        assert e[0] == 10_000.0
        assert o[0] == 10_000.0

    def test_deterministic_with_same_seed(self):
        np.random.seed(123)
        result1 = run_monte_carlo(0.07, 0.15, 10_000, 100, 5)
        np.random.seed(123)
        result2 = run_monte_carlo(0.07, 0.15, 10_000, 100, 5)
        assert result1 == result2

    def test_higher_volatility_produces_wider_spread(self):
        np.random.seed(42)
        p_low, _, o_low = run_monte_carlo(0.07, 0.05, 10_000, 0, 10)
        np.random.seed(42)
        p_high, _, o_high = run_monte_carlo(0.07, 0.30, 10_000, 0, 10)
        spread_low = o_low[-1] - p_low[-1]
        spread_high = o_high[-1] - p_high[-1]
        assert spread_high > spread_low

    def test_negative_mean_return_shrinks_expected_value(self):
        np.random.seed(42)
        _, e, _ = run_monte_carlo(-0.10, 0.05, 10_000, 0, 10)
        assert e[-1] < 10_000.0
