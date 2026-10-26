package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type HTTPNotifier struct {
	url    string
	client *http.Client
}

func NewHTTPNotifier(url string, client *http.Client) *HTTPNotifier {
	return &HTTPNotifier{url: url, client: client}
}

func (n *HTTPNotifier) Send(ctx context.Context, bookingID int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url,
		strings.NewReader(fmt.Sprintf(`{"booking_id":%d}`, bookingID)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
