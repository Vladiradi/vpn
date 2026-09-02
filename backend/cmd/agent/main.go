package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vpn-project/backend/internal/agentstream"
	"github.com/vpn-project/backend/internal/config"
	"github.com/vpn-project/backend/internal/dockerctl"
	"github.com/vpn-project/backend/internal/xray"
)

func main() {
	cfg := config.Load()

	if cfg.APIMode != "agent" {
		log.Println("warning: API_MODE is not 'agent', running in agent mode anyway")
	}

	if cfg.PanelURL == "" || cfg.NodeToken == "" {
		log.Fatal("PANEL_URL and NODE_TOKEN are required for agent mode")
	}

	// Optional local health endpoint.
	go runHealthServer()

	// Connect to panel and maintain websocket connection.
	connectAndServe(cfg.PanelURL, cfg.NodeToken, cfg.NodePublicIP)
}

func runHealthServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","component":"agent"}`))
	})
	log.Println("Agent health server on :8081")
	log.Println(http.ListenAndServe(":8081", nil))
}

func connectAndServe(panelURL, token, publicIP string) {
	wsURL, err := toWebSocketURL(panelURL)
	if err != nil {
		log.Fatalf("invalid panel url: %v", err)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		if err := runConnection(wsURL, token, publicIP, interrupt); err != nil {
			log.Printf("connection ended: %v", err)
		}

		log.Println("connection lost, reconnecting in 5s")
		select {
		case <-interrupt:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func runConnection(wsURL, token, publicIP string, interrupt chan os.Signal) error {
	log.Printf("connecting to panel: %s", wsURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	// Send hello message.
	hello := agentstream.Message{
		Type: agentstream.MessageTypeHello,
		Payload: map[string]any{
			"token":     token,
			"public_ip": publicIP,
			"version":   "0.1.0",
		},
	}
	if err := conn.WriteJSON(hello); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	log.Println("connected to panel")
	done := make(chan struct{})
	go readLoop(conn, done)

	heartbeatTicker := time.NewTicker(10 * time.Second)
	statsTicker := time.NewTicker(60 * time.Second)
	defer heartbeatTicker.Stop()
	defer statsTicker.Stop()

	for {
		select {
		case <-done:
			return fmt.Errorf("read loop closed")
		case <-heartbeatTicker.C:
			if err := conn.WriteJSON(agentstream.Message{
				Type: agentstream.MessageTypeHeartbeat,
				Payload: map[string]any{
					"status": "ok",
				},
			}); err != nil {
				return fmt.Errorf("heartbeat: %w", err)
			}
		case <-statsTicker.C:
			if err := sendStats(conn); err != nil {
				log.Printf("failed to send stats: %v", err)
			}
		case <-interrupt:
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return fmt.Errorf("interrupted")
		}
	}
}

func sendStats(conn *websocket.Conn) error {
	if os.Getenv("KUBERNETES_MODE") == "1" {
		// Stats collection via Docker API is not available inside Kubernetes.
		return nil
	}

	stats, err := collectStats()
	if err != nil {
		return err
	}

	uplink, downlink := xray.AggregateTraffic(stats)
	deviceStats := make([]agentstream.DeviceStat, 0, len(stats))
	for _, s := range stats {
		deviceStats = append(deviceStats, agentstream.DeviceStat{
			UUID:     s.UUID,
			Uplink:   s.Uplink,
			Downlink: s.Downlink,
		})
	}

	payload := map[string]any{
		"uplink":   uplink,
		"downlink": downlink,
		"devices":  deviceStats,
	}

	return conn.WriteJSON(agentstream.Message{
		Type:    agentstream.MessageTypeStats,
		Payload: payload,
	})
}

func collectStats() ([]xray.TrafficStat, error) {
	docker := dockerctl.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, err := docker.FindContainerByName(ctx, []string{"/xray", "xray"})
	if err != nil {
		return nil, fmt.Errorf("find xray container: %w", err)
	}

	output, err := docker.Exec(ctx, container.ID, []string{"xray", "api", "statsquery", "--server=127.0.0.1:10085", "-reset", "true"})
	if err != nil {
		return nil, fmt.Errorf("query xray stats: %w", err)
	}

	return xray.ParseStatsResponse(output)
}

func readLoop(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		var msg agentstream.Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("read error: %v", err)
			}
			return
		}

		switch msg.Type {
		case agentstream.MessageTypeConfig:
			if err := applyConfig(msg.Payload); err != nil {
				log.Printf("failed to apply config: %v", err)
			} else {
				log.Println("xray config applied and container restarted")
			}
		case agentstream.MessageTypePing:
			// No-op, ping keeps connection alive.
		case agentstream.MessageTypeError:
			payloadJSON, _ := json.Marshal(msg.Payload)
			log.Printf("panel error: %s", payloadJSON)
		default:
			log.Printf("unknown message type: %s", msg.Type)
		}
	}
}

const xrayConfigPath = "/data/xray/config.json"

func applyConfig(payload map[string]any) error {
	xrayConfig, ok := payload["xray_config"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing xray_config in payload")
	}

	data, err := json.MarshalIndent(xrayConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal xray config: %w", err)
	}

	if err := os.WriteFile(xrayConfigPath, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", xrayConfigPath, err)
	}

	if os.Getenv("KUBERNETES_MODE") == "1" {
		// In Kubernetes the xray sidecar reloads the config from the shared volume.
		return nil
	}

	docker := dockerctl.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, err := docker.FindContainerByName(ctx, []string{"/xray", "xray"})
	if err != nil {
		return fmt.Errorf("find xray container: %w", err)
	}

	if err := docker.RestartContainer(ctx, container.ID, 10); err != nil {
		return fmt.Errorf("restart xray container: %w", err)
	}
	return nil
}

func toWebSocketURL(input string) (string, error) {
	u, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/api/v1/nodes/stream"
	return u.String(), nil
}
