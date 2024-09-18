package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

type Network struct {
	id         party.ID
	parties    map[party.ID]string
	conns      map[party.ID]chan *protocol.Message
	listenAddr string
	mu         sync.Mutex
	incoming   chan *protocol.Message
}

func NewNetwork(id party.ID, listenAddr string) *Network {
	return &Network{
		id:         id,
		parties:    make(map[party.ID]string),
		conns:      make(map[party.ID]chan *protocol.Message),
		listenAddr: listenAddr,
		incoming:   make(chan *protocol.Message, 100),
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

func (n *Network) Connect(id party.ID, addr string) {
	for {
		n.mu.Lock()
		_, exists := n.conns[id]
		n.mu.Unlock()
		if exists {
			return
		}

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Printf("Error connecting to %s: %v. Retrying in 5 seconds...\n", id, err)
			time.Sleep(5 * time.Second)
			continue
		}

		ch := make(chan *protocol.Message, 100)
		n.mu.Lock()
		n.conns[id] = ch
		n.mu.Unlock()
		go n.handleOutgoing(id, conn, ch)
		go n.handleIncoming(id, conn)
		fmt.Printf("Connected to %s at %s\n", id, addr)
		return
	}
}

func (n *Network) handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		n.incoming <- &protocol.Message{
			From: party.ID(parts[0]),
			To:   n.id,
			Data: []byte(strings.TrimSpace(parts[1])),
		}
	}
}

func (n *Network) handleOutgoing(id party.ID, conn net.Conn, ch <-chan *protocol.Message) {
	defer conn.Close()
	for msg := range ch {
		_, err := fmt.Fprintf(conn, "%s: %s\n", msg.From, string(msg.Data))
		if err != nil {
			fmt.Printf("Error sending to %s: %v\n", id, err)
			n.mu.Lock()
			delete(n.conns, id)
			n.mu.Unlock()
			go n.Connect(id, n.parties[id])
			return
		}
	}
}

func (n *Network) handleIncoming(id party.ID, conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		n.incoming <- &protocol.Message{
			From: party.ID(parts[0]),
			To:   n.id,
			Data: []byte(strings.TrimSpace(parts[1])),
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from %s: %v\n", id, err)
		n.mu.Lock()
		delete(n.conns, id)
		n.mu.Unlock()
		go n.Connect(id, n.parties[id])
	}
}

func (n *Network) Send(msg *protocol.Message) error {
	n.mu.Lock()
	ch, ok := n.conns[msg.To]
	n.mu.Unlock()

	if !ok {
		// If not connected, queue the message and try to connect
		go func() {
			n.Connect(msg.To, n.parties[msg.To])
			n.Send(msg) // Try sending again after connection attempt
		}()
		return nil
	}

	select {
	case ch <- msg:
		return nil
	default:
		return fmt.Errorf("channel full, message to %s dropped", msg.To)
	}
}

func (n *Network) Receive() *protocol.Message {
	return <-n.incoming
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

	// Connect to all other parties
	for id, addr := range net.parties {
		if id != name {
			go net.Connect(id, addr)
		}
	}

	go func() {
		for msg := range net.incoming {
			fmt.Printf("Received from %s: %s\n", msg.From, string(msg.Data))
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
