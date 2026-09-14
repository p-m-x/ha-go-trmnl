// Package ha provides a minimal Home Assistant REST API client.
package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// State represents a Home Assistant entity state.
type State struct {
	EntityID    string            `json:"entity_id"`
	State       string            `json:"state"`
	Attributes  map[string]any    `json:"attributes"`
	LastChanged time.Time         `json:"last_changed"`
	LastUpdated time.Time         `json:"last_updated"`
}

// FriendlyName returns the friendly_name attribute or falls back to the entity ID.
func (s *State) FriendlyName() string {
	if v, ok := s.Attributes["friendly_name"]; ok {
		if name, ok := v.(string); ok && name != "" {
			return name
		}
	}
	// "sensor.living_room_temp" → "Living Room Temp"
	parts := strings.SplitN(s.EntityID, ".", 2)
	if len(parts) == 2 {
		return toTitle(parts[1])
	}
	return s.EntityID
}

// Unit returns the unit_of_measurement attribute or empty string.
func (s *State) Unit() string {
	if v, ok := s.Attributes["unit_of_measurement"]; ok {
		if u, ok := v.(string); ok {
			return u
		}
	}
	return ""
}

// Domain returns the domain portion of the entity ID (e.g. "sensor").
func (s *State) Domain() string {
	if idx := strings.Index(s.EntityID, "."); idx != -1 {
		return s.EntityID[:idx]
	}
	return ""
}

// Client is a Home Assistant REST API client.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New creates a new Client.
//
// Inside a Home Assistant add-on, pass baseURL="http://supervisor/core" and
// token=os.Getenv("SUPERVISOR_TOKEN"). The supervisor API is at
// http://supervisor/core/api/… and accepts the supervisor token as Bearer auth.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// GetState fetches the state of a single entity.
func (c *Client) GetState(ctx context.Context, entityID string) (*State, error) {
	url := fmt.Sprintf("%s/api/states/%s", c.baseURL, entityID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ha request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ha returned status %d for entity %s", resp.StatusCode, entityID)
	}

	var state State
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("decoding ha state: %w", err)
	}
	return &state, nil
}

// GetStates fetches the states of multiple entities concurrently.
// Errors are collected but do not abort the batch; unreachable entities
// are omitted from the returned slice.
func (c *Client) GetStates(ctx context.Context, entityIDs []string) ([]*State, []error) {
	type result struct {
		state *State
		err   error
	}
	ch := make(chan result, len(entityIDs))
	for _, id := range entityIDs {
		go func(id string) {
			s, err := c.GetState(ctx, id)
			ch <- result{s, err}
		}(id)
	}

	states := make([]*State, 0, len(entityIDs))
	var errs []error
	for range entityIDs {
		r := <-ch
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			states = append(states, r.state)
		}
	}

	// Restore original order.
	order := make(map[string]int, len(entityIDs))
	for i, id := range entityIDs {
		order[id] = i
	}
	sortStates(states, order)

	return states, errs
}

func sortStates(states []*State, order map[string]int) {
	for i := 1; i < len(states); i++ {
		for j := i; j > 0 && order[states[j].EntityID] < order[states[j-1].EntityID]; j-- {
			states[j], states[j-1] = states[j-1], states[j]
		}
	}
}

func toTitle(s string) string {
	words := strings.Split(strings.ReplaceAll(s, "_", " "), " ")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
