package main

import (
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

func getLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}

func executeFunction() {
	log.Println("Hola mundo desde el cliente 🚀")
}

func main() {
	wsURL := "ws://localhost:8080/ws"

	for {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Println("Servidor no disponible, reintentando en 10s...")
			time.Sleep(10 * time.Second)
			continue
		}

		info := ClientInfo{
			IP:       getLocalIP(),
			MAC:      "00:11:22:33:44:55",
			Username: os.Getenv("USER"),
		}

		conn.WriteJSON(info)
		log.Println("Conectado al servidor WS")

		for {
			var msg map[string]string
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("Conexión perdida, reconectando...")
				conn.Close()
				break
			}

			if msg["action"] == "EXECUTE" {
				executeFunction()
			}
		}
	}
}
