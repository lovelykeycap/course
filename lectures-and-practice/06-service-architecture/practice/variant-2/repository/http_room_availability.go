package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type HTTPRoomAvailability struct {
	url    string
	client *http.Client
}

func NewHTTPRoomAvailability(url string, client *http.Client) *HTTPRoomAvailability {
	return &HTTPRoomAvailability{url: url, client: client}
}

func (a *HTTPRoomAvailability) Check(ctx context.Context, roomID int64, start, end string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url,
		strings.NewReader(fmt.Sprintf(`{"room_id":%d,"start_at":%q,"end_at":%q}`, roomID, start, end)))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}
