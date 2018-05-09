package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/txbuilder/txresult"
	"github.com/chain/txvm/protocol/txvm"
	"github.com/chain/txvm/protocol/txvm/asm"
)

func main() {
	var (
		witness  = flag.Bool("witness", false, "expect a witness tuple on stdin")
		runlimit = flag.Int64("runlimit", math.MaxInt64, "runlimit")
		version  = flag.Int64("version", 3, "tx version")
	)

	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}
	subcommand := flag.Arg(0)

	inp, err := ioutil.ReadAll(os.Stdin)
	must(err)

	prog := inp

	if *witness {
		var rawTx bc.RawTx
		err = proto.Unmarshal(inp, &rawTx)
		must(err)

		runlimit = &rawTx.Runlimit
		version = &rawTx.Version
		prog = rawTx.Program
	}

	switch subcommand {
	case "id":
		vm, err := txvm.Validate(prog, *version, *runlimit, txvm.StopAfterFinalize)
		must(err)
		if !vm.Finalized {
			panic(txvm.ErrUnfinalized)
		}
		os.Stdout.Write(vm.TxID[:])

	case "validate":
		_, err := txvm.Validate(prog, *version, *runlimit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	case "trace":
		txvm.Validate(prog, *version, *runlimit, txvm.Trace(os.Stdout))

	case "log":
		_, err = txvm.Validate(prog, *version, *runlimit, txvm.StopAfterFinalize, txvm.OnFinalize(func(vm *txvm.VM) {
			for _, tuple := range vm.Log {
				dis, err := asm.Disassemble(txvm.Encode(tuple))
				must(err)
				fmt.Println(dis)
			}
		}))
		must(err)

	case "result":
		tx, err := bc.NewTx(prog, *version, *runlimit)
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

	default:
		usage()
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:

	tx [-witness] [-runlimit LIMIT] [-version VERSION] SUBCOMMAND

Available subcommands are: id, validate, trace, log, result.

By default, tx expects a transaction program on standard input,
assigning it a default version of 3 and a default runlimit of
2^63-1. The -runlimit and -version flags can override those default
values. The -witness flag tells tx to expect a transaction witness
tuple on standard input instead (such as can be produced with the
"block tx -raw" command, qv).

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
`)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
