package client

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/websocket"
)

const serverAddr = "ws://localhost:8080/ws"

func ConnectClient(host string, port int) {
	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v\n", err)
	}
	defer conn.Close()

	fmt.Println("Connected to the broadcast server.")
	fmt.Println("Type your message and press Enter to send. Type 'exit' to quit.")

	interrupt := make(chan os.Signal, 1)                    // a channel to receive os signals, buffer size 1 => store only one signal at a time.
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM) // os.Interrupt triggered w CTRL+C, syscall.SIGTERM in linux using kill,
	// interrupt is called with any one of these

	go handleUserInput(conn)
	go handleServerMessages(conn)
	waitForInterrupt(interrupt, conn)
}

func handleUserInput(conn *websocket.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "exit" {
			fmt.Println("Disconnecting...")
			conn.WriteMessage(websocket.TextMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			return
		}

		err := conn.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Println("Error sending message:", err)
			return
		}
	}
}

func handleServerMessages(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Connection closed by server.")
			return
		}
		fmt.Println("\n[Broadcast] : ", string(msg))
	}
}

func waitForInterrupt(interrupt chan os.Signal, conn *websocket.Conn) {
	<-interrupt
	fmt.Println("Closing connection...")
	conn.WriteMessage(websocket.TextMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
