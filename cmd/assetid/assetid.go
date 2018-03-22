package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/chain/txvm/standard"
	"github.com/chain/txvm/crypto/ed25519"
)

func main() {
	tagHex := flag.String("t", "", "hex asset tag")
	help := flag.Bool("h", false, "help")
	version := flag.Int("v", 2, "asset contract version")
	flag.Parse()
	if *help {
		usage(0)
	}
	tag, err := hex.DecodeString(*tagHex)
	must(err)
	if flag.NArg() < 2 {
		usage(1)
	}
	quorum, err := strconv.Atoi(flag.Arg(0))
	must(err)
	var pubkeys []ed25519.PublicKey
	for i := 1; i < flag.NArg(); i++ {
		pubkey, err := hex.DecodeString(flag.Arg(1))
		must(err)
		pubkeys = append(pubkeys, ed25519.PublicKey(pubkey))
	}
	assetID := standard.AssetID(*version, quorum, pubkeys, tag)
	_, err = os.Stdout.Write(assetID[:])
	must(err)
}

func usage(exitval int) {
	fmt.Println("Usage:")
	fmt.Printf("\t%s [-t taghex] quorum pubkey1hex pubkey2hex ...\n", os.Args[0])
	os.Exit(exitval)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
