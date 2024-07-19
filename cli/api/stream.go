package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bytes"

	"github.com/sequinstream/sequin/cli/context"
)

// StreamsResponse represents the structure of the API response for a list
type StreamsResponse struct {
	Streams []Stream `json:"data"`
}

// Stream represents the structure of a stream returned by the API
type Stream struct {
	ID    string `json:"id"`
	Idx   int    `json:"idx"`
	Name  string `json:"name"`
	Stats struct {
		ConsumerCount int `json:"consumer_count"`
		MessageCount  int `json:"message_count"`
		StorageSize   int `json:"storage_size"`
	} `json:"stats"`
	CreatedAt time.Time `json:"inserted_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BuildFetchStreams builds the HTTP request for fetching streams
func BuildFetchStreams(ctx *context.Context) (*http.Request, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", serverURL+"/api/streams", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// FetchStreams retrieves all streams from the API
func FetchStreams(ctx *context.Context) ([]Stream, error) {
	req, err := BuildFetchStreams(ctx)
	if err != nil {
		return nil, fmt.Errorf("error building fetch streams request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
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

// BuildFetchStreamInfo builds the HTTP request for fetching a specific stream's information
func BuildFetchStreamInfo(ctx *context.Context, streamID string) (*http.Request, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/streams/%s", serverURL, streamID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// FetchStreamInfo retrieves information for a specific stream from the API
func FetchStreamInfo(ctx *context.Context, streamID string) (*Stream, error) {
	req, err := BuildFetchStreamInfo(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("error building fetch stream info request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
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

// BuildAddStream builds the HTTP request for adding a new stream
func BuildAddStream(ctx *context.Context, name string) (*http.Request, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]string{"name": name}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	req, err := http.NewRequest("POST", serverURL+"/api/streams", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// AddStream adds a new stream with the given name
func AddStream(ctx *context.Context, name string) (*Stream, error) {
	req, err := BuildAddStream(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("error building add stream request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnprocessableEntity {
			var errorResponse struct {
				Summary          string              `json:"summary"`
				ValidationErrors map[string][]string `json:"validation_errors"`
			}
			if err := json.Unmarshal(body, &errorResponse); err == nil {
				for field, errors := range errorResponse.ValidationErrors {
					for _, errMsg := range errors {
						fmt.Printf("`%s` %s\n", field, errMsg)
					}
				}
				return nil, fmt.Errorf("validation failed")
			}
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var stream Stream
	err = json.Unmarshal(body, &stream)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return &stream, nil
}

// BuildRemoveStream builds the HTTP request for removing a stream
func BuildRemoveStream(ctx *context.Context, streamID string) (*http.Request, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/streams/%s", serverURL, streamID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	return req, nil
}

// RemoveStream removes a stream with the given ID
func RemoveStream(ctx *context.Context, streamID string) error {
	req, err := BuildRemoveStream(ctx, streamID)
	if err != nil {
		return fmt.Errorf("error building remove stream request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// BuildPublishMessage builds the HTTP request for publishing a message to a stream
func BuildPublishMessage(ctx *context.Context, streamID, subject, message string) (*http.Request, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/streams/%s/messages", serverURL, streamID)
	payload := map[string]interface{}{
		"messages": []map[string]string{
			{
				"subject": subject,
				"data":    message,
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// PublishMessage publishes a message to a stream
func PublishMessage(ctx *context.Context, streamID, subject, message string) error {
	req, err := BuildPublishMessage(ctx, streamID, subject, message)
	if err != nil {
		return fmt.Errorf("error building publish message request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

type MessagesResponse struct {
	Messages []Message `json:"data"`
}

// BuildListStreamMessages builds the HTTP request for listing stream messages
func BuildListStreamMessages(ctx *context.Context, streamIDOrName string, limit int, sort string, subjectPattern string) (*http.Request, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/streams/%s/messages?limit=%d&sort=%s", serverURL, streamIDOrName, limit, sort)
	if subjectPattern != "" {
		url += "&subject_pattern=" + subjectPattern
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	return req, nil
}

// ListStreamMessages retrieves messages from a stream
func ListStreamMessages(ctx *context.Context, streamIDOrName string, limit int, sort string, subjectPattern string) ([]Message, error) {
	req, err := BuildListStreamMessages(ctx, streamIDOrName, limit, sort, subjectPattern)
	if err != nil {
		return nil, fmt.Errorf("error building list stream messages request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var messagesResponse MessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&messagesResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return messagesResponse.Messages, nil
}

// GetStreamMessage retrieves a specific message from a stream by its key
func GetStreamMessage(ctx *context.Context, streamIDOrName, key string) (Message, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return Message{}, err
	}

	url := fmt.Sprintf("%s/api/streams/%s/messages/%s", serverURL, streamIDOrName, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Message{}, fmt.Errorf("error creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var message Message
	err = json.NewDecoder(resp.Body).Decode(&message)
	if err != nil {
		return Message{}, fmt.Errorf("error decoding JSON: %w", err)
	}

	return message, nil
}

func FetchMessageDetail(ctx *context.Context, streamID, messageKey string) (*MessageDetail, error) {
	serverURL, err := context.GetServerURL(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/streams/%s/messages/%s/consumer_info", serverURL, streamID, messageKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data MessageDetail `json:"data"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// MessageDetail represents the detailed structure of a message
type MessageDetail struct {
	Message       Message        `json:"message"`
	ConsumerInfos []ConsumerInfo `json:"consumer_info"`
}

// ConsumerInfo represents the structure of consumer information
type ConsumerInfo struct {
	State                        string     `json:"state"`
	NotVisibleUntil              *time.Time `json:"not_visible_until"`
	ConsumerID                   string     `json:"consumer_id"`
	AckID                        *string    `json:"ack_id"`
	DeliverCount                 *int       `json:"deliver_count"`
	LastDeliveredAt              *time.Time `json:"last_delivered_at"`
	ConsumerFilterSubjectPattern string     `json:"consumer_filter_subject_pattern"`
	ConsumerName                 string     `json:"consumer_name"`
}
