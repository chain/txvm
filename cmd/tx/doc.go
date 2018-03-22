/*

Command tx parses and operates on Chain protocol transactions and
performs various operations on them.

Usage:

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

*/
package main
