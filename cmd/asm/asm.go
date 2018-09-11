package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"i10r.io/protocol/txvm/asm"
)

func main() {
	doDisasm := flag.Bool("d", false, "disassemble")
	flag.Parse()
	if *doDisasm {
		disassemble()
	} else {
		assemble()
	}
}

func assemble() {
	src, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	res, err := asm.Assemble(string(src))
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(res)
}

func disassemble() {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	dis, err := asm.Disassemble(b)
	if err != nil {
		panic(err)
	}
	fmt.Println(dis)
}
