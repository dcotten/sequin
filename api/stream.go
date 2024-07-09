package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"sequin-cli/context"
)

// StreamsResponse represents the structure of the API response for a list
type StreamsResponse struct {
	Streams []Stream `json:"data"`
}

// Stream represents the structure of a stream returned by the API
type Stream struct {
	ID            string    `json:"id"`
	Idx           int       `json:"idx"`
	ConsumerCount int       `json:"consumer_count"`
	MessageCount  int       `json:"message_count"`
	CreatedAt     time.Time `json:"inserted_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FetchStreams retrieves all streams from the API
func FetchStreams(ctx *context.Context) ([]Stream, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(serverURL + "/api/streams")
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var streamsResponse StreamsResponse
	err = json.Unmarshal(body, &streamsResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return streamsResponse.Streams, nil
}

// FetchStreamInfo retrieves information for a specific stream from the API
func FetchStreamInfo(ctx *context.Context, streamID string) (*Stream, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(fmt.Sprintf("%s/api/streams/%s", serverURL, streamID))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var streamResponse Stream
	err = json.Unmarshal(body, &streamResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return &streamResponse, nil
}
