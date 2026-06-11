// Package imagegen generates images from text prompts via the Hugging Face Inference API.
// Requires a free Hugging Face API token in the HF_TOKEN environment variable.
// Get one at https://huggingface.co/settings/tokens (read access is sufficient).
package imagegen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const hfModel = "black-forest-labs/FLUX.1-schnell"

// Generate calls the Hugging Face Inference API to generate an image from prompt.
// apiToken is a Hugging Face API token.
// Returns raw image bytes on success.
func Generate(ctx context.Context, prompt, apiToken string) ([]byte, error) {
	reqBody, _ := json.Marshal(map[string]any{"inputs": prompt})
	url := "https://router.huggingface.co/hf-inference/models/" + hfModel

	// Retry once on cold start (503 = model loading).
	for attempt := 0; attempt < 2; attempt++ {
		data, loading, err := doRequest(ctx, url, apiToken, reqBody)
		if err != nil {
			return nil, err
		}
		if !loading {
			return data, nil
		}
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("model is still loading; try again in a moment")
}

func doRequest(ctx context.Context, url, token string, body []byte) (data []byte, loading bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("calling Hugging Face API: %w", err)
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, true, nil // model is loading, caller should retry
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("Hugging Face API returned %d: %s", resp.StatusCode, data)
	}
	return data, false, nil
}
