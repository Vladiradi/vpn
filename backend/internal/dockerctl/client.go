// Package dockerctl provides a minimal Docker Engine API client for the node agent.
package dockerctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const dockerSocket = "/var/run/docker.sock"

// Client is a thin Docker API client that talks over the local Unix socket.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a Docker API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", dockerSocket)
				},
			},
			Timeout: 30 * time.Second,
		},
		baseURL: "http://docker",
	}
}

// Container represents a subset of the Docker container list response.
type Container struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

// FindContainerByName returns the first container matching one of the given names.
// Names should include the leading slash as returned by Docker (e.g. "/xray").
func (c *Client) FindContainerByName(ctx context.Context, names []string) (*Container, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/containers/json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list containers: %s: %s", resp.Status, string(body))
	}

	var containers []Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decode container list: %w", err)
	}

	for _, cont := range containers {
		for _, n := range cont.Names {
			for _, wanted := range names {
				if n == wanted {
					return &cont, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("container not found")
}

// RestartContainer restarts the container with the given ID.
func (c *Client) RestartContainer(ctx context.Context, containerID string, timeoutSeconds int) error {
	url := fmt.Sprintf("%s/containers/%s/restart?t=%d", c.baseURL, containerID, timeoutSeconds)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("restart container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("restart container: %s: %s", resp.Status, string(body))
	}
	return nil
}
