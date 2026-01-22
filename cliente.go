package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type ClientInfo struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Username string `json:"username"`
}

var (
	CENTRALITA_IP = "ws://localhost:8080/ws"
	cancelFunc context.CancelFunc
)

// ===========================
// aqui va la ejecución del START
func executeLoop(ctx context.Context, payload string) {
	for {
		select {
		case <-ctx.Done():
			log.Println("se para")
			return
		default:
			log.Println("Hola mundo:", payload)
			// time.Sleep(2 * time.Second) //no hay sleep que valga 
		}
	}
}

func getLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			return ipnet.IP.String()
		}
	}
	return "unknown"
}

func main() {

	wsURL := CENTRALITA_IP

	for {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		info := ClientInfo{
			IP:       getLocalIP(),
			MAC:      "00:11:22:33:44:55",
			Username: os.Getenv("USER"),
		}
		conn.WriteJSON(info)

		for {
			var msg map[string]string
			if err := conn.ReadJSON(&msg); err != nil {
				if cancelFunc != nil {
					cancelFunc()
				}
				conn.Close()
				break
			}

			// =======================
			// 	aqui van las ordenes que se pueden recibir de la centralita
			switch msg["action"] {
			case "START":
				if cancelFunc != nil {
					cancelFunc()
				}

				ctx, cancel := context.WithCancel(context.Background())
				cancelFunc = cancel

				go executeLoop(ctx, msg["payload"])

			case "STOP":
				if cancelFunc != nil {
					cancelFunc()
					cancelFunc = nil
				}
			}
		}
	}
}
