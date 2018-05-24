/*

Command block parses Chain protocol blocks and performs various
operations on them.

Usage:

	block tx [-raw|-pretty] INDEX <BLOCK
	block header [-pretty] <BLOCK
	block validate [-prev PREVHEX] [-noprev] [-nosig] <BLOCK
	block new [-quorum QUORUM] [-time TIME] PUBKEYHEX PUBKEYHEX ... >BLOCK
	block sign -prev PREVHEX PRVHEX PRVHEX ... <BLOCK >BLOCK

	block hash <BLOCK_OR_HEADER

The tx subcommand causes block to extract and output the transaction
with the given index (zero-based). The default output contains only
the bytes of the transaction's program. With -raw the output is the
serialization of the txwitness triple (version, runlimit, and
program). With -pretty the output is a human-readable version of the
txwitness triple.

The header subcommand causes block to extract and output the block's
header. The default output is the serialized bytes of the raw
header. With -pretty the output is a human-readable version of the
block header, plus a count of the transactions in the block.

The validate subcommand causes block to validate the block. By default
the hex string representing the previous block's header must be given
as PREVHEX. If the block's height is 1, or if -noprev is given, then
PREVHEX may be omitted and those validation checks requiring a
previous block are skipped. If -nosig is given, the signatures in the
block are not checked against the predicate in the previous
block. Note, -noprev implies -nosig.

The new subcommand creates a new block with height 1 and no
transactions. Its predicate is given by the supplied PUBKEYs (which
must be hex-encoded ed25519 public keys) and the optional QUORUM
(which defaults to the number of pubkeys). Its timestamp is the
current time unless -time appears, in which case TIME must be a time
in RFC3339 format, e.g.:

	2006-01-02T15:04:05Z07:00

The sign subcommand adds signatures to a block using the given private
keys. PREVHEX gives the header of the previous block. The number of
PRVHEX arguments should equal the number of public keys in the
previous blockheader's NextPredicate field, and each private key
should correspond to the public key in the same position. Some of the
PRVHEX arguments may be the empty string, meaning no signature should
be added in the corresponding slot. It is an error to supply fewer
signatures than the quorum threshold specified in the previous
blockheader. Trailing empty-string arguments may be omitted. Note, any
signatures already present on the input block are removed before
producing the output block.

The hash subcommand accepts a block or a block header as input (such
as is produced by the output of the header subcommand). The output is
the hash of the block header that also serves as the ID of the block.

*/
package main
