package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/chain/txvm/crypto/ed25519"
	chainjson "github.com/chain/txvm/encoding/json"
	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/txbuilder"
	"github.com/chain/txvm/protocol/txbuilder/txresult"
	"github.com/chain/txvm/protocol/txvm"
	"github.com/chain/txvm/protocol/txvm/asm"
)

var args []string

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	subcommand := os.Args[1]
	args = os.Args[2:]

	switch subcommand {
	case "id":
		prog, version, runlimit := getWitness()
		vm, err := txvm.Validate(prog, version, runlimit, txvm.StopAfterFinalize)
		must(err)
		if !vm.Finalized {
			panic(txvm.ErrUnfinalized)
		}
		os.Stdout.Write(vm.TxID[:])

	case "validate":
		prog, version, runlimit := getWitness()
		_, err := txvm.Validate(prog, version, runlimit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	case "trace":
		prog, version, runlimit := getWitness()
		txvm.Validate(prog, version, runlimit, txvm.Trace(os.Stdout))

	case "log":
		prog, version, runlimit := getWitness()
		_, err := txvm.Validate(prog, version, runlimit, txvm.StopAfterFinalize, txvm.OnFinalize(func(vm *txvm.VM) {
			for _, tuple := range vm.Log {
				dis, err := asm.Disassemble(txvm.Encode(tuple))
				must(err)
				fmt.Println(dis)
			}
		}))
		must(err)

	case "result":
		prog, version, runlimit := getWitness()
		tx, err := bc.NewTx(prog, version, runlimit)
		must(err)
		result := txresult.New(tx)
		for i, iss := range tx.Issuances {
			if i == 0 {
				fmt.Println("Issuances:")
			}
			var refdata []byte
			meta := result.Issuances[i]
			if meta != nil {
				refdata = meta.RefData
			}
			fmt.Printf("  assetID %x amount %d anchor %x refdata [%x]\n", iss.AssetID.Bytes(), iss.Amount, iss.Anchor, refdata)
		}
		for i, ret := range tx.Retirements {
			if i == 0 {
				fmt.Println("Retirements:")
			}
			var refdata []byte
			meta := result.Retirements[i]
			if meta != nil {
				refdata = meta.RefData
			}
			fmt.Printf("  assetID %x amount %d anchor %x refdata [%x]\n", ret.AssetID.Bytes(), ret.Amount, ret.Anchor, refdata)
		}
		for i, inp := range tx.Inputs {
			if i == 0 {
				fmt.Println("Inputs:")
			}
			fmt.Printf("  contractID %x seed %x program [%x]", inp.ID.Bytes(), inp.Seed.Bytes(), inp.Program)
			if meta := result.Inputs[i]; meta != nil {
				fmt.Printf(" refdata [%x]", meta.RefData)
				if value := meta.Value; value != nil {
					fmt.Printf(" assetID %x amount %d anchor %x", value.AssetID.Bytes(), value.Amount, value.Anchor)
				}
			}
			fmt.Println()
		}
		for i, out := range tx.Outputs {
			if i == 0 {
				fmt.Println("Outputs:")
			}
			fmt.Printf("  contractID %x seed %x program [%x]", out.ID.Bytes(), out.Seed.Bytes(), out.Program)
			if meta := result.Outputs[i]; meta != nil {
				var pkstrs []string
				for _, p := range meta.Pubkeys {
					pkstrs = append(pkstrs, hex.EncodeToString(p))
				}
				fmt.Printf(" pubkeys [%s] refdata [%x] tokentags [%x]", strings.Join(pkstrs, " "), meta.RefData, meta.TokenTags)
				if value := meta.Value; value != nil {
					fmt.Printf(" assetID %x amount %d anchor %x", value.AssetID.Bytes(), value.Amount, value.Anchor)
				}
			}
			fmt.Println()
		}

	case "build":
		var (
			txfs      flag.FlagSet
			ttl       time.Duration
			txtagsStr string
		)
		txfs.DurationVar(&ttl, "ttl", time.Hour, "ttl")
		txfs.StringVar(&txtagsStr, "tags", "", "tx tags (as hex or JSON object)")

		err := txfs.Parse(args)
		must(err)
		args = txfs.Args()
		var txtags []byte
		if len(txtagsStr) > 0 {
			var m chainjson.Map
			err = m.UnmarshalJSON([]byte(txtagsStr))
			if err == nil {
				txtags, _ = m.MarshalJSON()
			} else {
				txtags, err = hex.DecodeString(txtagsStr)
			}
			must(err)
		}

		tpl := txbuilder.NewTemplate(time.Now().Add(ttl), txtags)
		for len(args) > 0 {
			var (
				fs         flag.FlagSet
				amount     int64
				refdataStr string
			)
			fs.Int64Var(&amount, "amount", 0, "amount")
			fs.StringVar(&refdataStr, "refdata", "", "refdata (as hex or JSON object)")

			buildcmd := args[0]
			args = args[1:]

			switch buildcmd {
			case "issue":
				var (
					version         int
					blockchainIDStr string
					assetTagStr     string
					quorum          int
					prvStrs         string
					pubStrs         string
					nonceStr        string
				)
				fs.IntVar(&version, "version", 2, "asset contract version")
				fs.StringVar(&blockchainIDStr, "blockchain", "", "blockchain ID (as hex)")
				fs.StringVar(&assetTagStr, "tag", "", "asset tag (as hex or JSON object)")
				fs.IntVar(&quorum, "quorum", 1, "quorum")
				fs.StringVar(&prvStrs, "prv", "", "private keys (as hex, space-separated)")
				fs.StringVar(&pubStrs, "pub", "", "public keys (as hex, space-separated)")
				fs.StringVar(&nonceStr, "nonce", "", "nonce (as hex)")

				err = fs.Parse(args)
				must(err)
				args = fs.Args()
				var refdata []byte
				if len(refdataStr) > 0 {
					var m chainjson.Map
					err = m.UnmarshalJSON([]byte(refdataStr))
					if err == nil {
						refdata, _ = m.MarshalJSON()
					} else {
						refdata, err = hex.DecodeString(refdataStr)
					}
					must(err)
				}

				var (
					blockchainID []byte
					assetTag     []byte
					prvs         [][]byte
					pubs         []ed25519.PublicKey
					nonce        []byte
				)
				if len(blockchainIDStr) > 0 {
					blockchainID, err = hex.DecodeString(blockchainIDStr)
					must(err)
				}
				if len(assetTagStr) > 0 {
					var m chainjson.Map
					err = m.UnmarshalJSON([]byte(assetTagStr))
					if err == nil {
						assetTag, _ = m.MarshalJSON()
					} else {
						assetTag, err = hex.DecodeString(assetTagStr)
					}
					must(err)
				}
				for _, prvStr := range strings.Fields(prvStrs) {
					prv, err := hex.DecodeString(prvStr)
					must(err)
					prvs = append(prvs, prv)
				}
				for _, pubStr := range strings.Fields(pubStrs) {
					pub, err := hex.DecodeString(pubStr)
					must(err)
					pubs = append(pubs, ed25519.PublicKey(pub))
				}
				if len(nonceStr) > 0 {
					nonce, err = hex.DecodeString(nonceStr)
					must(err)
				}
				tpl.AddIssuance(version, blockchainID, assetTag, quorum, prvs, nil, pubs, amount, refdata, nonce)

			case "input":
				var (
					version    int
					quorum     int
					prvStrs    string
					pubStrs    string
					assetIDStr string
					anchorStr  string
				)
				fs.IntVar(&version, "version", 2, "output [sic] contract version")
				fs.IntVar(&quorum, "quorum", 1, "quorum")
				fs.StringVar(&prvStrs, "prv", "", "private keys (as hex, space-separated)")
				fs.StringVar(&pubStrs, "pub", "", "public keys (as hex, space-separated)")
				fs.StringVar(&assetIDStr, "assetid", "", "asset ID (as hex)")
				fs.StringVar(&anchorStr, "anchor", "", "anchor (as hex)")

				err = fs.Parse(args)
				must(err)
				args = fs.Args()
				var refdata []byte
				if len(refdataStr) > 0 {
					var m chainjson.Map
					err = m.UnmarshalJSON([]byte(refdataStr))
					if err == nil {
						refdata, _ = m.MarshalJSON()
					} else {
						refdata, err = hex.DecodeString(refdataStr)
					}
					must(err)
				}

				var (
					prvs    [][]byte
					pubs    []ed25519.PublicKey
					anchor  []byte
					assetID bc.Hash
				)
				for _, prvStr := range strings.Fields(prvStrs) {
					prv, err := hex.DecodeString(prvStr)
					must(err)
					prvs = append(prvs, prv)
				}
				for _, pubStr := range strings.Fields(pubStrs) {
					pub, err := hex.DecodeString(pubStr)
					must(err)
					pubs = append(pubs, ed25519.PublicKey(pub))
				}
				err = assetID.UnmarshalText([]byte(assetIDStr))
				must(err)
				if len(anchorStr) > 0 {
					anchor, err = hex.DecodeString(anchorStr)
					must(err)
				}
				tpl.AddInput(quorum, prvs, nil, pubs, amount, assetID, anchor, refdata, version)

			case "output":
				var (
					quorum     int
					pubStrs    string
					assetIDStr string
					tagsStr    string
				)
				fs.IntVar(&quorum, "quorum", 1, "quorum")
				fs.StringVar(&pubStrs, "pub", "", "public keys (as hex, space-separated)")
				fs.StringVar(&assetIDStr, "assetid", "", "asset ID (as hex)")
				fs.StringVar(&tagsStr, "tags", "", "tags (as hex or JSON object)")

				err = fs.Parse(args)
				must(err)
				args = fs.Args()
				var refdata []byte
				if len(refdataStr) > 0 {
					var m chainjson.Map
					err = m.UnmarshalJSON([]byte(refdataStr))
					if err == nil {
						refdata, _ = m.MarshalJSON()
					} else {
						refdata, err = hex.DecodeString(refdataStr)
					}
					must(err)
				}

				var (
					pubs    []ed25519.PublicKey
					assetID bc.Hash
					tags    []byte
				)
				for _, pubStr := range strings.Fields(pubStrs) {
					pub, err := hex.DecodeString(pubStr)
					must(err)
					pubs = append(pubs, ed25519.PublicKey(pub))
				}
				err = assetID.UnmarshalText([]byte(assetIDStr))
				must(err)
				if len(tagsStr) > 0 {
					var m chainjson.Map
					err = m.UnmarshalJSON([]byte(tagsStr))
					if err == nil {
						tags, _ = m.MarshalJSON()
					} else {
						tags, err = hex.DecodeString(tagsStr)
					}
					must(err)
				}
				tpl.AddOutput(quorum, pubs, amount, assetID, refdata, tags)

			case "retire":
				var assetIDStr string
				fs.StringVar(&assetIDStr, "assetid", "", "asset ID (as hex)")

				err = fs.Parse(args)
				must(err)
				args = fs.Args()
				var refdata []byte
				if len(refdataStr) > 0 {
					var m chainjson.Map
					err = m.UnmarshalJSON([]byte(refdataStr))
					if err == nil {
						refdata, _ = m.MarshalJSON()
					} else {
						refdata, err = hex.DecodeString(refdataStr)
					}
					must(err)
				}

				var assetID bc.Hash
				err = assetID.UnmarshalText([]byte(assetIDStr))
				must(err)

				tpl.AddRetirement(amount, assetID, refdata)
			}
		}
		err = tpl.Sign(context.Background(), func(_ context.Context, msg []byte, prv []byte, _ [][]byte) ([]byte, error) {
			return ed25519.Sign(prv, msg), nil
		})
		must(err)
		tx, err := tpl.Tx()
		must(err)
		rawTx := &bc.RawTx{
			Version:  tx.Version,
			Runlimit: tx.Runlimit,
			Program:  tx.Program,
		}
		bits, err := proto.Marshal(rawTx)
		must(err)
		os.Stdout.Write(bits)

	default:
		usage()
	}
}

func getWitness() (prog []byte, version, runlimit int64) {
	var fs flag.FlagSet
	witness := fs.Bool("witness", false, "expect a witness tuple on stdin")
	fs.Int64Var(&runlimit, "runlimit", math.MaxInt64, "runlimit")
	fs.Int64Var(&version, "version", 3, "tx version")
	err := fs.Parse(args)
	must(err)
	args = fs.Args()

	inp, err := ioutil.ReadAll(os.Stdin)
	must(err)

	if *witness {
		var rawTx bc.RawTx
		err = proto.Unmarshal(inp, &rawTx)
		must(err)

		runlimit = rawTx.Runlimit
		version = rawTx.Version
		prog = rawTx.Program
	} else {
		prog = inp
	}

	return prog, version, runlimit
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:

	tx SUBCOMMAND ...args...

Available subcommands are: id, validate, trace, log, result, build.

All subcommands except build expect a transaction program on standard
input, assigning it a default version of 3 and a default runlimit of
2^63-1. The -runlimit and -version flags can override those default
values. These subcommands also accept a -witness flag tells tx to
expect a transaction witness tuple on standard input instead (such as
can be produced with the "block tx -raw" command, qv), which dictates
the version and runlimit.

The id subcommand causes tx to compute the transaction's ID and send
it to standard output. Errors in the transaction beyond the "finalize"
instruction are not detected.

The validate subcommand causes tx to validate the transaction. Exit
value 0 means the transaction is valid, non-zero means it is not.

The trace subcommand causes an execution trace of the tx to be sent to
standard output.

The log subcommand causes the transaction's log entries to be sent to
standard output in assembly-language syntax, one per line. Errors in
the transaction beyond the "finalize" instruction are not detected.

The result subcommand parses the transaction log for information
produced by "standard" issuance, retirement, input, and output
contracts and prints the information in human-readable form.

The build subcommand creates a transaction. It is used like this:

	tx build [-ttl TIME] [-tags TAGS] DIRECTIVE ...args... DIRECTIVE ...args...

where each DIRECTIVE is one of "issue," "input," "output," and
"retire." Each directive adds an entry to the transaction being
built. The -ttl flag specifies the transaction's time to live; its
format must be understood by Go's time.ParseDuration. The -tags flag
specifies a hex- or JSON-encoded set of tags for the transaction.

Each directive has its own set of arguments:

	issue:
		-version V       integer version of the asset contract to use
		-blockchain HEX  hex-encoded blockchain ID for unanchored issuances
		-tag TAG         hex- or JSON-encoded asset tag
		-quorum N        integer quorum
		-prv 'S1 S2 ...' hex-encoded, space-separated private keys for signing
		-pub 'P1 P2 ...' hex-encoded, space-separated public keys
		-amount N        integer amount to issue
		-refdata D       hex- or JSON-encoded reference data
		-nonce HEX       hex-encoded issuance nonce

	input:
		-quorum N        integer quorum
		-prv 'S1 S2 ...' hex-encoded, space-separated private keys for signing
		-pub 'P1 P2 ...' hex-encoded, space-separated public keys
		-amount N        integer amount to issue
		-assetid HEX     hex-encoded asset ID
		-anchor HEX      hex-encoded anchor
		-refdata D       hex- or JSON-encoded reference data
		-version V       integer version of the output [sic] contract to use

	output:
		-quorum N        integer quorum
		-pub 'P1 P2 ...' hex-encoded, space-separated public keys
		-amount N        integer amount to issue
		-assetid HEX     hex-encoded asset ID
		-refdata D       hex- or JSON-encoded reference data
		-tags T          hex- or JSON-encoded tags

	retire:
		-amount N        integer amount to issue
		-assetid HEX     hex-encoded asset ID
		-refdata D       hex- or JSON-encoded reference data
`)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
