package network

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

// Network implements a point-to-point network between different parties using TCP connections.
type Network struct {
	id             party.ID
	parties        party.IDSlice
	addresses      map[party.ID]string
	connections    map[party.ID]net.Conn
	listenChannels map[party.ID]chan *protocol.Message
	done           chan struct{}
	mtx            sync.Mutex
}

func NewNetwork(id party.ID, address string, parties party.IDSlice, addresses map[party.ID]string) *Network {
	n := &Network{
		id:             id,
		parties:        parties,
		addresses:      addresses,
		connections:    make(map[party.ID]net.Conn),
		listenChannels: make(map[party.ID]chan *protocol.Message),
		done:           make(chan struct{}),
	}
	go n.listen(address)
	go n.connectToParties()
	return n
}

func (n *Network) listen(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Error listening: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go n.handleConnection(conn)
	}
}

func (n *Network) handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		from, to := party.ID(parts[0]), party.ID(parts[1])
		msg := &protocol.Message{
			From: from,
			To:   to,
			Data: []byte(parts[2]),
		}
		n.mtx.Lock()
		if ch, ok := n.listenChannels[to]; ok {
			ch <- msg
		}
		n.mtx.Unlock()
	}
}

func (n *Network) connectToParties() {
	for _, p := range n.parties {
		if p == n.id {
			continue
		}
		go n.connectToParty(p)
	}
}

func (n *Network) connectToParty(id party.ID) {
	for {
		addr, ok := n.addresses[id]
		if !ok {
			fmt.Printf("No address for party %s\n", id)
			return
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Printf("Error connecting to %s: %v. Retrying in 5 seconds...\n", id, err)
			time.Sleep(5 * time.Second)
			continue
		}
		n.mtx.Lock()
		n.connections[id] = conn
		n.mtx.Unlock()
		fmt.Printf("Connected to %s\n", id)
		return
	}
}

func (n *Network) Next(id party.ID) <-chan *protocol.Message {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	if _, ok := n.listenChannels[id]; !ok {
		n.listenChannels[id] = make(chan *protocol.Message, 100)
	}
	return n.listenChannels[id]
}

func (n *Network) Send(msg *protocol.Message) {
	n.mtx.Lock()
	//fmt.Print(msg)
	conn, ok := n.connections[msg.To] //continue here : when msg.To is empty, it means send to everyone? See original function.
	n.mtx.Unlock()
	if !ok {
		fmt.Printf("No connection to %s, message dropped\n", msg.To)
		return
	}
	_, err := fmt.Fprintf(conn, "%s:%s:%s\n", msg.From, msg.To, string(msg.Data))
	if err != nil {
		fmt.Printf("Error sending message to %s: %v\n", msg.To, err)
		n.mtx.Lock()
		delete(n.connections, msg.To)
		n.mtx.Unlock()
		go n.connectToParty(msg.To)
	}
}

func (n *Network) Done(id party.ID) chan struct{} {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	if ch, ok := n.listenChannels[id]; ok {
		close(ch)
		delete(n.listenChannels, id)
	}
	if conn, ok := n.connections[id]; ok {
		conn.Close()
		delete(n.connections, id)
	}
	if len(n.listenChannels) == 0 && len(n.connections) == 0 {
		close(n.done)
	}
	return n.done
}

func (n *Network) Quit(id party.ID) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.parties = n.parties.Remove(id)
	if ch, ok := n.listenChannels[id]; ok {
		close(ch)
		delete(n.listenChannels, id)
	}
	if conn, ok := n.connections[id]; ok {
		conn.Close()
		delete(n.connections, id)
	}
}
