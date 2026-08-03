// Command keytool generates or derives an OpenChat validator identity:
// given (or freshly generating) a 32-byte Ed25519 seed, it prints the
// node's address (VALIDATOR_SET entry) and libp2p peer ID (the trailing
// `/p2p/<id>` component of a bootstrap multiaddr) — the two things you
// need to wire multiple validators together, without hand-deriving them.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"openchat/internal/p2p"
)

func main() {
	seedHex := flag.String("seed", "", "existing 32-byte hex Ed25519 seed to derive from (omit to generate a fresh random one)")
	flag.Parse()

	var seed []byte
	if *seedHex != "" {
		b, err := hex.DecodeString(*seedHex)
		if err != nil || len(b) != ed25519.SeedSize {
			fmt.Fprintf(os.Stderr, "keytool: -seed must be a %d-byte hex string\n", ed25519.SeedSize)
			os.Exit(1)
		}
		seed = b
	} else {
		seed = make([]byte, ed25519.SeedSize)
		if _, err := rand.Read(seed); err != nil {
			fmt.Fprintln(os.Stderr, "keytool: generate seed:", err)
			os.Exit(1)
		}
	}

	priv := ed25519.NewKeyFromSeed(seed)
	address := hex.EncodeToString(priv.Public().(ed25519.PublicKey))

	peerID, err := p2p.PeerIDFromSeed(seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keytool: derive libp2p peer id:", err)
		os.Exit(1)
	}

	fmt.Println("seed (VALIDATOR_PRIVATE_KEY_SEED):", hex.EncodeToString(seed))
	fmt.Println("address (goes in VALIDATOR_SET):  ", address)
	fmt.Println("libp2p peer id (goes after /p2p/):", peerID.String())
	fmt.Println()
	fmt.Println("example bootstrap multiaddr once this node is reachable at host:port:")
	fmt.Printf("  /dns4/<host>/tcp/<p2p-port>/p2p/%s\n", peerID.String())
}
