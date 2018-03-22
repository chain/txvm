/*

Command bcstate reads and writes blockchain state, optionally applying
a block.

Usage:

	bcstate [-block BLOCKFILE] [-state STATEFILE] >NEWSTATE

BLOCKFILE and STATEFILE are the names of files containing a block and
a previous state, respectively. Either (but not both) may be - to read
from standard input.

In normal usage, a state is read from STATEFILE, a block from
BLOCKFILE is applied to it, and the resulting updated state is written
to standard output. If STATEFILE is not specified then a new blank
state snapshot is used. If BLOCKFILE is not specified then the input
state is simply copied to standard output.

*/
package main
