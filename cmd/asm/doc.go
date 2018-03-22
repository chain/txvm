/*

Command asm assembles and disassembles

Usage:

	asm [-d] <program

By default, asm assembles a binary code from a TxVM assembly language.

Flag -d inverts the behavior: the binary code is read from stdin,
and the TxVM assembly is printed to stdout.

Examples:

	$ echo "[1 verify] contract call" | asm | hex
	6101303833

	$ echo "6101303833" | hex -d | asm -d
	[1 verify] contract call

*/
package main
