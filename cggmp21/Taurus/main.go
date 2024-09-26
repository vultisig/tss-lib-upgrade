package main

import (
	"Taurus/network"
	"errors"
	"fmt"
	"os"
	"strconv"
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

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run main.go <threshold> <id> <address>  <party1:address1> [party2:address2] ...")
		fmt.Println("Example: go run main.go 2 Alice localhost:8080  Bob:localhost:8081 Charlie:localhost:8082")
		return
	}

	id := party.ID(os.Args[2])
	ids := party.IDSlice{id}
	address := os.Args[3]
	threshold, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Error: threshold must be an integer")
		return
	}

	parties := make(party.IDSlice, 0)
	addresses := make(map[party.ID]string)
	parties = append(parties, id)
	addresses[id] = address

	for _, arg := range os.Args[4:] {
		fmt.Printf("Processing argument: %s\n", arg)
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid party address format: %s\n", arg)
			continue
		}
		partyID := party.ID(parts[0])
		fmt.Printf("Parsed partyID: %s\n", partyID)
		if partyID != id {
			ids = append(ids, partyID)
		}
		parties = append(parties, partyID)
		addresses[partyID] = parts[1]
	}

	net := network.NewNetwork(id, address, parties, addresses)

	var input string
	for {
		fmt.Print("Type 'start' when everyone is connected: ")
		fmt.Scanln(&input)
		if input == "start" {
			break
		}
	}
	go net.ConnectToParties()

	fmt.Printf("Joined the network as %s\n", id)
	fmt.Println("Type 'quit' to exit")
	fmt.Println("Enter messages in the format: recipient message")

	/*ids := party.IDSlice{
		"benchmark-london-03",
		"benchmark-singapore-05",
		"benchmark-nyc-01",

		/*"benchmark-london-01",
		"benchmark-london-06",
		"benchmark-london-05",
		"benchmark-london-02",
		"benchmark-london-04",
		"benchmark-singapore-06",
		"benchmark-singapore-02",
		"benchmark-singapore-01",
		"benchmark-singapore-04",
		"benchmark-singapore-07",
		"benchmark-singapore-03",
		"benchmark-nyc-05",
		"benchmark-nyc-03",
		"benchmark-nyc-02",
		"benchmark-nyc-04",
		"benchmark-nyc-07",
		"benchmark-nyc-06",
	}*/

	messageToSign := []byte("hello")

	var wg sync.WaitGroup
	wg.Add(1)
	go func(id party.ID) {
		pl := pool.NewPool(0)
		fmt.Println("Starting protocol")
		defer pl.TearDown()
		if err := All(id, ids, threshold, messageToSign, net, &wg, pl); err != nil {
			fmt.Println(err)
		}
	}(id)
	wg.Wait()
}

func All(id party.ID, ids party.IDSlice, threshold int, message []byte, n *network.Network, wg *sync.WaitGroup, pl *pool.Pool) error {

	fmt.Println("Starting CMP Keygen")
	startKeygen := time.Now()
	// CMP KEYGEN
	keygenConfig, err := CMPKeygen(id, ids, threshold, n, pl)
	if err != nil {
		return err
	}

	durationKeygen := time.Since(startKeygen)
	fmt.Printf("CMP Keygen completed in %s\n", durationKeygen)
	signers := ids[:threshold+1]
	/*if !signers.Contains(id) {
		fmt.Println("signers: ", signers)
		fmt.Println("ids: ", ids)
		fmt.Println("id: ", id)
		fmt.Println("do not contain id")
		n.Quit(id)
		return err
	}*/

	fmt.Println("Starting CMP PRESIGN")
	startPresign := time.Now()

	// CMP PRESIGN
	preSignature, err := CMPPreSign(keygenConfig, signers, n, pl)
	if err != nil {
		return err
	}
	durationPresign := time.Since(startPresign)
	fmt.Printf("CMP Presign completed in %s\n", durationPresign)
	startPresignOnline := time.Now()

	// CMP PRESIGN ONLINE
	err = CMPPreSignOnline(keygenConfig, preSignature, message, n, pl)
	if err != nil {
		return err
	}
	durationPresignOnline := time.Since(startPresignOnline)
	fmt.Printf("CMP Presign Online completed in %s\n", durationPresignOnline)
	return nil
}

func CMPKeygen(id party.ID, ids party.IDSlice, threshold int, n *network.Network, pl *pool.Pool) (*cmp.Config, error) {
	fmt.Printf("Starting CMPKeygen with id: %s, ids: %v, threshold: %d\n", id, ids, threshold)
	h, err := protocol.NewMultiHandler(cmp.Keygen(curve.Secp256k1{}, id, ids, threshold, pl), nil)
	if err != nil {
		fmt.Printf("Error creating MultiHandler: %v\n", err)
		return nil, err
	}
	network.HandlerLoop(id, h, n)
	fmt.Println("HandlerLoop completed")

	r, err := h.Result()
	if err != nil {
		fmt.Printf("Error getting result: %v\n", err)
		return nil, err
	}
	fmt.Println("Result obtained successfully")

	return r.(*cmp.Config), nil
}

func CMPPreSign(c *cmp.Config, signers party.IDSlice, n *network.Network, pl *pool.Pool) (*ecdsa.PreSignature, error) {
	h, err := protocol.NewMultiHandler(cmp.Presign(c, signers, pl), nil)
	if err != nil {
		return nil, err
	}
	network.HandlerLoop(c.ID, h, n)
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
	network.HandlerLoop(c.ID, h, n)
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
