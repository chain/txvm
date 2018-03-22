# TxVM

This repository contains the source code for TxVM and related components of the Chain Protocol. It also contains command line tools for creating, manipulating, and evaluating TxVM programs, transactions, blocks, and blockchains.

TxVM is a new model for blockchain transactions. TxVM seeks to achieve the expressiveness and flexibility of an imperative contract model such as Ethereum's while maintaining the efficiency, safety, and scalability of a declarative transaction model such as Bitcoin's.

* [TxVM White Paper](whitepaper/whitepaper.pdf)
* [TxVM Specification](specifications/txvm.md)
* [Documentation for TxVM command line tools](https://godoc.org/github.com/chain/txvm)

## Installation

```sh
go get github.com/chain/txvm/...
```

## Testing

```sh
go test -race -cover github.com/chain/txvm/...
```

## Usage

In TxVM, each transaction is a single program, whose execution produces a log of desired (and authorized) effects to be applied to the blockchain state.

Here's the bytecode for an example transaction:

```
> BIGTXPROGRAM=90025f2e7fda5ddf057de3cf8db186e619915b88f37f797a9be1a5a79195533741f8bec4a201542e012e5f2e0f2e7f00000000000000000000000000000000000000000000000000000000000000002e5f2e65c4e3bda5a42c202eb3012d512709412d522d012a30010241522d2d2d2d51042b2d51052b035458332d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a5040524244484348432d2d00325f2e5f2e012a0a322e7fc9a0c09b8c39ce445a7225e4f8c418392a9a0e94fa8eb6a911afcad13c4b26b501542e012e9f012d2d2d2d3c2d3c95012d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a50405242444748435f2e5f2e2e7f84efd963fd680e38c8255f0c00542aa6a9d791b5fbb4fe2b9d6829bedef0e2b001542e022e9f012d2d2d2d3c2d3c95012d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a50405242444748430065c4e3bda5a42c204d5f3c3f9f01981af94a30116a08c0d26c10ec545f8eddb0cee5a6d78f31da0a73cf0c80c3b7d3cdda88b7eb022c341277dd54dd96cfd04fcf0bcea2d16dd0f2d62b37b814092e83013e7f26ff39366e4a29b97c604f757e0783e652686cb6053e51961aba2f61e07b14a750402e43
```

To inspect the assembly language for this transaction, you can pass it to the disassembler, `asm -d`:

```
> echo $BIGTXPROGRAM | hex -d | asm -d
```

To compute its transaction log:

```
> echo $BIGTXPROGRAM | hex -d | tx log
```

For more on the commands `txvm` makes available, you can check out the [documentation](https://godoc.org/github.com/chain/txvm).

## Contributing

Chain has adopted the code of conduct defined by the Contributor
Covenant. It can be read in full [here](CODE_OF_CONDUCT.md).

Contributors must have signed the [Contributor License Agreement](https://cla-assistant.io/chain/txvm).

## License

The code in this repository is licensed under version 2.0 of [the Apache License](LICENSE).
