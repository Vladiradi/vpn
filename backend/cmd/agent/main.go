package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vpn-project/backend/internal/agentstream"
	"github.com/vpn-project/backend/internal/config"
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
		log.Printf("connecting to panel: %s", wsURL)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("dial failed: %v, retrying in 5s", err)
			select {
			case <-interrupt:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

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
			log.Printf("failed to send hello: %v", err)
			conn.Close()
			continue
		}

		log.Println("connected to panel")
		done := make(chan struct{})

		go readLoop(conn, done)

		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-done:
				ticker.Stop()
				conn.Close()
				break
			case <-ticker.C:
				if err := conn.WriteJSON(agentstream.Message{
					Type: agentstream.MessageTypeHeartbeat,
					Payload: map[string]any{
						"status": "ok",
					},
				}); err != nil {
					log.Printf("heartbeat failed: %v", err)
				}
			case <-interrupt:
				ticker.Stop()
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				conn.Close()
				return
			}
		}

		log.Println("connection lost, reconnecting in 5s")
		select {
		case <-interrupt:
			return
		case <-time.After(5 * time.Second):
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
			log.Printf("received config: %v", msg.Payload)
			// TODO: write xray config.json and restart xray container
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
