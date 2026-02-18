from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/optimize', methods=['POST'])
def optimize_portofolio():
    mock_portfolio = {
        "status": "success",
        "allocation": {
            "AAPL": 0.40,
            "MSFT": 0.30,
            "BND": 0.30
        },
        "expected_return": 0.08,
        "risk": 0.12
    }
    
    return jsonify(mock_portfolio)

if __name__ == '__main__':
    # run on 0.0.0.0 to let connections from outside the container (go node)
    app.run(host='0.0.0.0', port=5000)