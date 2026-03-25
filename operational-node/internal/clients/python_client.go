package clients

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
)

// SimulateInvestment calls the external Python node to generate models
// uses a custom http.Client with a timeout from configuration
func SimulateInvestment() ([]byte, error) {
	url := strings.TrimRight(config.Env.PythonNodeURL, "/") + "/generate-models"

	client := &http.Client{Timeout: config.Env.PythonClientTimeout}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error communicating with python-node: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read python-node response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("python-node returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
