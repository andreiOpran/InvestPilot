from services.rebalance_service import compute_rebalance


class TestComputeRebalance:
    def test_all_within_threshold_all_truly_skipped(self):
        current = {"AAPL": 0.5, "MSFT": 0.5}
        target = {"AAPL": 0.51, "MSFT": 0.49}
        final, truly_skipped = compute_rebalance(current, target, threshold=0.05, cash_first=False)
        assert abs(sum(final.values()) - 1.0) < 1e-3
        assert set(truly_skipped) == {"AAPL", "MSFT"}

    def test_all_beyond_threshold_renormalized_to_target(self):
        current = {"AAPL": 0.3, "MSFT": 0.7}
        target = {"AAPL": 0.6, "MSFT": 0.4}
        final, truly_skipped = compute_rebalance(current, target, threshold=0.05, cash_first=False)
        assert abs(final.get("AAPL", 0) - 0.6) < 1e-3
        assert abs(final.get("MSFT", 0) - 0.4) < 1e-3
        assert truly_skipped == []

    def test_cash_first_deploys_usd_to_underweights(self):
        current = {"AAPL": 0.3, "MSFT": 0.3, "USD": 0.4}
        target = {"AAPL": 0.5, "MSFT": 0.5}
        final, _ = compute_rebalance(current, target, threshold=0.02, cash_first=True)
        assert abs(final.get("AAPL", 0) - 0.5) < 1e-3
        assert abs(final.get("MSFT", 0) - 0.5) < 1e-3
        assert "USD" not in final

    def test_cash_first_false_ignores_usd(self):
        current = {"AAPL": 0.3, "MSFT": 0.3, "USD": 0.4}
        target = {"AAPL": 0.5, "MSFT": 0.5}
        final, truly_skipped = compute_rebalance(current, target, threshold=0.02, cash_first=False)
        assert "USD" not in final
        assert truly_skipped == []
        assert abs(sum(final.values()) - 1.0) < 1e-3

    def test_new_asset_not_in_current_gets_allocated(self):
        current = {"AAPL": 1.0}
        target = {"AAPL": 0.5, "MSFT": 0.5}
        final, _ = compute_rebalance(current, target, threshold=0.02, cash_first=False)
        assert "AAPL" in final
        assert "MSFT" in final
        assert abs(sum(final.values()) - 1.0) < 1e-3

    def test_weights_always_sum_to_one(self):
        current = {"AAPL": 0.25, "MSFT": 0.25, "SPY": 0.25, "IEF": 0.25}
        target = {"AAPL": 0.40, "MSFT": 0.10, "SPY": 0.30, "IEF": 0.20}
        final, _ = compute_rebalance(current, target, threshold=0.05, cash_first=False)
        assert abs(sum(final.values()) - 1.0) < 1e-3

    def test_usd_only_portfolio_deploys_all_cash(self):
        current = {"USD": 1.0}
        target = {"AAPL": 0.6, "MSFT": 0.4}
        final, _ = compute_rebalance(current, target, threshold=0.02, cash_first=True)
        assert "USD" not in final
        assert abs(final.get("AAPL", 0) - 0.6) < 1e-3
        assert abs(final.get("MSFT", 0) - 0.4) < 1e-3

    def test_cash_deployed_asset_not_truly_skipped(self):
        # cash moved AAPL from 0.3 -> 0.5, so final != original -> not truly skipped
        current = {"AAPL": 0.3, "MSFT": 0.3, "USD": 0.4}
        target = {"AAPL": 0.5, "MSFT": 0.5}
        _, truly_skipped = compute_rebalance(current, target, threshold=0.02, cash_first=True)
        assert "AAPL" not in truly_skipped
        assert "MSFT" not in truly_skipped

    def test_mixed_one_skipped_one_active(self):
        # AAPL at target -> threshold lock; MSFT far from target -> active pool
        current = {"AAPL": 0.5, "MSFT": 0.1}
        target = {"AAPL": 0.5, "MSFT": 0.5}
        final, truly_skipped = compute_rebalance(current, target, threshold=0.05, cash_first=False)
        assert "AAPL" in truly_skipped
        assert abs(final.get("AAPL", 0) - 0.5) < 1e-3
        assert abs(sum(final.values()) - 1.0) < 1e-3

    def test_zero_weight_assets_excluded_from_final(self):
        current = {"AAPL": 0.5, "MSFT": 0.5}
        # NVDA not in current and target weight 0 -> should not appear
        target = {"AAPL": 0.7, "MSFT": 0.3}
        final, _ = compute_rebalance(current, target, threshold=0.02, cash_first=False)
        assert all(v > 0 for v in final.values())
