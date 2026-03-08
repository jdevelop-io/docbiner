package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type apiClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newClient() (*apiClient, error) {
	key := getAPIKey()
	if key == "" {
		return nil, fmt.Errorf("no API key configured. Run 'docbiner auth login' or set DOCBINER_API_KEY")
	}

	return &apiClient{
		baseURL: getBaseURL(),
		apiKey:  key,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

func (c *apiClient) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "docbiner-cli/1.0")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (c *apiClient) get(path string) ([]byte, error) {
	resp, err := c.doRequest(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (c *apiClient) post(path string, body interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, path, bytes.NewReader(jsonData), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (c *apiClient) postBinary(path string, body interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, path, bytes.NewReader(jsonData), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errData, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(errData))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

func (c *apiClient) delete(path string) error {
	resp, err := c.doRequest(http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return nil
}
