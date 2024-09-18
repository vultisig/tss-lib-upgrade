package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

type Network struct {
	id         party.ID
	parties    map[party.ID]string
	conns      map[party.ID]net.Conn
	listenAddr string
	mu         sync.Mutex
}

func NewNetwork(id party.ID, listenAddr string) *Network {
	return &Network{
		id:         id,
		parties:    make(map[party.ID]string),
		conns:      make(map[party.ID]net.Conn),
		listenAddr: listenAddr,
	}
}

func (n *Network) Listen() error {
	listener, err := net.Listen("tcp", n.listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	fmt.Printf("Listening on %s\n", n.listenAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go n.handleConnection(conn)
	}
}

func (n *Network) Connect(id party.ID, addr string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.conns[id]; exists {
		return nil // Already connected
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	n.conns[id] = conn
	n.parties[id] = addr
	go n.handleConnection(conn)
	fmt.Printf("Connected to %s at %s\n", id, addr)
	return nil
}

func (n *Network) handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		fmt.Printf("Received: %s\n", scanner.Text())
	}
}

func (n *Network) Send(msg *protocol.Message) error {
	n.mu.Lock()
	conn, ok := n.conns[msg.To]
	addr, addrOk := n.parties[msg.To]
	n.mu.Unlock()

	if !ok {
		if !addrOk {
			return fmt.Errorf("no address for party %s", msg.To)
		}
		err := n.Connect(msg.To, addr)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %v", msg.To, err)
		}
		n.mu.Lock()
		conn = n.conns[msg.To]
		n.mu.Unlock()
	}

	_, err := fmt.Fprintf(conn, "%s: %s\n", msg.From, string(msg.Data))
	return err
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run example.go <name> <listen_address> [other_party1:address1] [other_party2:address2] ...")
		fmt.Println("Example: go run example.go Alice :8080 Bob:localhost:8081 Charlie:localhost:8082")
		return
	}

	name := party.ID(os.Args[1])
	listenAddr := os.Args[2]
	net := NewNetwork(name, listenAddr)

	for _, arg := range os.Args[3:] {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid party address format: %s\n", arg)
			continue
		}
		net.parties[party.ID(parts[0])] = parts[1]
	}

	go func() {
		err := net.Listen()
		if err != nil {
			fmt.Printf("Error listening: %v\n", err)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter recipient and message (e.g., 'Bob Hello!'): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" {
			break
		}

		parts := strings.SplitN(input, " ", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid input. Use format: 'Recipient Message'")
			continue
		}

		recipient := party.ID(parts[0])
		message := parts[1]

		msg := &protocol.Message{
			From: name,
			To:   recipient,
			Data: []byte(message),
		}

		err := net.Send(msg)
		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		}
	}
}
