package communication

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

// Network simulates a network between different parties using TCP connections over localhost.
type Network struct {
	parties     party.IDSlice
	connections map[party.ID]net.Conn
	listenAddrs map[party.ID]string
	done        chan struct{}
	mtx         sync.Mutex
}

func NewNetwork(parties party.IDSlice) *Network {
	return &Network{
		parties:     parties,
		connections: make(map[party.ID]net.Conn),
		listenAddrs: make(map[party.ID]string),
		done:        make(chan struct{}),
	}
}

func (n *Network) init() {
	basePort := 8000
	for i, id := range n.parties {
		addr := fmt.Sprintf("localhost:%d", basePort+i)
		n.listenAddrs[id] = addr
	}
}

func (n *Network) Listen(id party.ID) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	if len(n.listenAddrs) == 0 {
		n.init()
	}

	addr := n.listenAddrs[id]
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		n.connections[id] = conn
	}()

	return nil
}

func (n *Network) Connect(id party.ID) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	addr := n.listenAddrs[id]
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	n.connections[id] = conn
	return nil
}

func (n *Network) Send(msg *protocol.Message) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	for id, conn := range n.connections {
		if msg.IsFor(id) {
			data, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			_, err = conn.Write(data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Network) Receive(id party.ID) (*protocol.Message, error) {
	n.mtx.Lock()
	conn, ok := n.connections[id]
	n.mtx.Unlock()

	if !ok {
		return nil, fmt.Errorf("no connection for party %v", id)
	}

	var msg protocol.Message
	decoder := json.NewDecoder(conn)
	err := decoder.Decode(&msg)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func (n *Network) Close(id party.ID) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	conn, ok := n.connections[id]
	if !ok {
		return nil
	}

	err := conn.Close()
	delete(n.connections, id)

	if len(n.connections) == 0 {
		close(n.done)
	}

	return err
}

func (n *Network) Done() <-chan struct{} {
	return n.done
}

func (n *Network) Quit(id party.ID) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.parties = n.parties.Remove(id)
	n.Close(id)
}
