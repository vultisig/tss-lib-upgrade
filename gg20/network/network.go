package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
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
	for _, partyID := range parties {
		n.listenChannels[partyID] = make(chan *protocol.Message, 1000000)
	}
	go n.listen(address)
	// Wait for 3 seconds before connecting to parties
	//time.Sleep(3 * time.Second)
	//go n.connectToParties()
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
	defer conn.Close() // Ensure the connection is closed when we're done

	scanner := bufio.NewScanner(conn)

	// Increase the buffer size
	buf := make([]byte, 0, 64*1024) // 64KB buffer
	scanner.Buffer(buf, 1024*1024)  // Allow up to 1MB per line

	for scanner.Scan() { // Read messages line by line
		// Read the incoming JSON data
		data := scanner.Bytes()
		//fmt.Printf(" \n Received raw data \n")
		//fmt.Printf("Received raw data: %s\n", string(data))

		// Deserialize the message
		var msg protocol.Message
		err := json.Unmarshal(data, &msg)
		if err != nil {
			fmt.Printf("Error deserializing message: %v\n", err)
			continue // Skip this message if deserialization fails
		}

		n.mtx.Lock() // Lock to safely access shared resources

		if msg.Broadcast {
			// Handle broadcast messages
			//fmt.Printf("Broadcasting message from %s\n", msg.From)
			for id, ch := range n.listenChannels {
				if id != msg.From { // Don't send back to the sender
					select {
					case ch <- &msg:
						//fmt.Printf("Broadcast message delivered to %s\n", id)
					default:
						fmt.Printf("Channel full, broadcast message to %s dropped\n", id)
					}
				}
			}
		} else if ch, ok := n.listenChannels[msg.To]; ok {
			// Handle direct messages
			//fmt.Printf("Delivering message to %s:\n", msg.To)
			select {
			case ch <- &msg:
				//fmt.Printf("Message delivered to %s\n", msg.To)
			default:
				fmt.Printf("Channel full, message to %s dropped\n", msg.To)
			}
		} else {
			fmt.Printf("No listen channel for %s\n", msg.To)
		}

		n.mtx.Unlock() // Unlock after we're done with shared resources

		//fmt.Printf("Processed message from %s to %s:\n", msg.From, msg.To)
	}

	// Check for any errors that occurred during scanning
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from connection: %v\n", err)
	}
}

func (n *Network) ConnectToParties() {
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
		fmt.Printf("Creating listen channel for %s\n", id)
		n.listenChannels[id] = make(chan *protocol.Message, 1000)
	}
	return n.listenChannels[id]
}

func (n *Network) Send(msg *protocol.Message) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	//fmt.Println("\n Sending message \n", msg)
	//fmt.Println(msg.RoundNumber)

	fmt.Println("\n Sending message", msg)

	for id, conn := range n.connections {
		if msg.IsFor(id) && conn != nil {
			// Serialize the entire message
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("Error serializing message for %s: %v\n", id, err)
				continue
			}

			// Send the serialized message
			//fmt.Printf("Sending message to %s: \n", id)

			//fmt.Println(msg.RoundNumber)
			_, err = conn.Write(append(data, '\n'))
			if err != nil {
				fmt.Printf("Error sending message to %s: %v\n", id, err)
				delete(n.connections, id)
				go n.connectToParty(id)
			}
		}
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

// HandlerLoop blocks until the handler has finished. The result of the execution is given by Handler.Result().
func HandlerLoop(id party.ID, h protocol.Handler, network *Network) {
	for {
		fmt.Printf("\n[Party %s] About to enter select statement", id)
		select {
		// outgoing messages
		case msg, ok := <-h.Listen():
			fmt.Print("\n Here we receive a msg from handler ", msg)
			if !ok {
				//<-network.Done(id)
				fmt.Print("\n CHANNEL CLOSED ", msg)
				// the channel was closed, indicating that the protocol is done executing.
				return
			}
			go network.Send(msg)

		// incoming messages
		case msg := <-network.Next(id):
			fmt.Print("\n Here we receive a msg from network: ", msg)
			h.Accept(msg)
			fmt.Print("\n Msg accepted from network to handler: ", msg)

		default:
			//fmt.Printf("[Party %s] No messages available, sleeping for 1 second\n", id)
			//time.Sleep(1 * time.Second)
		}
		fmt.Printf("\n[Party %s] Exited select statement", id)
	}
}
