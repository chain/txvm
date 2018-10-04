package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"i10r.io/sequence/protocol/bc"
	"i10r.io/sequence/protocol/state"
)

func main() {
	var (
		blockFile = flag.String("block", "", "filename containing block to apply")
		stateFile = flag.String("state", "", "filename containing previous state")
	)

	flag.Parse()

	if *blockFile == "-" && *stateFile == "-" {
		log.Fatal("only one of -block and -state may be -")
	}

	blockInp := getReader(*blockFile)
	if blockInp != nil {
		defer blockInp.Close()
	}

	stateInp := getReader(*stateFile)
	if stateInp != nil {
		defer stateInp.Close()
	}

	var snapshot *state.Snapshot

	if stateInp == nil {
		snapshot = state.Empty()
	} else {
		b, err := ioutil.ReadAll(stateInp)
		must(err)
		snapshot = new(state.Snapshot)
		err = snapshot.FromBytes(b)
		must(err)
	}

	if blockInp != nil {
		b, err := ioutil.ReadAll(blockInp)
		must(err)
		var block bc.Block
		err = block.FromBytes(b)
		must(err)
		err = snapshot.ApplyBlock(block.UnsignedBlock)
		if err != nil {
			log.Fatal(err)
		}
	}

	b, err := snapshot.Bytes()
	must(err)
	os.Stdout.Write(b)
}

func getReader(arg string) io.ReadCloser {
	switch arg {
	case "":
		return nil
	case "-":
		return os.Stdin
	default:
		f, err := os.Open(arg)
		must(err)
		return f
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
