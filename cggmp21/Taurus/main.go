package main

import (
	"Taurus/network"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/taurusgroup/multi-party-sig/pkg/ecdsa"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/pool"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
	"github.com/taurusgroup/multi-party-sig/protocols/cmp"
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

	parties = append(parties, id)
	addresses[id] = address

	net := network.NewNetwork(id, address, parties, addresses)

	fmt.Printf("Joined the network as %s\n", id)
	fmt.Println("Type 'quit' to exit")
	fmt.Println("Enter messages in the format: recipient message")

	ids := party.IDSlice{"Alice", "Bob", "Charlie"}
	threshold := 2
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

	// CMP PRESIGN
	preSignature, err := CMPPreSign(keygenConfig, signers, n, pl)
	if err != nil {
		return err
	}

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
	network.HandlerLoop(id, h, n)
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
