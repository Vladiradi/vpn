package agentstream

// MessageType defines the kind of message exchanged between panel and agent.
type MessageType string

const (
	// Agent → Panel
	MessageTypeHello     MessageType = "hello"
	MessageTypeHeartbeat MessageType = "heartbeat"
	MessageTypeStats     MessageType = "stats"

	// Panel → Agent
	MessageTypeConfig MessageType = "config"
	MessageTypePing   MessageType = "ping"
	MessageTypeError  MessageType = "error"
)

// Message is the envelope for all websocket messages.
type Message struct {
	Type MessageType `json:"type"`
	// Payload is type-specific JSON data.
	Payload map[string]any `json:"payload,omitempty"`
}

// HelloPayload is sent by the agent on connection.
type HelloPayload struct {
	Token    string `json:"token"`
	PublicIP string `json:"public_ip"`
	Version  string `json:"version"`
}

// ConfigPayload is sent by the panel to configure XRay.
type ConfigPayload struct {
	XRayConfig map[string]any `json:"xray_config"`
}

// HeartbeatPayload is sent periodically by the agent.
type HeartbeatPayload struct {
	Status string `json:"status"`
}

// StatsPayload is sent periodically by the agent with traffic data.
type StatsPayload struct {
	NodeID   string         `json:"node_id"`
	Uplink   int64          `json:"uplink"`
	Downlink int64          `json:"downlink"`
	Devices  []DeviceStat   `json:"devices"`
}

// DeviceStat holds per-device traffic.
type DeviceStat struct {
	UUID     string `json:"uuid"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}
