import yfinance as yf
from sqlalchemy import create_engine, text
import os
import pandas as pd
import numpy as np
import pika
import json
import logging
import time
from datetime import datetime, timezone
from scipy.cluster.hierarchy import linkage
from scipy.spatial.distance import squareform
from typing import Dict

import matplotlib
matplotlib.use('Agg')  # render images directly to file system
import matplotlib.pyplot as plt
import seaborn as sns
from scipy.cluster.hierarchy import dendrogram

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