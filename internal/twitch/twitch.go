package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client handles interaction with the Twitch API endpoints for authentication,
// subscriptions, and channel data lookups.
type Client struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
	httpClient   *http.Client
}

// authResponse represents a response from the Twitch OAuth endpoint.
type authResponse struct {
	AccessToken string `json:"access_token"`
}

// StreamInfo holds public stream status information.
type StreamInfo struct {
	Title       string `json:"title"`
	ViewerCount int    `json:"viewer_count"`
	Type        string `json:"type"`
}

// StreamResponse holds multiple stream records from streams API.
type StreamResponse struct {
	Data []StreamInfo `json:"data"`
}

// UserInfo represents limited internal Twitch user details.
type UserInfo struct {
	Id string `json:"id"`
}

// UserResponse holds multiple user records from users API.
type UserResponse struct {
	Data []UserInfo `json:"data"`
}

// EventSubCondition identifies the stream target (broadcaster ID) for an EventSub subscription.
type EventSubCondition struct {
	BroadcasterUserId string `json:"broadcaster_user_id"`
}

// EventSubTransport describes the webhook details to be verified and executed.
type EventSubTransport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
	Secret   string `json:"secret"`
}

// EventSubSubscription is the payload shape sent to Twitch for creating an EventSub webhook.
type EventSubSubscription struct {
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Condition EventSubCondition `json:"condition"`
	Transport EventSubTransport `json:"transport"`
}

// NewClient returns an initialized Twitch API client that requires subsequent Auth().
func NewClient(clientId string, clientSecret string) *Client {
	return &Client{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Auth exchanges the configured ClientId and ClientSecret for an app access token (Client Credentials flow).
// This access token is then saved globally in the Client.
func (c *Client) Auth() error {
	post := url.Values{
		"client_id":     {c.ClientId},
		"client_secret": {c.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	resp, err := c.httpClient.PostForm("https://id.twitch.tv/oauth2/token", post)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"twitch api error: Auth: status %d, body %s",
			resp.StatusCode,
			string(errBody))
	}

	var buff authResponse
	err = json.NewDecoder(resp.Body).Decode(&buff)
	if err != nil {
		return err
	}
	c.AccessToken = buff.AccessToken

	return nil
}

// doRequest wraps http.Client.Do, automatically injecting auth headers
// and retrying exactly once if a 401 Unauthorized is returned (token expired).
func (c *Client) doRequest(method string, apiUrl string, payload []byte) (*http.Response, error) {
	req, err := c.createRequest(method, apiUrl, payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		err = c.Auth()
		if err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}

		req, err = c.createRequest(method, apiUrl, payload)
		if err != nil {
			return nil, err
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// createRequest is a helper that prepares the http.Request with JSON payload and auth headers.
func (c *Client) createRequest(method string, apiUrl string, payload []byte) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewBuffer(payload)
	}

	req, err := http.NewRequest(method, apiUrl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", c.ClientId)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	if payload != nil || method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// sendGET implements a standard authenticated GET request to Helix endpoints.
func (c *Client) sendGET(apiUrl string) (*http.Response, error) {
	resp, err := c.doRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"twitch api error: status %d, body %s",
			resp.StatusCode,
			string(errBody),
		)
	}

	return resp, nil
}

// sendPOST implements a standard authenticated POST request to Helix endpoints.
func (c *Client) sendPOST(apiUrl string, payload []byte) (*http.Response, error) {
	resp, err := c.doRequest("POST", apiUrl, payload)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"twitch api error: status %d, body %s",
			resp.StatusCode,
			string(errBody),
		)
	}

	return resp, nil
}

// GetStreamInfo calls Helix to see if a channel is currently broadcasting.
// Returns true if stream is live.
func (c *Client) GetStreamInfo(channelName string) (bool, error) {
	apiUrl := "https://api.twitch.tv/helix/streams?user_login=" + channelName

	resp, err := c.sendGET(apiUrl)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var streamResponse StreamResponse
	err = json.NewDecoder(resp.Body).Decode(&streamResponse)
	if err != nil {
		return false, err
	}

	if len(streamResponse.Data) > 0 {
		return true, nil
	}
	return false, nil
}

// GetUserId handles resolving a human-readable broadcaster login into its internal ID.
func (c *Client) GetUserId(channelName string) (string, error) {
	apiUrl := "https://api.twitch.tv/helix/users?login=" + channelName

	resp, err := c.sendGET(apiUrl)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var userResponse UserResponse
	err = json.NewDecoder(resp.Body).Decode(&userResponse)
	if err != nil {
		return "", err
	}

	if len(userResponse.Data) > 0 {
		return userResponse.Data[0].Id, nil
	}
	return "", fmt.Errorf("user not found")
}

// SubscribeToStream issues an HTTP call to the EventSub service instructing it
// to POST payloads back to our specific webhook listener for `stream.online` event type.
// Returns the registered subscription UUID on success.
func (c *Client) SubscribeToStream(
	userId string,
	callbackUrl string,
	secret string) (string, error) {
	event := EventSubSubscription{
		Type:    "stream.online",
		Version: "1",
		Condition: EventSubCondition{
			BroadcasterUserId: userId,
		},
		Transport: EventSubTransport{
			Method:   "webhook",
			Callback: callbackUrl,
			Secret:   secret,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	apiUrl := "https://api.twitch.tv/helix/eventsub/subscriptions"
	resp, err := c.sendPOST(apiUrl, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	if len(result.Data) > 0 {
		return result.Data[0].ID, nil
	}
	return "", nil
}

// DeleteSubscription removes an active EventSub tracking by its ID, ceasing any future pings.
func (c *Client) DeleteSubscription(eventSubID string) error {
	apiUrl := "https://api.twitch.tv/helix/eventsub/subscriptions?id=" + eventSubID

	resp, err := c.doRequest("DELETE", apiUrl, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch api error: delete subscription: status %d, body %s",
			resp.StatusCode, string(errBody))
	}
	return nil
}
