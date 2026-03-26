package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client handles interaction with the Twitch API endpoints for authentication,
// subscriptions, and channel data lookups.
type Client struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
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

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", post)
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

// sendGET implements a standard authenticated GET request to Helix endpoints.
func (c *Client) sendGET(apiUrl string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("GET", apiUrl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", c.ClientId)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := http.DefaultClient.Do(req)
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
func (c *Client) sendPOST(apiUrl string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", c.ClientId)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := c.sendGET(apiUrl, nil)
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

	resp, err := c.sendGET(apiUrl, nil)
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
	resp, err := c.sendPOST(apiUrl, bytes.NewBuffer(data))
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

	req, err := http.NewRequest("DELETE", apiUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Client-Id", c.ClientId)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := http.DefaultClient.Do(req)
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
