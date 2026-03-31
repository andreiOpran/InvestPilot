from fastapi import FastAPI
from pydantic import BaseModel
import yfinance as yf
from sqlalchemy import create_engine, text
import os
import pandas as pd
import numpy as np
from scipy.cluster.hierarchy import linkage
from scipy.spatial.distance import squareform
from typing import Dict

import matplotlib
matplotlib.use('Agg')  # render images directly to file system
import matplotlib.pyplot as plt
import seaborn as sns
from scipy.cluster.hierarchy import dendrogram

app = FastAPI(title="Robo-Advisory Decisional Node", version="1.0")

DATABASE_URL = os.environ.get("DATABASE_URL", "postgresql://admin:pass@db:5432/robo_advisory")
engine = create_engine(DATABASE_URL)  # connection pool for the db

# TODO: in production these asset universes should be send via RabbitMQ message
# payload from Go

# growth assets (higher risk, higher expected return over long horizons)
EQUITY_TICKERS = [
    "VTI", "VOO", "QQQ", "VTV", "VUG", "IWM", # US broad market equities
    "VEA", "VWO",                             # International equities
    "VNQ", "VNQI",                            # Real estate (REITs)
    "XLF", "XLV", "XLE", "XLK"                # US sector ETFs
]

# safety assets (lower risk to act as a portfolio stabilizer)
BOND_TICKERS = [
    "BND",  # Total US bond market
    "TLT",  # long-term US government bonds (>20 years)
    "LQD",  # Investment-grade corporate bonds
    "HYG",  # High-yield (junk) corporate bonds (higher risk within bonds)
    "BNDX"  # International bonds
]

# combine all assets in a single var, to streamline yfinance download call
ALL_TICKERS = EQUITY_TICKERS + BOND_TICKERS

# ForecastRequest represents what the operational-node sends for a forecast
class ForecastRequest(BaseModel):
    weights: Dict[str, float]    # e.g. {"VTI": 0.42, "BND": 0.30, ...}
    initial_investment: float    # starting portfolio value in dollars
    monthly_contribution: float  # dollars added per month
    years: int                   # forecast horizon


def save_debug_csv(data, filename: str):
    """Saves any common data structure to a CSV file in the debug_output/ folder."""
    out_dir = "debug_output"
    os.makedirs(out_dir, exist_ok=True)
    filepath = os.path.join(out_dir, filename)
    
    if isinstance(data, pd.DataFrame):
        data.to_csv(filepath)
    elif isinstance(data, pd.Series):
        data.to_csv(filepath, header=True)
    elif isinstance(data, np.ndarray):
        np.savetxt(filepath, data, delimiter=',', fmt='%f')
    elif isinstance(data, list):
        with open(filepath, 'w') as f:
            for item in data:
                f.write(f"{item}\n")
    else:
        with open(filepath, 'w') as f:
            f.write(str(data))
    
    print(f"[DEBUG] Saved: {filepath}")
    
def plot_dendrogram_chart(links, tickers, filename, title="HRP Dendrogram"):
    """
    Renders the hierarchical clustering tree (dendrogram).
    Each leaf is an ETF. Branches show which assets were grouped together
    and at what correlation distance. Assets that are close on the tree
    move similarly in the market.
    """
    out_dir = "debug_output"
    os.makedirs(out_dir, exist_ok=True)
    filepath = os.path.join(out_dir, filename)
    
    plt.figure(figsize=(12, 6))
    dendrogram(links, labels=tickers, leaf_rotation=90, leaf_font_size=12)
    plt.title(title)
    plt.ylabel("Distance (inverse of correlation)")
    plt.tight_layout()

    plt.savefig(filepath, dpi=300)
    plt.close()  # release memory
    print(f"[DEBUG] Chart saved: {filepath}")

def plot_heatmap_chart(matrix, tickers, filename, title):
    """
    Renders a heatmap of a covariance or correlation matrix.
    Bright cells = high value (assets move together strongly).
    Dark cells = low value (assets are unrelated or move opposite).
    """
    out_dir = "debug_output"
    os.makedirs(out_dir, exist_ok=True)
    filepath = os.path.join(out_dir, filename)
    
    if matrix.empty or len(tickers) == 0:
        print(f"[!] Matrix is empty, skipping heatmap: {filename}")
        return

    plt.figure(figsize=(10, 8))
    sns.heatmap(matrix, xticklabels=tickers, yticklabels=tickers, cmap="viridis")
    plt.title(title)
    plt.tight_layout()

    plt.savefig(filepath, dpi=300)
    plt.close()
    print(f"[DEBUG] Chart saved: {filepath}")

def reorder_assets_by_cluster_similarity(linkage_matrix):
    """
    HRP STEP 1 OF 3: REORDER ASSET LIST BY SIMILARITY (QUASI-DIAGONALIZATION)
    
    Reorder the asset list so that similar assets are adjacent
    Input: linkage matrix returned by scipy.linkage()
    Output: list of original asset indices in the sorted order
    
    linkage_matrix shape is (N-1) x 4, where each row is a merge event:
        col 0: index of left cluster merged
        col 1: index of right cluster merged
        col 2: distance at which they merged
        col 3: total number of original assets in the new merged cluster
        
    Original asset indices are 0..N-1
    Merged cluster indices start at N
    """
    # we cast to int to avoid float indices errors
    # (we lose the distance col, however we don't use it)
    linkage_matrix = linkage_matrix.astype(int)
    
    # start with the two root clusters from the last merge
    # col 0 is left child, col 1 is right child
    cluster_order = pd.Series([linkage_matrix[-1, 0], linkage_matrix[-1, 1]])
    
    # total number of original assets = col 3 of the last row
    # any index >= num_assets is a merged cluster (non-leaf) 
    # that still needs to be expanded into its childer
    num_assets = linkage_matrix[-1, 3]
    
    # expand non-leaf clusters until only original indices remain (leafs)
    while cluster_order.max() >= num_assets:
        # re-index to even numbers to make room for inserting
        # right children at odd positions between existing entries
        cluster_order.index = range(0, cluster_order.shape[0] * 2, 2)
        
        # find non-leaf entries (index >= N)
        non_leaf_clusters = cluster_order[cluster_order >= num_assets]
        
        # where in cluster_order we have non-leaf entries
        positions_in_series = non_leaf_clusters.index
        
        # which row of the linkage matrix describes this cluster:
        # cluster N is in row 0, cluster N+1 is in row 1
        row_in_linkage = non_leaf_clusters.values - num_assets
        
        # replace non-leaf cluster with its left child (col 0 if its linkage row)
        cluster_order[positions_in_series] = linkage_matrix[row_in_linkage, 0]
        
        # build a series of right children (col 1) placed at the odd positions
        # after each left child
        right_children = pd.Series(
            linkage_matrix[row_in_linkage, 1],
            index=positions_in_series + 1  # one slot after each left child
        )
        
        # merge left children (already in cluster_order) with right children
        # then sort by index so left/right interleave correctly
        cluster_order = pd.concat([cluster_order, right_children])
        cluster_order = cluster_order.sort_index()
        
    # we have only asset index leaves (0..N-1) so we cast to list and return
    return cluster_order.toList()
        

def compute_cluster_variance(cov_matrix, cluster_tickers):
    """
    HRP STEP 2 OF 3: CLUSTER VARIANCE
    
    Compute the variance of a mini Inverse Variance Portfolio built from
    only the assets in one cluster. This is used to compare risk between
    left and right halves during bisection.
    
    "Variance" = statistical measure of how much the cluster's daily
    returns fluctuates -> higher variance, higher risk
    """
    # extract sub-matrix that contains our cluster
    cluster_cov = cov_matrix.loc[cluster_tickers, cluster_tickers]
    
    # diagonal of matrix holds each asset's variance
    # we get the invers variance weight, where less volatile
    # assets get higher weight, more volatile get lower weight
    inverse_variance_weights = 1.0 / np.diag(cluster_cov)
    
    # normalize so all weights sum to 1.0
    inverse_variance_weights /= inverse_variance_weights.sum()
    
    # compute portfolio variance using: w' * Σ * w
    # where w = weights vector, Σ = covariance matrix
    # this accounts for individual asset variances and their co-movements
    cluster_variance = np.dot(
        inverse_variance_weights.T,
        np.dot(cluster_cov, inverse_variance_weights)
    )
    return cluster_variance

def compute_hrp_weights(returns, verbose: bool = False, prefix: str = ""):
    """
    HRP STEP 3 OF 3: FULL HRP ALGORITHM
    
    Compute weights via Hierarchical Risk Parity algorithm on a returns DataFrame
    Returns a dict of {ticker: weight} where all weights sum to 1.0.
    
    Combines all three phases:
        Phase A: build covariance and correlation matrices
        Phase B: cluster assets by similarity (via distance matrix + linkage)
        Phase C: bisect the tree and allocate weights to evenly distribute risk
        
    "prefix" is used to avoid overwriting debug files when function is called twice
    """
    if returns.empty:
        print(f"[!] Warning: empty returns DataFrame passed to HRP ({prefix}).")
        return {}
    
    # PHASE A: STATISTICAL BASE
    # measures how much eack pair of assets move together
    # diagonal = each asset's own variance (volatility squared)
    # off-diagonal = how asset A and B co-move (positive = same direction)
    cov_matrix = returns.cov()
    
    # covariance normalized to [-1, +1]
    # +1 = always move together, -1 = always move opposite, 0 = unrelated
    # we use correlation and not covariance for clustering because it's
    # scale-independent (refering to a ETF's value)
    corr_matrix = returns.corr()
    
    # used in debug diagrams
    tickers_original = returns.columns.tolist()
    
    if verbose:
        save_debug_csv(cov_matrix, f"{prefix}_2_covariance_matrix.csv")
        save_debug_csv(corr_matrix, f"{prefix}_3_correlation_matrix.csv")
        plot_heatmap_chart(
            cov_matrix, 
            tickers_original, 
            f"{prefix}_chart_1_heatmap_before_reordering.png",
            f"Covariance before reordering asset list ({prefix.strip('_').upper()})"
        )
        
    # PHASE B: BUILDING THE CLUSTER TREE
    # we can't use correlations directly as distance
    # corr=+1 needs to be mapped to distance=0
    # corr=-1 needs to be mapped to distance=1
    # we use the formula distace = sqrt(0.5 * (1 - corr))
    distance_matrix = np.sqrt(0.5 * (1 - corr_matrix))
    
    if verbose:
        save_debug_csv(distance_matrix, f"{prefix}_4_distance_matrix.csv")
        
    # linkage builds hierarchical cluster tree by linking two closest
    # clusters of assets until we have one big cluster
    # squareform() casts the matrix into a 1D format of the upper triangle
    # method="single" chaining assets that are highly correlated together
    # even if some cluster members are less correlated with each other
    cluster_tree = linkage(squareform(distance_matrix), method="single")
    
    if verbose:
        save_debug_csv(cluster_tree, f"{prefix}_5_linkage_tree.csv")
        plot_dendrogram_chart(
            cluster_tree,
            tickers_original,
            f"{prefix}_chart_2_dendrogram.png",
            title=f"HRP Dendrogram ({prefix.strip('_').upper()})"
        )

    # traverse cluster tree to reorder the assets so that similar
    # ones are adjacent
    sorted_asset_indices = reorder_assets_by_cluster_similarity(cluster_tree)
    
    # map numeric indices back to ticker names
    ordered_tickers = returns.colums[sorted_asset_indices].tolist()
    
    if verbose:
        save_debug_csv(ordered_tickers, f"{prefix}_6_sorted_tickers_list.csv")
        # arrange the covariance matrix rows and columns using the previously computed order
        # we will see the high-covariance values cluster along the diagonal
        cov_sorted = cov_matrix.iloc[sorted_asset_indices, sorted_asset_indices]
        plot_heatmap_chart(
            cov_sorted,
            ordered_tickers,
            f"{prefix}_chart_3_heatmap_after_sort.png",
            f"Covariance after reordering asset list ({prefix.strip('_').upper()})"
        )
        
    # PHASE C: RECURSIVE BISECTION TO ALLOCATE WEIGHTS
    # all weights are initialized to 1.0, and will get multiplied
    # by a fraction at each level of bisection
    weight_budgets = pd.Series(1.0, index=ordered_tickers)
    
    # start with all assets
    current_clusters = [ordered_tickers]
    
    while len(current_clusters) > 0:
        # for each cluster we produce 2 sub-clusters (left and right half)
        # clusters with only 1 asset means it is final
        current_clusters = [
            half 
            for cluster in current_clusters
            for half in (cluster[:len(cluster) // 2], cluster[len(cluster) // 2:])
            if len(cluster) > 1
        ]
        
        # process pairs of clusters (left, right) that came from the same bisection
        for pair_index in range(0, len(current_clusters), 2):
            left_cluster  = current_clusters[pair_index]
            right_cluster = current_clusters[pair_index + 1]
            
            # compute Inverse Variance Portfolio for each half
            # we get how risky is the left group vs right group
            var_left  = compute_cluster_variance(cov_matrix, left_cluster)
            var_right = compute_cluster_variance(cov_matrix, right_cluster)
            
            # alpha = fraction of current budget allocated to left cluster
            # (riskier clusters receive less weight)
            # var_left == var_right -> alpha = 0.5 (equal split)
            # var_left  > var_right -> alpha < 0.5 (less to riskier left)
            # var_left  < var_right -> alpha > 0.5 (more to safer left)
            # goal: evenly distribute risk across portfolio
            alpha = 1.0 - (var_left / (var_left + var_right))
            weight_budgets[left_cluster]  *= alpha
            weight_budgets[right_cluster] *= (1.0 - alpha)
            
    if verbose:
        save_debug_csv(weight_budgets, f"{prefix}_7_hrp_final_weights.csv")
        
    # budgets sum to 1.0 because each bisection splits
    # the budget without creating or losing weight
    return weight_budgets.to_dict()

def run_monte_carlo(mean_return, volatility, initial_amount, monthly_contrib, years, num_simulations=10000, verbose=False):
    pass

@app.post('/sync')
def sync():
    pass

@app.post('/generate-models')
def compute_and_store_model_portfolios(verbose: bool = False):
    pass

@app.post('/forecast')
def run_portfolio_forecast(req: ForecastRequest, verbose: bool = False):
    pass

if __name__ == '__main__':
    import uvicorn
    uvicorn.run(app, host='0.0.0.0', port=5000)
