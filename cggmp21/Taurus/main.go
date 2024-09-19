package main

import (
	"Taurus/network"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/taurusgroup/multi-party-sig/pkg/ecdsa"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/pool"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
	"github.com/taurusgroup/multi-party-sig/protocols/cmp"
)

var (
	// sessionID should be agreed upon beforehand, and must be unique among all protocol executions.
	// Alternatively, a counter may be used, which must be incremented after before every protocol start.
	sessionID []byte
	// group defines the cryptographic group over which
	group        = curve.Secp256k1{}
	participants = []party.ID{"a", "b", "c", "d", "e"}
	selfID       = participants[0] // we run the protocol as "a"
	threshold    = 3               // 4 or more participants are required to generate a signature
	message      = []byte("Hello, world!")
	ids          = party.IDSlice{"a", "b", "c", "d", "e"}
	id           = selfID
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

	net := network.NewNetwork(id, address, parties, addresses)

	/*// Wait for all parties to connect
	allConnected := make(chan struct{})
	go func() {
		for {
			net.mtx.Lock()
			if len(net.connections) == len(parties)-1 { // -1 because we don't connect to ourselves
				net.mtx.Unlock()
				close(allConnected)
				return
			}
			net.mtx.Unlock()
			time.Sleep(1 * time.Second)
		}
	}()

	fmt.Println("Waiting for all parties to connect...")
	<-allConnected
	fmt.Println("All parties connected. Proceeding with the protocol.")*/

	fmt.Printf("Joined the network as %s\n", id)
	fmt.Println("Type 'quit' to exit")
	fmt.Println("Enter messages in the format: recipient message")

	fmt.Println("Waiting 10 seconds before proceeding...")
	time.Sleep(3 * time.Second)
	fmt.Println("10 seconds have passed. Continuing with the protocol.")
	//go receiveMessages(net, id)
	//defer network.Done(id)

	//
	//
	//
	//

	ids := party.IDSlice{"Alice", "Bob", "Charlie"}
	threshold := 2
	messageToSign := []byte("hello")

	//net := network.NewNetwork(id, address, ids, addresses)

	var wg sync.WaitGroup

	/*for _, id := range ids {*/
	wg.Add(1)
	go func(id party.ID) {
		pl := pool.NewPool(0)
		fmt.Println("Starting protocol")
		defer pl.TearDown()
		if err := All(id, ids, threshold, messageToSign, net, &wg, pl); err != nil {
			fmt.Println(err)
		}
	}(id)
	/*} */
	wg.Wait()
}

func receiveMessages(network *network.Network, id party.ID) {
	for msg := range network.Next(id) {
		fmt.Printf("Received from %s: %s\n", msg.From, string(msg.Data))
	}
}

//
//
//
//
////
//
//
//
////
//
//
//
////
//
//
//
//

// HandlerLoop blocks until the handler has finished. The result of the execution is given by Handler.Result().
func HandlerLoop(id party.ID, h protocol.Handler, network *network.Network) {
	for {
		select {
		// outgoing messages
		case msg, ok := <-h.Listen():
			//fmt.Print("Here when we have to send a msg \n ", msg)
			if !ok {
				<-network.Done(id)
				// the channel was closed, indicating that the protocol is done executing.
				return
			}
			go network.Send(msg)

		// incoming messages
		case msg := <-network.Next(id):
			fmt.Print("Here is msg when received: ", msg)
			h.Accept(msg)
		}
	}
}

func All(id party.ID, ids party.IDSlice, threshold int, message []byte, n *network.Network, wg *sync.WaitGroup, pl *pool.Pool) error {

	//defer wg.Done()

	fmt.Println("Starting CMP Keygen")
	// CMP KEYGEN
	keygenConfig, err := CMPKeygen(id, ids, threshold, n, pl)
	if err != nil {
		return err
	}

	signers := ids[:threshold+1]
	if !signers.Contains(id) {
		n.Quit(id)
		return err
	}

	fmt.Println("Starting CMP Presign")

	// CMP PRESIGN
	preSignature, err := CMPPreSign(keygenConfig, signers, n, pl)
	if err != nil {
		return err
	}

	fmt.Println("Starting CMP Presign Online")

	// CMP PRESIGN ONLINE
	err = CMPPreSignOnline(keygenConfig, preSignature, message, n, pl)
	if err != nil {
		return err
	}
	return nil
}

func CMPKeygen(id party.ID, ids party.IDSlice, threshold int, n *network.Network, pl *pool.Pool) (*cmp.Config, error) {
	h, err := protocol.NewMultiHandler(cmp.Keygen(curve.Secp256k1{}, id, ids, threshold, pl), nil)
	if err != nil {
		return nil, err
	}
	fmt.Println("Starting CMP Keygen HandlerLoop")
	HandlerLoop(id, h, n)
	fmt.Println("CMP Keygen HandlerLoop finished")
	r, err := h.Result()
	if err != nil {
		return nil, err
	}

	return r.(*cmp.Config), nil
}

func CMPPreSign(c *cmp.Config, signers party.IDSlice, n *network.Network, pl *pool.Pool) (*ecdsa.PreSignature, error) {
	h, err := protocol.NewMultiHandler(cmp.Presign(c, signers, pl), nil)
	if err != nil {
		return nil, err
	}

	HandlerLoop(c.ID, h, n)

	signResult, err := h.Result()
	if err != nil {
		return nil, err
	}

	preSignature := signResult.(*ecdsa.PreSignature)
	if err = preSignature.Validate(); err != nil {
		return nil, errors.New("failed to verify cmp presignature")
	}
	return preSignature, nil
}

func CMPPreSignOnline(c *cmp.Config, preSignature *ecdsa.PreSignature, m []byte, n *network.Network, pl *pool.Pool) error {
	h, err := protocol.NewMultiHandler(cmp.PresignOnline(c, preSignature, m, pl), nil)
	if err != nil {
		return err
	}
	HandlerLoop(c.ID, h, n)

	signResult, err := h.Result()
	if err != nil {
		return err
	}
	signature := signResult.(*ecdsa.Signature)
	if !signature.Verify(c.PublicPoint(), m) {
		return errors.New("failed to verify cmp signature")
	}
	return nil
}

/*func main() {

	pl := pool.NewPool(0) // use the maximum number of threads.
	defer pl.TearDown()   // destroy the pool once the protocol is done.

	handler, err := protocol.NewMultiHandler(cmp.Keygen(group, selfID, participants, threshold, pl), sessionID)
	if err != nil {
		// the handler was not able to start the protocol, most likely due to incorrect configuration.
	}

	// runProtocol blocks until the protocol succeeds or aborts
	runProtocol(handler)

	// obtain the final result, or a possible error
	result, err := handler.Result()
	protocolError := protocol.Error{}
	if errors.As(err, protocolError) {
		// get the list of culprits by calling protocolError.Culprits
	}
	// if the error is nil, then we can cast the result to the expected return type
	config := result.(*cmp.Config)
}

func runProtocol(handler *protocol.Handler) {
	// Message handling loop
	for {
		select {

		// Message to be sent to other participants
		case msgOut, ok := <-(*handler).Listen():
			// a closed channel indicates that the protocol has finished executing
			if !ok {
				return
			}
			if msgOut.Broadcast {
				// ensure this message is reliably broadcast
			}
			for _, id := range participants {
				if msgOut.IsFor(id) {
					// send the message to `id`
				}
			}

		// Incoming message
		case msgIn := <-Receive():
			if !(*handler).CanAccept(msg) {
				// basic header validation failed, the message may be intended for a different protocol execution.
				continue
			}
			(*handler).Update(msgIn)
		}
	}
}*/
