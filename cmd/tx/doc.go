/*

Command tx parses and operates on Chain protocol transactions and
performs various operations on them.

Usage:

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

See example.md for an extended example of creating realistic
blockchain data.

*/
package main
