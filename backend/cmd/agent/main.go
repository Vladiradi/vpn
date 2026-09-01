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

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return fmt.Errorf("read loop closed")
		case <-ticker.C:
			if err := conn.WriteJSON(agentstream.Message{
				Type: agentstream.MessageTypeHeartbeat,
				Payload: map[string]any{
					"status": "ok",
				},
			}); err != nil {
				return fmt.Errorf("heartbeat: %w", err)
			}
		case <-interrupt:
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return fmt.Errorf("interrupted")
		}
	}
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
