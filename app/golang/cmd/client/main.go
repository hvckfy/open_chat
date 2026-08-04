// Command client is the reference CLI for OpenChat end users: generate a
// wallet, send an E2EE message, or listen for incoming ones. The actual
// logic lives in pkg/client so the identical code can be embedded in a
// mobile app (e.g. via gomobile bind ./pkg/client).
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"openchat/pkg/client"
	"openchat/pkg/tlsutil"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "keygen":
		cmdKeygen(os.Args[2:])
	case "address":
		cmdAddress(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	case "listen":
		cmdListen(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `openchat client — decentralized E2EE messenger CLI

Usage:
  client keygen                                  generate a new wallet (mnemonic + address)
  client address -mnemonic "..."                 derive and print the address for a mnemonic
  client send    -mnemonic "..." -to ADDR -to-x25519 PUB -text "hi"   [network flags]
  client listen  -mnemonic "..."                                     [network flags]

Network flags (send/listen):
  -bootstrap host:port,host:port   override the built-in bootstrap gateway list
  -ca path/to/ca.pem               trust a private network CA instead of the system store
  -insecure                        skip TLS verification (LOCAL/DEV ONLY)
`)
}

func cmdKeygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	passphrase := fs.String("passphrase", "", "optional BIP-39 passphrase (25th word)")
	fs.Parse(args)

	w, err := client.NewWallet("", *passphrase)
	fatalIf(err)

	fmt.Println("=== SAVE THIS MNEMONIC SOMEWHERE SAFE. IT IS THE ONLY WAY TO RECOVER YOUR ACCOUNT. ===")
	fmt.Println(w.Keys.Mnemonic)
	fmt.Println()
	fmt.Println("address (share this — it's your phone number):        ", w.Address())
	fmt.Println("x25519 pubkey (share this — needed to receive E2EE):  ", w.Keys.EncryptionPublicHex())
}

func cmdAddress(args []string) {
	fs := flag.NewFlagSet("address", flag.ExitOnError)
	mnemonic := fs.String("mnemonic", "", "BIP-39 mnemonic")
	passphrase := fs.String("passphrase", "", "optional BIP-39 passphrase")
	fs.Parse(args)

	m := requireMnemonic(*mnemonic)
	w, err := client.NewWallet(m, *passphrase)
	fatalIf(err)
	fmt.Println("address:      ", w.Address())
	fmt.Println("x25519 pubkey:", w.Keys.EncryptionPublicHex())
}

func cmdSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	mnemonic := fs.String("mnemonic", "", "BIP-39 mnemonic")
	passphrase := fs.String("passphrase", "", "optional BIP-39 passphrase")
	to := fs.String("to", "", "recipient address (hex ed25519 pubkey)")
	toX25519 := fs.String("to-x25519", "", "recipient X25519 pubkey (hex)")
	text := fs.String("text", "", "message text")
	bootstrap := fs.String("bootstrap", "", "comma-separated gateway host:port list")
	ca := fs.String("ca", "", "path to network CA cert (PEM)")
	insecure := fs.Bool("insecure", false, "skip TLS verification (dev only)")
	fs.Parse(args)

	if *to == "" || *toX25519 == "" || *text == "" {
		fatalIf(fmt.Errorf("send: -to, -to-x25519 and -text are required"))
	}

	w, err := client.NewWallet(requireMnemonic(*mnemonic), *passphrase)
	fatalIf(err)

	recipientPub, err := client.DecodeRecipientX25519(*toX25519)
	fatalIf(err)

	c := newClient(*bootstrap, *ca, *insecure)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	txHash, err := c.SendMessage(ctx, w, *to, recipientPub, []byte(*text))
	fatalIf(err)
	fmt.Println("sent. tx_hash =", txHash)
}

func cmdListen(args []string) {
	fs := flag.NewFlagSet("listen", flag.ExitOnError)
	mnemonic := fs.String("mnemonic", "", "BIP-39 mnemonic")
	passphrase := fs.String("passphrase", "", "optional BIP-39 passphrase")
	bootstrap := fs.String("bootstrap", "", "comma-separated gateway host:port list")
	ca := fs.String("ca", "", "path to network CA cert (PEM)")
	insecure := fs.Bool("insecure", false, "skip TLS verification (dev only)")
	fs.Parse(args)

	w, err := client.NewWallet(requireMnemonic(*mnemonic), *passphrase)
	fatalIf(err)

	c := newClient(*bootstrap, *ca, *insecure)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("listening for messages to", w.Address(), "(ctrl-c to quit)")
	err = c.Listen(ctx, w, func(m client.IncomingMessage) bool {
		fmt.Printf("\n[block %d] from %s:\n  %s\n", m.BlockHeight, m.From, string(m.Plaintext))
		return true
	})
	if err != nil && ctx.Err() == nil {
		fatalIf(err)
	}
}

func newClient(bootstrapCSV, caFile string, insecure bool) *client.Client {
	tlsCreds, err := tlsutil.ClientTLS(caFile, insecure)
	fatalIf(err)

	var bootstrap []string
	if bootstrapCSV != "" {
		bootstrap = strings.Split(bootstrapCSV, ",")
	}

	c := client.New(bootstrap, tlsCreds)
	c.OnEvent = func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "[net] "+format+"\n", args...)
	}
	return c
}

func requireMnemonic(m string) string {
	if m != "" {
		return m
	}
	fmt.Fprint(os.Stderr, "enter mnemonic: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
