package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const path = "MessageHistory.txt"

var (
	clients             = make(map[*websocket.Conn]string) // indicate a particular client at that index is holding a connection or not
	clientsLock         = sync.Mutex{}
	broadcast           = make(chan string)
	clientCounter       = 0
	maxClients          = 100
	clientSemaphore     = make(chan struct{}, maxClients)
	msgHistory          []string
	historyLock         = sync.Mutex{}
	disconnectedClients int
	disconnectLock      = sync.Mutex{}
)

func StartServer(port int) {
	// start the server
	http.HandleFunc("/ws", handleConnections)

	// start a go routine to listen to messages
	go handleMessages()

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("WebSocket server started on port %d\n", port)
	server := &http.Server{Addr: addr}
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server Error:", err)
		}
	}()

	handleServerShutdown(server)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// upgrade the connection, handle the error and defer its closing
	// lock - add conn to clients - unlock
	// listen continuously for the message
	// read the msg, if theres an error => lock delete unlock and tell client discon
	// add to broadcast

	select {
	case clientSemaphore <- struct{}{}:
		defer func() { <-clientSemaphore }()
	default:
		http.Error(w, "Server is at max capacity, try again later", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close() // never ever forget this

	clientsLock.Lock()
	clientCounter++
	clientID := fmt.Sprintf("client%d", clientCounter)
	clients[conn] = clientID
	clientsLock.Unlock()

	fmt.Println("New client connected...")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			clientsLock.Lock()
			delete(clients, conn)
			clientsLock.Unlock()

			disconnectLock.Lock()
			disconnectedClients++
			disconnectLock.Unlock()

			fmt.Println("Client disconnected")
			break
		}
		clientsLock.Lock()
		senderID := clients[conn]
		clientsLock.Unlock()

		formattedMsg := fmt.Sprintf("%s: %s", senderID, string(msg))
		broadcast <- formattedMsg
	}
}

func handleMessages() {
	// keep listening for a message, once u get one from the broadcast
	// lock the clients, go over them
	// writemessage to everyone if any error, close the conn and then delete conn from clients
	// unlock the clients
	for msg := range broadcast {

		historyLock.Lock()
		msgHistory = append(msgHistory, msg)
		historyLock.Unlock()

		clientsLock.Lock()
		for conn := range clients {
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				conn.Close() // remember to close the connection before deleting
				delete(clients, conn)
			}
		}
		clientsLock.Unlock()
	}
}

func handleServerShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("\nShutting down server...")
	close(broadcast)
	clientsLock.Lock()
	for conn := range clients {
		conn.WriteMessage(websocket.TextMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shutting down server"))
		conn.Close()
	}
	clientsLock.Unlock()

	msgHistory = msgHistory[:len(msgHistory)-disconnectedClients]
	err := writeToFile(msgHistory, path)
	if err != nil {
		fmt.Println("Error committing the message history:", err)
	}

	server.Close()
	fmt.Println("Server gracefully stopped.")
}

func writeToFile(msgHistory []string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range msgHistory {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			file.Close()
			return err
		}
	}
	return nil
}
