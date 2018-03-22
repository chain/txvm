# Blockchain specification

* [Introduction](#introduction)
* [General requirements](#general-requirements)
    * [Interfaces](#interfaces)
    * [Optimization](#optimization)
    * [Serializability](#serializability)
* [Definitions](#definitions)
    * [Blockchain state](#blockchain-state)
    * [Block header](#block-header)
    * [Block](#block)
    * [Block ID](#block-id)
    * [Authentication predicate](#authentication-predicate)
    * [Transactions merkle root](#transactions-merkle-root)
    * [Transaction witness commitment](#transaction-witness-commitment)
    * [Contracts merkle root](#contracts-merkle-root)
    * [Nonces merkle root](#nonces-merkle-root)
    * [Nonce commitment](#nonce-commitment)
    * [Merkle root](#merkle-root)
    * [Merkle binary tree](#merkle-binary-tree)
    * [Merkle patricia tree](#merkle-patricia-tree)
* [Validation procedures](#validation-procedures)
    * [Validate block](#validate-block)
    * [Join new network](#join-new-network)
    * [Join existing network](#join-existing-network)
    * [Make initial block](#make-initial-block)
    * [Apply block](#apply-block)
    * [Apply transaction log](#apply-transaction-log)

## Introduction

This is the specification of the Chain Protocol blockchain network
state replication protocol, version 3.

Data structures are defined in terms of [TxVM](txvm.md) types to allow
efficient introspection of the blockchain.  Each block in the chain is
authenticated using an
[authentication predicate](#authentication-predicate) that checks
signatures in the block against the public keys specified in the
previous block.

Each node in the network maintains its view of the blockchain’s
[state](#blockchain-state).  The
[validation procedures](#validation-procedures) below describe the
rules for updating a node’s state. Some of the algorithms are used
only by other algorithms defined here. Others are entry points,
triggered by network activity or user input.

## General requirements

### Interfaces

Each algorithm under [validation procedures](#validation-procedures)
specifies its interface to the outside world under the labels
*inputs*, *outputs*, *references*, and *affects*.

* *Input* is data that must be provided to the algorithm by its
  invoking context (either another algorithm or a user).
* *Output* is data that is returned from the algorithm to its invoking
  context.
* *References* lists elements of the persistent state that might be
  read.
* *Affects* lists elements of the persistent state that can be
  modified.

### Optimization

A conforming implementation must behave as if it is following the
algorithms described below. However, for the sake of efficiency or
convenience, it is permitted to take other actions instead as long as
they yield the same result. For example, a conforming implementation
might “memoize” the result of a computation rather than recomputing it
multiple times, or it might perform the steps of an algorithm in a
different but equivalent order.

### Serializability

A conforming implementation must be serializable with respect to these
algorithms. That is, it can execute them concurrently in parallel, but
it must produce the same output and side effects as if they had run
serially in some order.

This requirement also implies that all side effects together must be
atomic for each algorithm.


## Definitions

### Blockchain state

All nodes store a *current blockchain state*, which can be replaced
with a new blockchain state.

A **blockchain state** contains:

0. `initialheader`: The initial [block header](#block-header) (at
   height=1).
1. `header`: The latest [block header](#block-header) (“the tip”).
2. `refids`: A subset of contiguous block IDs ending with the ID of
   the current `header`. The size of this set is equal to
   `block.header.refscount` (which may be 0). This is the set of block
   IDs that [nonces](txvm.md#nonce) can refer to.
3. `contracts`: A set of [snapshot IDs](txvm.md#snapshot-id)
   representing available contracts (“unspent outputs”).
4. `nonces`: A set of pairs ([nonce commitment](#nonce-commitment),
   expiration timestamp). It records recent nonce entries in order to
   prevent duplicates. The expiration timestamp is used to prune
   outdated records.

### Block header

A **block header** is a [tuple](txvm.md#tuple) with the following
fields:

0. `version`, an int, block version (3 or higher).
1. `height`, an int, block serial number. Initial block has
   height 1. Height increases by 1 with each new block.
2. `previd`, a string, the previous block’s [ID](#block-id). In the
   case of the initial block (which has no previous block), an
   all-zero string of 32 bytes.
3. `timestamp`, an int, time of the block in milliseconds since
   00:00:00 UTC Jan 1, 1970. Each new block must have a time strictly
   later than the block before it.
4. `refscount`, an int, a number of block ids to be stored for
   reference. New blocks may decrease `refscount` but may not increase
   it by more than 1.
5. `runlimit`, an int, a total runlimit allocated to verifying
   transactions and authenticating the current block.
6. `txroot`, a string, a 32-byte
   [transactions merkle root](#transactions-merkle-root) of the
   transactions included in the block.
7. `csroot`, a string, a 32-byte
   [contracts merkle root](#contracts-merkle-root) that commits to the
   state of all contracts after applying transactions in the block.
8. `nsroot`, a string, a 32-byte
   [nonces merkle root](#nonces-merkle-root) that commits to the state
   of all nonces after applying transactions in the block.
9. `nextpredicate`, an
   [authentication predicate](#authentication-predicate) tuple that
   represents an authentication challenge for adding a new block after
   this one.

Future versions of this protocol may add new fields to the blockheader
tuple. Those fields are ignored by older software for validation
purposes, but must be included in the [block ID](#block-id).

### Block

A **block** is a tuple that carries actual transactions and signatures
in addition to the [block header](#block-header).

0. `header`, a [block header](#block-header) tuple.
1. `txs`, a tuple of
   [transaction witness structures](txvm.md#transaction-witness).
2. `args`, a tuple of authentication arguments (typically signatures)
   satisfying the `nextpredicate` in the previous block (identified by
   `header.previd`). Empty for the initial block.

### Block ID

A **block ID** is a hash of the [block header](#block-header)
structure using the [serialization](txvm.md#serialization) and
[VMHash](txvm.md#vmhash) procedures defined in [TxVM](txvm.md):

    blockid = VMHash("BlockID", serialize(blockheader))

### Authentication predicate

An **authentication predicate** is a [tuple](txvm.md#tuple) whose
first field is an [integer](txvm.md#int) `version` that determines the
content and semantics of the fields that follow.

This specification supports only `version = 1`, which indicates the
multisignature predicate:

    {1, <M>, <pubkey1>,...,<pubkeyN>}

Here, `M` is an integer representing a _quorum_, a threshold number of
signatures required for validity; and `pubkey1` through `pubkeyN` are
Ed25519 public keys (as strings) as defined by
[RFC8032](https://tools.ietf.org/html/rfc8032). `N` must be equal to
or greater than `M`.

When a block uses this authentication predicate, the next block’s
`args` tuple must contain exactly `N` strings, of which exactly `M`
must be valid Ed25519 signatures matching the corresponding
pubkey. The remaining `N-M` strings must be empty. See the
[Validate Block](#validate-block) procedure for details.

### Transactions merkle root

The **transactions merkle root** is
the [root hash](#merkle-root) of the [merkle binary hash tree](#merkle-binary-tree)
over the
[transaction witness commitments](#transaction-witness-commitment),
one per transaction.

### Transaction witness commitment

Each transaction has a **transaction witness commitment** calculated
as:

    txwc = txid || VMHash("WitnessHash", serialize(txwitness))

where:

* `txid` is the TxVM [Transaction ID](txvm.md#transaction-id).
* `serialize` is the TxVM [serialization](txvm.md#serialization)
  procedure.
* `txwitness` is the TxVM
  [Transaction Witness](txvm.md#transaction-witness) tuple `{version,
  runlimit, program}`.
* `VMHash` is the TxVM [VMHash](txvm.md#vmhash) function.

The resulting string is committed to a block header via its
[transactions merkle root](#transactions-merkle-root) which enables
efficient proofs of publication both for the results of the
transaction (embedded in the `txid`) and for the raw transaction
witness.

### Contracts merkle root

The **contracts merkle root** is the [root hash](#merkle-root) of the
[merkle patricia tree](#merkle-patricia-tree) formed by all existing
[contract snapshot IDs](txvm.md#snapshot-id) after
[applying](#apply-block) the block. This allows bootstrapping nodes
from recent blocks and an archived copy of the corresponding merkle
patricia tree rather than by processing all historical transactions.

### Nonces merkle root

The **nonces merkle root** is the [root hash](#merkle-root) of the
[merkle patricia tree](#merkle-patricia-tree) formed by all existing
[nonce commitments](#nonce-commitment) after [applying](#apply-block)
the block.

### Nonce commitment

A **nonce commitment** is a 40-byte string formed by concatenating a
hash of a serialized nonce with its expiration timestamp encoded as
little-endian 64-bit integer:

    nc = VMHash("Nonce", serialize(nonce)) || uint64le(nonce.exp)

Here `nonce` is a TxVM tuple from the
[transaction log](txvm.md#transaction-log):

    nonce = {"N", vm.caller, vm.currentcontract.seed, blockid, exp}

[Serialization](txvm.md#serialization) and [VMHash](txvm.md#vmhash)
are defined in the [TxVM](txvm.md) specification.

### Merkle root

The hash at the root of a *merkle tree* ([binary](#merkle-binary-tree)
or [patricia](#merkle-patricia-tree)). Merkle roots are used within
blocks to commit to a set of transactions and the complete state of
the blockchain. They are also used in merkleized programs (not
discussed here) and may also be used for structured reference-data
commitments.

### Merkle binary tree

The protocol uses a binary merkle hash tree for efficient proofs of
validity. The construction is from
[RFC 6962 Section 2.1](https://tools.ietf.org/html/rfc6962#section-2.1),
but using SHA3–256 instead of SHA2–256. It is reproduced here, edited
to update the hashing algorithm.

The input to the *merkle binary tree hash* (MBTH) is a list of data
entries; these entries will be hashed to form the leaves of the merkle
hash tree. The output is a single 32-byte hash value. Given an ordered
list of n inputs, `D[n] = {d(0), d(1), ..., d(n-1)}`, the MBTH is thus
defined as follows:

The hash of an empty list is the hash of an empty string:

    MBTH({}) = SHA3-256("")

The hash of a list with one entry (also known as a leaf hash) is:

    MBTH({d(0)}) = SHA3-256(0x00 || d(0))

For n > 1, let k be the largest power of two smaller than n (i.e., k <
n ≤ 2k). The merkle binary tree hash of an n-element list `D[n]` is
then defined recursively as

    MBTH(D[n]) = SHA3-256(0x01 || MBTH(D[0:k]) || MBTH(D[k:n]))

where `||` is concatenation and `D[k1:k2]` denotes the list `{d(k1),
d(k1+1),..., d(k2-1)}` of length `(k2 - k1)`. (Note that the hash
calculations for leaves and nodes differ. This domain separation is
required to give second preimage resistance.)

Note that we do not require the length of the input list to be a power
of two. The resulting merkle binary tree may thus not be balanced;
however, its shape is uniquely determined by the number of leaves.

![Merkle binary tree](merkle-binary-tree.png)

### Merkle patricia tree

The protocol uses a binary radix tree with variable-length branches to
implement a *merkle patricia tree*. This tree structure is used for
efficient concurrent updates of the
[contracts merkle root](#contracts-merkle-root) and compact recency
proofs for unspent outputs.

The input to the *merkle patricia tree hash* (MPTH) is a list of data
entries; these entries will be hashed to form the leaves of the merkle
hash tree. The output is a single 32-byte hash value. The input list
must be prefix-free; that is, no element can be a prefix of any
other. Given a sorted list of n unique inputs, `D[n] = {d(0), d(1),
..., d(n-1)}`, the MPTH is thus defined as follows:

The hash of an empty list is a 32-byte all-zero string:

    MPTH({}) = 0x0000000000000000000000000000000000000000000000000000000000000000

The hash of a list with one entry (also known as a leaf hash) is:

    MPTH({d(0)}) = SHA3-256(0x00 || d(0))

For n > 1, let the bit string p be the longest common prefix of all
items in `D[n]`, and let k be the number of items that have a prefix
`p||0` (that is, p concatenated with the single bit 0). The merkle
patricia tree hash of an n-element list `D[n]` is then defined
recursively as:

    MPTH(D[n]) = SHA3-256(0x01 || MPTH(D[0:k]) || MPTH(D[k:n]))

where `||` is concatenation and `D[k1:k2]` denotes the list `{d(k1),
d(k1+1),..., d(k2-1)}` of length `(k2 - k1)`. (Note that the hash
calculations for leaves and nodes differ. This domain separation is
required to give second-preimage resistance.)

Note that the resulting merkle patricia tree may not be balanced;
however, its shape is uniquely determined by the input data.

![Merkle patricia tree](merkle-patricia-tree.png)


## Validation procedures

### Validate block

**Inputs:**

1. `block`, current [block](#block) tuple.
2. `prevheader`, a [block header](#block-header) tuple from the
   previous block.

**Outputs:**

1. `false` if validation failed.
2. List of `(txlog,txid)` pairs per transaction if validation
   succeeded.

**Algorithm:**

1. Verify that `block.header.version` is greater than or equal to
   `prevheader.version`.
2. Verify that `block.header.height` is equal to `prevheader.height +
   1`.
3. Verify that `block.header.previd` is equal to the
   [block ID](#block-id) of `prevheader`.
4. Verify that `block.header.timestamp` is strictly greater than
   `prevheader.timestamp`.
5. Verify that `block.header.refscount` is less than or equal to
   `prevheader.refscount + 1`.
6. Authenticate the block:
    1. Verify that `prevheader.nextpredicate` is a tuple with first
       item set to 1 and the second item being a
       non-negative [integer](txvm.md#int).
    2. Let `N` be `len(prevheader.nextpredicate)-2`.
    3. Let `M` be the second item in the `prevheader.nextpredicate`
       (the “quorum”).
    4. Let `M’ = 0`.
    5. Verify that `block.args` contains exactly `N` strings.
    6. Let `pubkey_i` be the `i+2`-th item in
       `prevheader.nextpredicate` tuple (`i` starts with 1).
    7. Let `sig_i` be the `i`th string in `block.args`.
    8. For `i` from 1 to `N`:
        * Verify that `pubkey_i` has length 32.
        * If `sig_i` is not empty: verify it is 64-byte long and a
          valid Ed25519 signature for `pubkey_i` and `block`’s
          [ID](#block-id) as a message.
        * If `sig_i` is not empty, set `M’ = M’ + 1`.
    9. Verify `M’` is equal to `M`.
7. Let `R` be the remaining runlimit, initially set to
   `block.header.runlimit`.
8. For each transaction witness `txwit` in the `block.txs` list:
    1. If the `block.header.version` is 3, verify that `tx.version` is
       equal to 3.
    2. Reduce `R` by `txwit.runlimit`. Fail if `R` becomes negative.
    3. [Execute the transaction](txvm.md#vm-operation) per TxVM
       specification, producing transaction log `txlog` and
       transaction ID `txid`. If execution fails or the transaction is
       not finalized, return false.
9. Compute [transactions merkle root](#transactions-merkle-root)
   `txroot’` using `txid` and `txwit`.
10. Verify that `txroot’` is equal to `block.header.txroot`.
11. If the `block.header.version` is 3: verify that the `block.header`
    tuple does not have excess fields that are not defined in this
    specification.
12. Return a list of `(txlog,txid)` pairs for updating the state.

Note: Each transaction decreases the block’s runlimit by the
_declared_ runlimit in the transaction witness (as opposed to the
runlimit actually consumed during TxVM execution). This is necessary
for future upgrades to TxVM where additional runlimit is consumed by
new operations, making non-upgraded nodes not aware of it.

### Join new network

A new node starts here when joining a new network (with height = 1).

**Inputs:**

1. consensus predicate,
2. time.

**Output:** true.

**Affects:** current blockchain state.

**Algorithm:**

1. [Make an initial block](#make-initial-block) with the given time
   and [consensus predicate](#block-header).
2. Allocate an empty `contracts` set, empty `nonces` set, and an empty
   `refids` list.
3. The initial block and these empty sets together constitute the
   _initial state_.
4. Assign the initial state to the current blockchain state.
5. Return true.

### Join existing network

A new node starts here when joining a running network (with height >
1). In that case, it does not validate all historical blocks, and the
correctness of the blockchain state must be established out of band,
for example, by comparing the [block ID](#block-id) to a known-good
value.

**Input:** `state`, a blockchain state.

**Output:** true or false.

**Affects:** current blockchain state.

**Algorithm:**

1. Compute the [contracts merkle root](#contracts-merkle-root)
   `csroot’` of the `state.contracts`.
2. Verify that `state.header.csroot` is equal to `csroot’`; if not,
   halt and return false.
3. Assign the input state to the current blockchain state.
4. Return true.

### Make initial block

**Inputs:**

1. `predicate`, an
   [authentication predicate](#authentication-predicate) tuple,
2. `time`, a timestamp im milliseconds.

**Output**: block.

**Algorithm:**

1. Return a block with the following values:
    * `block.header.version = 3`.
    * `block.header.height = 1`.
    * `block.header.previd`: string with 32 zero bytes.
    * `block.header.timestamp = time`.
    * `block.header.refscount = 0`.
    * `block.header.runlimit = 0`.
    * `block.header.txroot`:
      [transactions merkle root](#transactions-merkle-root) of an
      empty list.
    * `block.header.csroot`:
      [contracts merkle root](#contracts-merkle-root) of an empty set.
    * `block.header.nsroot`: [nonces merkle root](#nonces-merkle-root)
      of an empty set.
    * `block.header.nextpredicate = predicate`.
    * `block.args = {}`, an empty tuple.
    * `block.txs = {}`, an empty tuple.

### Apply block

**Inputs:**

1. `block`, a [block](#block),
2. `state`, a [blockchain state](#blockchain-state).

**Output:** `state′`, a new blockchain state.

**Algorithm:**

1. [Validate the block](#validate-block) `block` with `prevheader` set
   to current `state.header`; if validation fails, halt and return
   `state` unchanged.
2. Let `state′` be `state`.
3. For each transaction log produced in the previous step:
    1. [Apply the transaction log](#apply-transaction-log) using the
       transaction log and the current `state′`, yielding a new state
       `state′′`.
    2. If transaction failed to be applied (did not change blockchain
       state), halt and return `state` unchanged.
    3. Replace `state′` with `state′′`.
4. Test that the [contracts merkle root](#contracts-merkle-root) of
   `state′.contracts` is equal to `block.header.csroot`; if not, halt
   and return `state` unchanged.
5. Remove elements of the `state′.nonces` set where the expiration
   timestamp is less than `block.header.timestamp`.
6. Test that [nonces merkle root](#nonces-merkle-root) of
   `state′.nonces` is equal to the `block.header.nsroot`; if not, halt
   and return `state` unchanged.
7. Set `state′.header` to `block.header`.
8. Add `state′.header` to the end of the `state′.refids` subset.
9. Prune the `state′.refids` set to the number indicated by
   `block.header.refscount`, removing the oldest IDs if `refscount` is
   less than the number of `state′.refids`. (The
   [block validation](#validate-block) step ensures that `refscount`
   is less than or equal to the number of `state′.refids` before
   pruning.)
10. Return the new state `state′`.

### Apply transaction log

**Inputs:**

1. `txlog`, finalized [transaction log](txvm.md#transaction-log),
2. `state`, a [blockchain state](#blockchain-state).

**Output:** new blockchain state.

**Algorithm:**

1. Apply transaction log to the `state` as follows. If any of the
   following steps reject the transaction, halt and return the input
   blockchain state unchanged.
    1. For each timerange tuple `{"R", ctx, min, max}` in the
       transaction log:
        1. If the `min` is greater than the block’s timestamp, reject
           the transaction.
        2. If the `max` is not zero, and is less than the block’s
           timestamp, reject the transaction.
    2. For each nonce tuple `{"N", ctx, contractseed, blockid, exp}`
       in the transaction log:
        1. Verify that `blockid` is one of the following, rejecting
           the transaction if it’s not:
            * an all-zero 32-byte string, or
            * the ID of the initial block header, or
            * one of the block ids in `state.refids`.
        2. Compute the [nonce commitment](#nonce-commitment) `nc` from
           the nonce tuple.
        3. If `nc` is already present in `state.nonces`, reject the
           transaction.
        4. Add `nc` to `state.nonces`.
    3. For each contract tuple `{"I", ctx, snapshotid}` or `{"O", ctx,
       snapshotid}` in the transaction log:
        1. If the tuple is an input, remove `snapshotid` from the
           `state.contracts` set.
        2. If the tuple is an output, add `snapshotid` to the
           `state.contracts` set.
    4. Return the updated blockchain state.
