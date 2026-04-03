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
from utils.debug import save_debug_csv, plot_dendrogram_chart, plot_heatmap_chart

def run_monte_carlo(
    mean_return,
    volatility,
    initial_amount,
    monthly_contrib,
    years,
    num_simulations=10000,
    verbose=False
):
    """
    MONTE CARLO SIMULATION: runs scenarios of portfolio growth
    
    Each scenario applies a randomly drawn annual return from
    a normal distribution parameterized by historical mean and
    volatility, plus an annual contribution, compounding year by year
    
    Returns 5th, 50th and 95th percentile outcomes per year
    """
    # create 2D array of zeros, rows = years, columns = simulations
    simulation_grid = np.zeros((years + 1, num_simulations))
    
    # year 0, all simulations start with same initial investment amount
    simulation_grid[0] = initial_amount
    
    # convert monthly contribution to annual
    annual_contrib = monthly_contrib * 12
    
    for year in range(1, years + 1):
        # draw one random annual return per simulation from a normal distribution
        random_annual_returns = np.random.normal(
            loc=mean_return,     # center of distribuition = historical expected annual return
            scale=volatility,    # std deviation = historical annual volatility
            size=num_simulations # number of values to generate = one per simulation
        )
            
        # apply Geometric Brownian Motion (grow last year's vaue by this
        # year's return, then add annual contribution)
        simulation_grid[year] = (
            simulation_grid[year - 1] * (1 + random_annual_returns) + annual_contrib
        )
        
    if verbose:
        save_debug_csv(simulation_grid, "mc_1_all_simulations.csv")
        
    # for each year (row), compute various percentiles across all sim values (columns)
    # axis=1 means one percentile value per row (year)
    pessimistic = np.percentile(simulation_grid, 5,  axis=1)
    expected    = np.percentile(simulation_grid, 50, axis=1)
    optimistic  = np.percentile(simulation_grid, 95, axis=1)
    
    if verbose:
        summary = pd.DataFrame({
            "Year": range(years + 1),
            "Pessimistic_5th":  pessimistic,
            "Expected_50th":    expected,
            "Optimistic_95th":  optimistic
        })
        save_debug_csv(summary, "mc_2_final_percentiles.csv")
        
    return pessimistic.tolist(), expected.tolist(), optimistic.tolist()

