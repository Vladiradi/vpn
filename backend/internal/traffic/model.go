// Package traffic stores traffic statistics reported by node agents.
package traffic

import "time"

// Stat represents one traffic measurement for a device on a node.
type Stat struct {
	ID       string    `json:"id"`
	DeviceID string    `json:"device_id"`
	NodeID   string    `json:"node_id"`
	Ts       time.Time `json:"ts"`
	Uplink   int64     `json:"uplink"`
	Downlink int64     `json:"downlink"`
}
