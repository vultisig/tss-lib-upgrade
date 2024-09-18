package network

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <id> <address> <party1:address1> [party2:address2] ...")
		fmt.Println("Example: go run main.go Alice localhost:8080 Bob:localhost:8081 Charlie:localhost:8082")
		return
	}

	id := party.ID(os.Args[1])
	address := os.Args[2]

	parties := make(party.IDSlice, 0)
	addresses := make(map[party.ID]string)

	for _, arg := range os.Args[3:] {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid party address format: %s\n", arg)
			continue
		}
		partyID := party.ID(parts[0])
		parties = append(parties, partyID)
		addresses[partyID] = parts[1]
	}

	// Add self to parties and addresses
	parties = append(parties, id)
	addresses[id] = address

	network := NewNetwork(id, address, parties, addresses)

	fmt.Printf("Joined the network as %s\n", id)
	fmt.Println("Type 'quit' to exit")
	fmt.Println("Enter messages in the format: recipient message")

	go receiveMessages(network, id)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if input == "quit" {
			break
		}

		parts := strings.SplitN(input, " ", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid input. Use format: 'recipient message'")
			continue
		}

		recipient := party.ID(parts[0])
		message := parts[1]

		msg := &protocol.Message{
			From: id,
			To:   recipient,
			Data: []byte(message),
		}

		network.Send(msg)
	}

	network.Done(id)
	fmt.Println("Exiting...")
}

func receiveMessages(network *Network, id party.ID) {
	for msg := range network.Next(id) {
		fmt.Printf("Received from %s: %s\n", msg.From, string(msg.Data))
	}
}
