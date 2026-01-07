package netbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewClient(url, token string) *Client {
	return &Client{
		baseURL: url,
		token:   token,
		client:  &http.Client{},
	}
}

func (c *Client) get(endpoint string, target interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+"/api/dcim/"+endpoint+"/?limit=0", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

func (c *Client) FetchDevices() (*DeviceResponse, error) {
	var resp DeviceResponse
	if err := c.get("devices", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) FetchInterfaces() (*InterfaceResponse, error) {
	var resp InterfaceResponse
	if err := c.get("interfaces", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) FetchCables() (*CableResponse, error) {
	var resp CableResponse
	if err := c.get("cables", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
