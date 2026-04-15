package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// generate:reset
// HTTPProvider реализует провайдера аудита через HTTP.
type HTTPProvider struct {
	client   *http.Client
	endpoint string
}

// NewHTTPProvider создаёт и возвращает нового AuditHTTPProvider.
func NewHTTPProvider(endpoint string) *HTTPProvider {
	return &HTTPProvider{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		endpoint: endpoint,
	}
}

func (p *HTTPProvider) Send(
	ctx context.Context,
	actionType ActionType,
	userID *string,
	url string,
) error {
	payload := Action{
		Timestamp:  time.Now().Unix(),
		ActionType: actionType,
		UserID:     userID,
		URL:        url,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to complete the request: %w", err)
	}
	defer response.Body.Close()

	return nil
}

func (p *HTTPProvider) Close() error {
	return nil
}
