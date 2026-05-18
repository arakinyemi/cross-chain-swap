package bitnob

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
}

func NewClient(clientID, clientSecret, baseURL string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) sign(payload string) (map[string]string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := hex.EncodeToString(nonceBytes)
	message := fmt.Sprintf("%s:%s:%s:%s", c.clientID, timestamp, nonce, payload)

	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	if _, err := mac.Write([]byte(message)); err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}

	return map[string]string{
		"X-Auth-Client":    c.clientID,
		"X-Auth-Timestamp": timestamp,
		"X-Auth-Nonce":     nonce,
		"X-Auth-Signature": hex.EncodeToString(mac.Sum(nil)),
	}, nil
}

func (c *Client) Do(method, path string, body any, out any) error {
	var payload string
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		payload = string(bodyBytes)
	}

	headers, err := c.sign(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s %s failed with status %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("unmarshal response body: %w", err)
	}

	return nil
}

func (c *Client) Whoami() error {
	var resp any
	if err := c.Do(http.MethodPost, "/api/whoami", nil, &resp); err != nil {
		return err
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		log.Printf("bitnob whoami response: %+v", resp)
		return nil
	}
	log.Printf("bitnob whoami response: %s", encoded)

	return nil
}
