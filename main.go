package main

import (
	"flag"
	"fmt"
	"os"
	"broadcast-server/client"
	"broadcast-server/server"
)

func main(){
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	connectCmd := flag.NewFlagSet("connect", flag.ExitOnError)

	startPort := startCmd.Int("port", 8080, "")
	connectHost := connectCmd.String("host", "localhost", "")
	connectPort := connectCmd.Int("port", 8080, "")

	if len(os.Args) < 2 {
		fmt.Println("Usage: broadcast-server [start|connect] [--options]")
		fmt.Println("Commands:")
		fmt.Println("  start    Start the WebSocket broadcast server")
		fmt.Println("  connect  Connect to the server as a client")
		fmt.Println("\nOptions:")
		fmt.Println("  --port <port>   Specify the server/client port (default: 8080)")
		fmt.Println("  --host <host>   Specify the server hostname (default: localhost)")
		os.Exit(0)
	}

	switch os.Args[1]{
	case "start":
		startCmd.Parse(os.Args[2:])
		server.StartServer(*startPort)
	case "connect":
		connectCmd.Parse(os.Args[2:])
		client.ConnectClient(*connectHost, *connectPort)
	default:
		fmt.Println("Invalid command. Use 'start' or 'connect'.")
		os.Exit(1)
	}
}