package turnstile

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
)

type turnstileResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
}

// Verify token retrieved from fronted using Cloudflare API
func Verify(token string, remoteIP string) error {
	if config.Env.TurnstileSecretKey == "" {
		return nil
	}

	// format data for API call
	formData := url.Values{
		"secret":   {config.Env.TurnstileSecretKey},
		"response": {token},
		"remoteip": {remoteIP},
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", formData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tr turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return err
	}

	if !tr.Success {
		return ErrInvalidCaptcha
	}

	return nil
}
