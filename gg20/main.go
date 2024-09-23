package main

import (
	"fmt"
	"math/big"
	"time"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func main() {
	// Set up parameters
	threshold := 2
	participants := 3

	// Create party IDs
	var partyIDs tss.SortedPartyIDs
	for i := 0; i < participants; i++ {
		partyIDs = append(partyIDs, tss.NewPartyID(fmt.Sprintf("%d", i), "", big.NewInt(int64(i))))
	}

	// Set up parameters for key generation
	params := tss.NewParameters(tss.Edwards(), partyIDs, threshold, participants)

	// Create channels for communication
	outCh := make(chan tss.Message, participants)
	endCh := make(chan keygen.LocalPartySaveData, participants)

	// Start local parties
	parties := make([]*keygen.LocalParty, 0, participants)
	for i := 0; i < participants; i++ {
		P := keygen.NewLocalParty(params, outCh, endCh).(*keygen.LocalParty)
		parties = append(parties, P)
		go func(P *keygen.LocalParty) {
			if err := P.Start(); err != nil {
				fmt.Printf("Error starting party: %v\n", err)
			}
		}(P)
	}

	// Simulate message passing between parties
	go func() {
		for msg := range outCh {
			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index != msg.GetFrom().Index {
						P.UpdateFromBytes(msg.GetWire(), msg.GetFrom(), msg.IsBroadcast())
					}
				}
			} else {
				for _, P := range parties {
					if P.PartyID().Index == dest[0].Index {
						P.UpdateFromBytes(msg.GetWire(), msg.GetFrom(), msg.IsBroadcast())
						break
					}
				}
			}
		}
	}()

	// Wait for key generation to complete
	keygenData := make([]keygen.LocalPartySaveData, 0, participants)
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(30 * time.Second)
		timeout <- true
	}()

	for {
		select {
		case save := <-endCh:
			keygenData = append(keygenData, save)
			if len(keygenData) == participants {
				fmt.Println("Key generation completed successfully!")
				for i, save := range keygenData {
					fmt.Printf("Party %d public key: %x\n", i, save.ECDSAPub.X())
				}
				return
			}
		case <-timeout:
			fmt.Println("Key generation timed out")
			return
		}
	}
}

func main() {
	allParties := []*party{
		NewParty(1, logger("pA", t.Name())),
		NewParty(2, logger("pB", t.Name())),
		NewParty(3, logger("pC", t.Name())),
	}

	/*benchmarks := []struct {
		threshold  int
		numParties int
	}{
		{2, 3}, /*{2, 4}, {3, 4}, {2, 5}, {3, 5}, {4, 5},
		{2, 6}, {3, 6}, {4, 6}, {5, 6}, {2, 7},{11, 17}, {11, 20}, {12, 20}, {13, 20}, {14, 20}
	}*/

	numParties := 3
	threshold := 2
	numRuns := 1

	for i := 0; i < numRuns; i++ {
		parties := parties(allParties[:numParties])

		parties.init(senders(parties), threshold)

		// DKG
		shares, err := parties.keygen()
		//assert.NoError(t, err)

		parties.init(senders(parties), threshold)
		parties.setShareData(shares)

		// Signing
		msgToSign := []byte("bla bla")
		sigs, err := parties.sign(digest(msgToSign))
		//assert.NoError(t, err)

		// Verification (only done once per benchmark for simplicity)
		if i == 0 {
			sigSet := make(map[string]struct{})
			for _, s := range sigs {
				sigSet[string(s)] = struct{}{}
			}
			//assert.Len(t, sigSet, 1)

			parties[0].TPubKey()
			//assert.NoError(t, err)
			//assert.True(t, ecdsa.VerifyASN1(pk, digest(msgToSign), sigs[0]))
		}
	}

}
