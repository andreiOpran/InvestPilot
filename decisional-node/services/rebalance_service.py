def compute_rebalance(current_allocation: dict, target_weights: dict, threshold: float, cash_first: bool):
    """
    Computes an adjusted portfolio to be as close
    as possible to the model portfolio while minimizing
    buy/sell orders to avoid "taxes"
    
    Steps:
    1. Cash-first rule: deploy available USD to the most underweight assets first
    2. Threshold filtering: after cash deployment, if an asset's drift is within
    the threshold, its weight is locked to avoid generating taxable sell events
    3. Renormalization: distributes the remaining portfolio space across the
    unlocked assets so that final weights sum to 1.0
    """
    # normalize inputs
    all_tickers = set(current_allocation.keys()).union(target_weights.keys())
    all_tickers.discard("USD")  # USD is funding, not a target asset
    
    usd_available = current_allocation.get("USD", 0.0)
    
    # 1. simulated cash deployment
    # create simulated reality where we've spent all our USD buying the underweights
    simulated_current = {k: current_allocation.get(k, 0.0) for k in all_tickers}
    
    if cash_first and usd_available > 0.0001:
        deficits = {}
        for t in all_tickers:
            target = target_weights.get(t, 0.0)
            current = simulated_current[t]
            if current < target:
                deficits[t] = target - current
                
        total_deficit = sum(deficits.values())
        
        if total_deficit > 0:
            # disttribute USD proportionally to how much each asset is starving
            for t, deficit in deficits.items():
                allocation_ratio = deficit / total_deficit
                cash_to_deploy = min(usd_available, total_deficit) * allocation_ratio
                simulated_current[t] += cash_to_deploy
                
    # 2. threshold filter and locking
    adjusted_targets = {}
    skipped = []
    remaining_target_space = 1.0
    active_target_pool = 0.0
    
    for ticker in all_tickers:
        # compare target agains simulated current (after cash injection)
        current_sim = simulated_current[ticker]
        target = target_weights.get(ticker, 0.0)
        
        if abs(target - current_sim) <= threshold:
            # asset is close enough, lock it at its simulated weight
            adjusted_targets[ticker] = current_sim
            remaining_target_space -= current_sim
            skipped.append(ticker)
        else:
            # asset drifted too far, it goes into active pool to be rebalanced
            active_target_pool += target
            
    # 3. renormalization
    if active_target_pool > 0:
        for ticker in all_tickers:
            if ticker not in skipped:
                original_target = target_weights.get(ticker, 0.0)
                # scale target to fit exactly in the remaining percentage space
                normalized = (original_target / active_target_pool) * remaining_target_space
                adjusted_targets[ticker] = normalized
    else:
        # all assets were within the threshold, but the user deposited massive amounts
        # of cash that didn't get fully deployed, so we distribute remaining space
        if remaining_target_space > 0.0001:
            for ticker in skipped:
                target = target_weights.get(ticker, 0.0)
                # fallback to pure target proportionality
                adjusted_targets[ticker] += remaining_target_space * target
                
    # 4. cleanup
    final_targets = {k: round(v, 4) for k, v in adjusted_targets.items() if round(v, 4) > 0}
    
    # return tickers as skipped if thir final target
    # exactly matches their original current_allocation
    truly_skipped = []
    for ticker in skipped:
        original = round(current_allocation.get(ticker, 0.0), 4)
        if final_targets.get(ticker) == original:
            truly_skipped.append(ticker)
            
    return final_targets, truly_skipped
