package youtube

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	youtubeapi "google.golang.org/api/youtube/v3"
)

const userAgent = "EngineeringKiosk-awesome-software-engineering-movies"

// Client is a thin wrapper around the official YouTube Data API v3
// SDK. It exists so command code never imports the SDK directly and
// internal types (Video, Channel) stay decoupled from generated SDK
// types — the JSON we write to disk is then ours alone to evolve.
type Client struct {
	svc *youtubeapi.Service
}

// NewClient constructs a Client backed by the YouTube Data API v3
// service, authenticated via the given API key.
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("youtube: API key must not be empty")
	}

	svc, err := youtubeapi.NewService(ctx,
		option.WithAPIKey(apiKey),
		option.WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, fmt.Errorf("youtube: create service: %w", err)
	}

	return &Client{svc: svc}, nil
}
