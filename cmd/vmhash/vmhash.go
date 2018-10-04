package main

import (
	"io/ioutil"
	"os"

	"i10r.io/sequence/protocol/txvm"
)

func main() {
	if len(os.Args) < 2 {
		panic("usage: vmhash funcname <input")
	}
	funcname := os.Args[1]
	inp, err := ioutil.ReadAll(os.Stdin)
	must(err)
	h := txvm.VMHash(funcname, inp)
	os.Stdout.Write(h[:])
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
