# TxVM

This is the specification for TxVM, the transaction virtual machine.

TxVM defines a procedural representation for blockchain transactions
and the rules for a virtual machine to interpret them and ensure their
validity.

* [Overview](#overview)
    * [Motivation](#motivation)
    * [Concepts](#concepts)
    * [Types](#types)
    * [Plain data](#plain-data)
    * [Entries](#entries)
    * [Derived types](#derived-types)
    * [Conservation law](#conservation-law)
    * [Compatibility](#compatibility)
* [Definitions](#definitions)
    * [Argument stack](#argument-stack)
    * [Contract stack](#contract-stack)
    * [Transaction log](#transaction-log)
    * [VMHash](#vmhash)
    * [IDs](#ids)
* [VM operation](#vm-operation)
    * [VM state](#vm-state)
    * [VM execution](#vm-execution)
    * [Running programs](#running-programs)
    * [Versioning](#versioning)
    * [Runlimit](#runlimit)
    * [Encoding](#encoding)
* [Instructions](#instructions)
    * [Numeric and boolean instructions](#numeric-and-boolean-instructions)
    * [Crypto instructions](#crypto-instructions)
    * [Stack instructions](#stack-instructions)
    * [Control flow instructions](#control-flow-instructions)
    * [Value instructions](#value-instructions)
    * [Data instructions](#data-instructions)
* [Examples](#examples)
    * [Deferred programs](#deferred-programs)
    * [Pay To Public Key](#pay-to-public-key)
    * [Signature programs](#signature-programs)
    * [Single-asset transfer](#single-asset-transfer)
    * [Secure nonce](#secure-nonce)
    * [Nonce and issuance](#nonce-and-issuance)
    * [More examples](#more-examples)

## Overview

### Motivation

Earlier versions of the Chain Protocol (and other blockchain systems)
represent transactions with a static data structure, exposing the
pieces of information — the inputs and outputs, with their associated
fields — needed to test the transaction’s validity. An ad hoc set of
validation rules applied to that information produces a true/false
result.

In TxVM, this model is inverted. A transaction is a program that runs
in a specialized virtual machine that embodies validation rules. The
_result_ of the transaction is information about inputs and outputs,
guaranteed valid by the successful completion of the transaction
program.

The “scripts” that in other blockchain systems are used to lock and
unlock pieces of blockchain value are, in TxVM, simply subroutines of
the overall transaction program. This allows the creation of
sophisticated, secure value flows that are difficult or impossible to
do otherwise.

### Concepts

The transaction program, together with a version number and runlimit,
is called the [transaction witness](#transaction-witness). It contains
all data and logic required to produce a unique
[transaction ID](#transaction-id). It also contains any necessary
signatures (although these typically appear after “finalization” and
don’t contribute to the transaction ID).

A [witness program](#witness-program) runs in the context of a
stack-based virtual machine.  When the virtual machine executes the
program, it creates and manipulates data of various types:
[plain data](#plain-data), including integers, strings, and tuples;
and special [entry types](#entries) that include [values](#values) and
[contracts](#contracts). A value is a specific amount of a specific
asset that can be merged or split, issued or retired, but not
otherwise created or destroyed. A contract encapsulates a program (as
TxVM bytecode) plus its runtime state, and once created must be
executed to completion or persisted in the global state for later
execution.

Some TxVM instructions (such as storing a contract for later
execution) propose alterations to the global blockchain state. These
proposals accumulate in the [transaction log](#transaction-log), a
data structure in the virtual machine that is the principal result of
executing a transaction. Hashing the transaction log gives the unique
[transaction ID](#transaction-id).

A TxVM transaction is valid if and only if it runs to completion
without encountering failure conditions and without leaving any data
on the virtual machine’s stacks.

After a TxVM program runs, the proposed state changes in the
transaction log are compared with the global state to determine the
transaction’s applicability to the blockchain.

### Types

The items on TxVM stacks are typed. The available types fall into two
broad categories: [plain data](#plain-data) and [entries](#entries).

Plain data items can be freely created, copied, and destroyed. Entries
are subject to special rules as to when and how they may be created
and destroyed, and may never be copied.

### Plain data

TxVM supports these plain data types:

* [Int](#int)
* [String](#string)
* [Tuple](#tuple)

Items of these types can be freely created, copied, and destroyed.

#### Int

A signed 64-bit integer, i.e. an integer between -2^63 and 2^63 - 1.

#### String

A byte string with length between 0 and 2^31 - 1 bytes.

#### Tuple

An immutable sequence of zero or more plain data items.


### Entries

TxVM supports these entry types:

* [Values](#values)
* [Contracts](#contracts)
* [Wrapped contracts](#wrapped-contracts)

Items of these types may not be created, destroyed, or copied except
as described below.

#### Values

A value is a specific amount of a specific asset type. Values are
created with [issue](#issue) and destroyed with [retire](#retire). As
a special case, zero-amount values may be created with [nonce](#nonce)
and destroyed with [drop](#drop). A value may be [split](#split) into
two values adding to the same amount, and two values of like type may
be [merged](#merge) together.

Each value also contains an _anchor_, a string derived from the nonce,
issuance, or upstream value(s) that created it. Anchors ensure the
global uniqueness of distinct values even when they have identical
amounts and asset types. Anchors also prevent “replay attacks.”

Values are secured by “locking them up” inside
[contracts](#contracts). Each contract has its own stack where values
(and other data) may be stored, and also its own program expressing
the conditions for releasing or otherwise manipulating any values it
controls.

A value can be [converted](#conversion) to plain data by rendering it
as a [tuple](#tuple) containing these items:

0. `"V"`, a string, the type code for converted values.
1. `amount`, an integer.
2. `assetid`, a string.
3. `anchor`, a string.

#### Contracts

A contract is a [program](#program) (a string of bytecode) and
associated storage in the form of a stack. Contracts are created with
the [contract](#contract) instruction and destroyed by running them to
completion, at which point the contract’s stack must be empty.

Each contract also contains a [seed](#contract-seed) which is a hash
of the initial program with which the contract was created. The
program can change during the contract’s lifecycle — the
[output](#output), [yield](#yield), and [wrap](#wrap) instructions all
update it — but the seed will not.

A running contract that has not yet completed may remove itself from
the current transaction (and return control to its caller) with the
[output](#output) instruction. Such a contract is stored in the global
blockchain state. It may be recovered from the global state with the
[input](#input) instruction, whereupon it can resume execution with
the stack contents it had before. Both actions are recorded in the
[transaction log](#transaction-log).

A contract may also suspend its execution with the [wrap](#wrap) and
[yield](#yield) instructions.

A contract can be [converted](#conversion) to plain data by rendering
it as a [tuple](#tuple) containing these items:

0. `"C"`, a string, the type code for converted contracts.
1. `seed`, a string.
2. `program`, a string.

followed by the contents of its stack as zero or more items, each item
[converted](#conversion). The bottom item of the stack appears first
and the top item appears last.

The plain data tuple representation of a contract is used as an
argument to the [input](#input) instruction, which “un-converts” it,
reconstituting it as a callable contract object. (The
[output](#output) instruction stores only the contract’s
[snapshot ID](#snapshot-id) in the global state. It is the caller’s
responsibility when using [input](#input) to construct a tuple
producing the same snapshot ID as was previously stored.)

#### Wrapped contracts

A wrapped contract is a contract that has been made
[portable](#portable-types) with the [wrap](#wrap) instruction. When
[called](#call), a wrapped contract automatically “unwraps” and
becomes a plain contract.

A wrapped contract can be [converted](#conversion) to plain data by
rendering it as a [tuple](#tuple) containing these items:

0. `"W"`, a string, the type code for converted wrapped contracts.
1. `seed`, a string.
2. `program`, a string.

These are followed by the contents of the stack as zero or more items,
each item [converted](#conversion). The bottom item of the stack
appears first and the top item appears last.


### Derived types

The plain data and entry types described above are all the types
defined by TxVM. However, in some contexts, values of the built-in
types are used as if they were other, domain-specific types. This
section describes how certain types are sometimes interpreted.

#### Boolean

A boolean is a true-or-false value. Any plain data value may be
interpreted as a boolean. All values are “true” except for the integer
`0`, which is “false.” (Note, empty tuples and empty strings are also
“true.”) Operations that produce booleans produce `0` for false and
`1` for true.

Important: Entry types cannot be interpreted as booleans. Operations
that expect a boolean value fail execution if it is not a plain data
item.

#### Program

A program is a string containing a sequence of TxVM instructions. Each
instruction is an **opcode** optionally followed by **immediate
data**.

The **opcode** is a variable-length unsigned integer encoded using
[LEB128](https://en.wikipedia.org/wiki/LEB128).

The **immediate data** is 0 or more bytes, depending on the opcode.
All opcodes below 0x5f (95 in decimal) have no immediate data.
Opcodes starting with 0x5f are [pushdata](#pushdata)
instructions. Each is followed by `opcode - 95` bytes of immediate
data. (So 0x5f is followed by zero bytes of immediate data and pushes
a zero-byte string to the stack, 0x60 is followed by one byte of
immediate data and pushes that as a one-byte string to the stack, 0x61
is followed by two bytes, and so on.)

#### Witness program

A witness program is a [program](#program) (a string of bytecode)
specified by a [transaction witness](#transaction-witness), which runs
as a “top-level contract” during TxVM execution.

#### Contract snapshot

A contract snapshot is a [tuple](#tuple) representing the complete
state of a [contract](#contracts) at the point that its execution is
suspended with the [output](#output) instruction.

See also [snapshot ID](#snapshot-id).

#### Transaction witness

A transaction witness is a [tuple](#tuple) with the following fields:

0. `version`, an int.
1. `runlimit`, an int.
2. `program`, a [program](#program) (i.e. a string of bytecode),
   referred to as the [witness program](#witness-program).

#### Portable types

When a contract executes the [wrap](#wrap) instruction to become a
[wrapped contract](#wrapped-contracts), or when it is
[snapshotted](#contract-snapshot) by the [output](#output)
instruction, every item on its stack must be _portable_.  It is an
error at such times for the stack to contain any non-portable items.

All [plain data items](#plain-data) (integers, strings, and tuples)
are portable, as are [values](#values) and
[wrapped contracts](#wrapped-contracts). Ordinary
[contracts](#contracts) are _not_ portable.


### Conservation law

Rules regarding the handling of values and contracts ensure that all
transactions balance and that values cannot be created except as
authorized, or destroyed without leaving a record.

These rules can be thought of as the conservation laws of TxVM.

The most important law is that a transaction cannot complete
successfully unless the top-level contract stack and the argument
stack are both empty. Values and contracts are limited in the ways
they can be removed from stacks once added:

* A contract is removed if it runs to completion (with its own stack
  empty);
* A value is removed if destroyed with the [retire](#retire)
  instruction, which creates a record in the
  [transaction log](#transaction-log);
* A contract is removed, together with the contents of its stack, if
  it uses the [output](#output) instruction, which creates a record in
  the [transaction log](#transaction-log) and an entry in the global
  blockchain state allowing the contract (and its stack) to be
  recreated later.

Because of this law, every contract must execute to completion or
invoke [output](#output); and every value must be safely locked in a
contract that invokes [output](#output), or be retired.


#### The value lifecycle

New values may be created with the [issue](#issue) instruction. The
asset type of the value is given by its [asset ID](#asset-id), an
identifier derived from the contract containing the `issue`
instruction that created it. An `issue` instruction in any other
contract necessarily creates values of a different asset type. In this
way, an asset’s issuance contract is unique and solely responsible for
deciding when issuance is authorized.

Values with a zero amount may be copied and dropped from the stack
freely, like plain data. Zero-amount values are useful for the anchors
they contain.

Non-zero values never exist in two places at once.  They may not be
copied or dropped, but they may be moved between stacks with
[get](#get) and [put](#put).

When [retire](#retire) is used to destroy a value, a retirement entry
is added to the [transaction log](#transation-log).

The [split](#split) instruction turns a single value into two values
with the same sum. The [merge](#merge) instruction turns two values of
like type into a single value whose amount is the sum of the inputs.

#### The contract lifecycle

A contract never exists in two places at once. It cannot be destroyed
or stored anywhere without its own permission.

When a contract is created with the [contract](#contract) instruction,
it exists on a parent contract’s stack.

When a contract is called with [call](#call), it is removed from the
caller’s stack and exists transiently in the VM's internal call stack.

When a contract completes execution with an empty contract stack, it
is destroyed by the VM.

When a contract suspends execution via [yield](#yield), it is placed
back on the argument stack.

When a contract suspends execution via [output](#output), it is moved
(in the form of its [snapshot ID](#snapshot-id)) to the global
blockchain state.

When a contract is recreated using [input](#input), it is pushed to
the contract stack and its snapshot ID is removed from the global
blockchain state.

When a contract is [wrapped](#wrap), it is returned to the argument
stack in a [portable](#portable-types) form. A portable contract can
be stored inside other contracts or [called](#call).

(If wrapped contracts can be called like normal contracts, why have
the wrapped/unwrapped distinction at all? It’s to ensure that a
contract may not be persisted to the global state (on the stack of a
parent contract) without its own permission. The only way to wrap a
contract, after all, is for the contract itself to execute the `wrap`
instruction.)


### Compatibility

TxVM is incompatible with versions 1 and 2 of the Chain protocol. The
version number for TxVM transactions and blocks therefore starts at 3.

Forward- and backward-compatible upgrades (“soft forks”) are possible
with [extension instructions](#ext), enabled by the
[extension flag](#versioning) and higher version numbers. It is
possible to write a compatible contract that uses features of a newer
transaction version while remaining usable by non-upgraded software
(that understands only older transaction versions) as long as
new-version code paths are protected by checks for the transaction
version. To facilitate that, a hypothetical TxVM upgrade may introduce
an extension instruction “version assertion” that fails execution if
the version is below a given number (e.g. `4 versionverify`).


## Definitions

### Argument stack

The argument stack is a shared stack used to pass items between
[contracts](#contracts).

The argument stack may contain [plain data items](#plain-data) and
[entries](#entries). Items are moved from the current contract stack
to the argument stack using the [put](#put) instruction, and retrieved
using the [get](#get) instruction.

### Contract stack

Each [contract](#contracts) has its own *contract stack*.

A contract stack may contains [plain data items](#plain-data) and
[entries](#entries). When a contract is running (via the [call](#call)
instruction), its stack becomes the current contract stack. All
stack-affecting instructions operate on the current contract
stack. The [get](#get) and [put](#put) instructions move items between
the current contract stack and the [argument stack](#argument-stack).

If a contract executes the [wrap](#wrap) or [output](#output)
instruction, its stack must contain no
[non-portable types](#portable-types).

### Transaction log

The *transaction log* contains [tuples](#tuple) that describe the
effects of various instructions.

The transaction log is empty at the beginning of a TxVM program. It is
append-only. Items are added to it upon execution of any of the
following instructions:

* [finalize](#finalize)
* [input](#input)
* [issue](#issue)
* [log](#log)
* [nonce](#nonce)
* [output](#output)
* [retire](#retire)
* [timerange](#timerange)

The details of the item added to the log differs for each
instruction. See the instruction’s description for more information.

The `finalize` instruction prohibits further changes to the
transaction log. Every TxVM program must execute `finalize` exactly
once.

### VMHash

TxVM defines a family of hash functions, collectively denoted
`VMHash(F,X)`, based on SHA-3 Derived Functions as specified by
[NIST SP 800-185](http://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-185.pdf).

Each hash function has a variable _function name_ string `F` that is
appended to a constant customization string `S` (in NIST terms) to
allow efficient precomputation of any specific hash function
instance. (This is due to customization strings being padded to 168
bytes that are fully permuted using the Keccak function.)

The value of `S` for all hash functions in this specification is
`ChainVM.`

For a given “function name” `F`, `VMHash(F,X)` is a secure hash
function that takes an input string `X` and outputs a 256-bit hash.

    VMHash(F,X) = cSHAKE128(X, L=256, N="", S="ChainVM." || F)

This document gives specific values of `F` for different uses of
`VMHash`. See details below.

### IDs

#### Transaction ID

The unique ID of a transaction is computed from the
[transaction log](#transaction-log) after execution of the
[finalize](#finalize) instruction, which prohibits further changes to
the log. A TxVM program that does not execute `finalize` does not have
a transaction ID.

To compute the transaction ID, each item in the log is
[serialized](#serialization) and a
[merkle binary tree](blockchain.md#merkle-binary-tree) is constructed
using these serialized items. The ID is the root hash of the tree.

    txid = MBTH({serialize(firstitem), ..., serialize(lastitem)})

#### Contract seed

The *seed* of a contract is a hash of the `program` argument used in
the [contract](#contract) instruction:

    contractseed = VMHash("ContractSeed", program)

It remains the same even as a running contract’s changes during its
lifecycle (via [yield](#yield), [wrap](#wrap), or [output](#output)).

Note 1: The contract seed that a given program will produce can be
computed with the sequence `<program> "ContractSeed" vmhash`.

Note 2: The contract seed of the currently running contract is
accessible via the [self](#self) instruction.

Note 3: The contract seed of another contract is accessible via the
[seed](#seed) instruction.

#### Asset ID

When the [issue](#issue) instruction is executed, the resulting value
has an *asset ID* that is a hash of the [seed](#contract-seed) of the
issuing contract combined with a customization `tag`:

    assetid = VMHash("AssetID", contractseed || tag)

Note: The asset ID that a given contract and tag will produce can be
computed with the sequence `<contractseed> <tag> cat "AssetID"
vmhash`.

#### Snapshot ID

The *snapshot ID* of a contract at a given point in its execution is a
hash of the [serialized](#serialization)
[snapshot](#contract-snapshot):

    snapshotid = VMHash("SnapshotID", serialize(snapshot))

The snapshot ID is used by the [output](#output) and [input](#input)
instructions.



## VM operation

Execution of a TxVM transaction happens in a virtual
machine. Successful completion proves the validity of the transaction
and produces a log of global state changes to be applied to the
blockchain.

Note: It is important to distinguish *validation* of a transaction,
which happens in isolation, from *application* of a transaction to the
blockchain state. A transaction that is valid in isolation may still
be invalid in the context of the blockchain — if, for instance, it
tries to spend some value that has already been spent elsewhere.

(Here, “spending value” specifically means “reconstituting a contract
with the [input](#input) instruction whose [snapshot ID](#snapshot-id)
does not appear in the global blockchain state.”)

### VM state

The TxVM virtual machine is a state machine with these attributes:

1. Extension flag `extension` (boolean)
2. Finalized flag `finalized` (boolean)
3. Unwinding flag `unwinding` (boolean)
4. Runlimit `runlimit` (int)
5. Caller Seed `caller` (string)
6. Argument stack `argstack` (a stack of [data items](#types))
7. Current contract `currentcontract` (a tuple of a stack of data
   items, a [seed](#contract-seed), and a [current program](#program))
8. Transaction log `log` (a sequence of [tuples](#tuple))

### VM execution

1. The VM is initialized with a
[transaction witness](#transaction-witness) `txwitness` as follows:
    * `extension` flag set to true or false according to the [transaction versioning](#versioning) rules for `txwitness.version`,
    * `version` integer set to `txwitness.version`,
    * `finalized` flag set to `false`,
    * `unwinding` flag set to `false`,
    * `runlimit` set to the `txwitness.runlimit`,
    * `caller` set to an all-zero 32-byte string,
    * `argstack` empty,
    * `currentcontract` with:
        * `stack`: empty,
        * `seed`: an all-zero 32-byte string,
        * `program = txwitness.program`.
    * `log` empty
2. The VM [runs](#running-programs) `txwitness.program`.
3. Execution fails if any of the following is true:
    1. Remaining runlimit is negative,
    2. The argument stack or current contract stack is non-empty.
4. Results are the VM’s `finalized` flag and its `log`. If `finalized`
   is true, a [transaction id](#transaction-id) may be computed from
   the `log`.

Note 1: Remaining runlimit in step 3 is allowed to be greater than 0,
in part to accommodate future extensions.

Note 2: The [witness program](#witness-program) (i.e., the top-level
contract) _may_ end with [output](#output), saving itself to the
global blockchain state and leaving an empty stack. It _may not_ end
with [wrap](#wrap) or [yield](#yield), since that would leave the
contract on the argument stack, causing execution failure.

Note 3: After execution, the resulting transaction log is further
validated against the blockchain state, as discussed above. That step,
called _application_, is described in
[the blockchain specification](blockchain.md#apply-transaction-log).

### Running programs

A [program](#program) is a sequence of instructions represented as
bytecode. A **run** executes the instructions one after another. A
**program counter** records the byte position of the next
instruction. It begins at zero and advances as each instruction is
decoded and executed. The [jumpif](#jumpif) instruction can set the
program counter to a value other than the position of the next
instruction, causing execution to branch to the new location.

A new instruction is executed only if `vm.unwinding` is false. A run
terminates normally after the program’s last instruction is
executed. It terminates abnormally if `vm.unwinding` becomes true.

Each instruction consumes some amount of the VM
[runlimit](#runlimit). If `vm.runlimit` is exhausted (drops below
zero), execution fails.

Runs may nest, as when one program invokes another via [call](#call)
or [exec](#exec). When an inner run terminates normally, the outer run
resumes where it left off. When an inner run terminates abnormally,
the outer run also terminates, unless the outer run was begun with
`call`. See [call](#call) for details. There is no resuming from an
execution failure.

Note 1: The purpose of the `vm.unwinding` flag is to terminate programs
early that were started with [exec](#exec), backing out to the nearest
enclosing [call](#call) and resuming from there.

Note 2: It is useful to distinguish a _program_, which is a simple
sequence of instructions, from a _contract_, which _contains_ a
program plus some other information. A bare program is executed with
[exec](#exec). The program in a contract is executed with
[call](#call).

### Versioning

1. Each transaction has a version number. Each
   [block](blockchain.md#block-header) also has a version number.
2. All TxVM [transaction witness](#transaction-witness) tuples must
   have transaction version 3 or greater. This is to avoid confusion
   with transactions from earlier iterations of Chain’s blockchain
   protocol.
3. Blocks that include TxVM transactions must have version 3 or
   greater. This is also to avoid confusion with blocks from earlier
   versions of Chain software.
4. Block version numbers must be monotonically non-decreasing: each
   block must have a version number equal to or greater than the
   version of the block before it.
5. The **current block version** is 3. The **current transaction
   version** is 3.

Extensions:

1. If the block version is equal to the **current block version**, no
   transaction in the block may have a version higher than the
   **current transaction version**.
2. If a transaction’s version is higher than the **current transaction
   version**, the TxVM `extension` flag is set to `true`. Otherwise,
   the `extension` flag is set to `false`.

### Runlimit

The runlimit specified by a
[transaction witness](#transaction-witness) is an amount of abstract
“cost” units that the VM is allowed to use during execution. Every
instruction executed decreases the remaining available runlimit. A
transaction that exhausts its runlimit fails execution.

Runlimits enable fine-grained accounting for the operational costs of
the network.

1. Blocks commit to a total runlimit for the transactions they
   contain. This total is greater than or equal to the sum of the
   runlimits specified in the block’s transactions. The block runlimit
   total may exceed the sum of the transaction runlimits if needed for
   flexibility and future extensions.
2. The VM is initialized with the runlimit specified in
   a [transaction witness](#transaction-witness).
3. Each instruction reduces the VM’s runlimit according to the total
   cost, consisting of:
    1. The [base cost](#base-cost), charged for each instruction when
       it is executed.
    2. The cost of creating a new data item, based on the item’s type
       ([string](#string-cost), [tuple](#tuple-cost), or
       [entry](#entry-cost)). Ints are free to create.
    3. A [copy cost](#copy-cost) for each copied item according to its
       type.
4. Copy- or item-creation costs are described along with each
   instruction in the sections below, where applicable. The base cost
   applies to all instructions and is not explicitly described.
5. If the runlimit goes below zero before the program counter reaches
   the length of the program, execution fails.
6. Execution of the transaction can leave some runlimit unconsumed.
   This is allowed for future extensions.
7. The runlimit specified in a
   [transaction witness](#transaction-witness) must be equal to or
   greater than the length of the transaction witness’s program (in
   bytes). This is to allow early detection and abort when receiving
   intractably long program strings.

#### Base cost

Each instruction immediately costs `1` when executed.

#### String cost

Each created string costs `1 + n` where `n` is the length of the
string in bytes.

#### Tuple cost

Each created tuple costs `1 + n` where `n` is the number of items in
the tuple.

#### Entry cost

Each created [entry](#entries) (values, contracts, and wrapped
contracts) costs `128` which reflects the cost of underlying hash
operations.

#### Copy cost

* [Ints](#int) have zero copy cost.
* [Strings](#string) have copy cost equal to
  [string cost](#string-cost).
* [Tuples](#tuple) have copy cost equal to [tuple cost](#tuple-cost).

Copy cost does not apply to [entries](#entries).


### Encoding

#### Serialization

Every [plain data item](#plain-data) can be serialized as a TxVM
program fragment that produces the item when executed.

* A string is encoded as a [pushdata](#pushdata) instruction with the
  string as the instruction’s “immediate data.”
* Integers in the range 0–19 inclusive are encoded as the appropriate
  [small integer](#smallint) opcode.
* All other integers are first encoded with
  [LEB128](https://en.wikipedia.org/wiki/LEB128). The result is
  serialized with a [pushdata](#pushdata) instruction and the encoded
  integer as immediate data, followed by an [int](#int) instruction.
* A tuple is encoded recursively as the encoding of each item in the
  tuple, followed by the encoding of integer `n` where `n` is the
  length of the tuple, followed by the [tuple](#tuple) instruction.

A serialized data item can be passed to [exec](#exec) to deserialize
the item and push it to the current contract stack.

#### Conversion

The [output](#output) and [input](#input) instructions must represent
contracts as [plain data](#plain-data), even when their stacks may
contain non-plain [entries](#entries). To do this, they employ a
**conversion procedure** as follows:

1. An item of any [plain data type](#plain-data) is converted to a
   2-element tuple: a _type code_ (a string) and a copy of that
   item. See the table below.
2. A [contract](#contracts) with `n` items on its stack is converted to
   an `n+3`-item tuple consisting of the type code `"C"`, the
   contract’s [seed](#contract-seed), the contract’s current
   [program](#program), and converted copies of the `n` stack items,
   from bottom-most to top-most.
3. A [wrapped contract](#wrapped-contracts) is converted as a contract,
   but with type code `"W"`.
4. A [value](#values) is converted to a four-item tuple: the type code
   `"V"`, its integer amount, its asset ID (as a string), and its
   anchor (as a string).

Type                                       | Type code | Example input       | Example output
-------------------------------------------|-----------|---------------------|-------------------------------------
[Int](#int)                                | `"Z"`     | `123              ` | `{"Z", 123}`
[String](#int)                             | `"S"`     | `"abc"            ` | `{"S", "abc"}`
[Tuple](#tuple)                            | `"T"`     | `{123,"abc"}      ` | `{"T", {123,"abc"}}`
[Value](#values)                           | `"V"`     | `<Value>          ` | `{"V", <amount>, <assetid>, <anchor>}`
[Contract](#contracts)                     | `"C"`     | `<Contract>       ` | `{"C", <seed>, <program>, convert(bottomitem), ..., convert(topitem) }`
[Wrapped Contract](#wrapped-contracts)     | `"W"`     | `<WrappedContract>` | `{"W", <seed>, <program>, convert(bottomitem), ..., convert(topitem) }`

Note: Transaction log items resemble converted data items, in that
they are tuples with a leading type code. Their codes are chosen to
avoid accidental ambiguity with the conversion type codes above.

Logging operation                          | Type code | Example log entry
-------------------------------------------|-----------|-----------------------------------------------------------
[input](#input)                            | `"I"`     | `{"I", vm.currentcontract.seed, inputid}`
[output](#output)                          | `"O"`     | `{"O", vm.caller, outputid}`
[log](#log)                                | `"L"`     | `{"L", vm.currentcontract.seed, item}`
[timerange](#timerange)                    | `"R"`     | `{"R", vm.currentcontract.seed, min, max}`
[nonce](#nonce)                            | `"N"`     | `{"N", vm.caller, vm.currentcontract.seed, blockid, exp}`
[issue](#issue)                            | `"A"`     | `{"A", contextid, amount, assetid, anchor}`
[retire](#retire)                          | `"X"`     | `{"X", vm.currentcontract.seed, amount, assetid, anchor}`
[finalize](#finalize)                      | `"F"`     | `{"F", vm.currentcontract.seed, vm.version, anchor}`



## Instructions

This table shows the numeric opcode for all instructions up to 0x5f,
which is “pushdata 0” (for pushing a zero-byte string to the current
contract stack). Higher-numbered opcodes are “pushdata N”
instructions, where N is the opcode minus 0x5f.

[Smallints](#smallint)      | [Smallints](#smallint) | [Int](#numeric-and-boolean-instructions)/[Stack](#stack-instructions) | [Values](#value-instructions)/[Crypto](#crypto-instructions)/[Tx](#transaction-instructions) | [Control flow](#control-flow-instructions)  | [Data](#data-instructions)
----------------------------|------------------------|-----------------------------------------------------------------------|----------------------------------------------------------------------------------------------|---------------------------------------------|-------------------------
`00` [0 / false](#smallint) | `10` [16](#smallint)   | `20` [int](#int)                                                      | `30` [nonce](#nonce)                                                                         | `40` [verify](#verify)                      | `50` [eq](#eq)
`01` [1 / true](#smallint)  | `11` [17](#smallint)   | `21` [add](#add)                                                      | `31` [merge](#merge)                                                                         | `41` [jumpif](#jumpif)                      | `51` [dup](#dup)
`02` [2](#smallint)         | `12` [18](#smallint)   | `22` [neg](#neg)                                                      | `32` [split](#split)                                                                         | `42` [exec](#exec)                          | `52` [drop](#drop)
`03` [3](#smallint)         | `13` [19](#smallint)   | `23` [mul](#mul)                                                      | `33` [issue](#issue)                                                                         | `43` [call](#call)                          | `53` [peek](#peek)
`04` [4](#smallint)         | `14` [20](#smallint)   | `24` [div](#div)                                                      | `34` [retire](#retire)                                                                       | `44` [yield](#yield)                        | `54` [tuple](#tuple)
`05` [5](#smallint)         | `15` [21](#smallint)   | `25` [mod](#mod)                                                      | `35` [amount](#amount)                                                                       | `45` [wrap](#wrap)                          | `55` [untuple](#untuple)
`06` [6](#smallint)         | `16` [22](#smallint)   | `26` [gt](#gt)                                                        | `36` [assetid](#assetid)                                                                     | `46` [input](#input)                        | `56` [len](#len)
`07` [7](#smallint)         | `17` [23](#smallint)   | `27` [not](#not)                                                      | `37` [anchor](#anchor)                                                                       | `47` [output](#output)                      | `57` [field](#field)
`08` [8](#smallint)         | `18` [24](#smallint)   | `28` [and](#and)                                                      | `38` [vmhash](#vmhash)                                                                       | `48` [contract](#contract)                  | `58` [encode](#encode)
`09` [9](#smallint)         | `19` [25](#smallint)   | `29` [or](#or)                                                        | `39` [sha256](#sha256)                                                                       | `49` [seed](#seed)                          | `59` [cat](#cat)
`0a` [10](#smallint)        | `1a` [26](#smallint)   | `2a` [roll](#roll)                                                    | `3a` [sha3](#sha3)                                                                           | `4a` [self](#self)                          | `5a` [slice](#slice)
`0b` [11](#smallint)        | `1b` [27](#smallint)   | `2b` [bury](#bury)                                                    | `3b` [checksig](#checksig)                                                                   | `4b` [caller](#caller)                      | `5b` [bitnot](#bitnot)
`0c` [12](#smallint)        | `1c` [28](#smallint)   | `2c` [reverse](#reverse)                                              | `3c` [log](#log)                                                                             | `4c` [contractprogram](#contractprogram)    | `5c` [bitand](#bitand)
`0d` [13](#smallint)        | `1d` [29](#smallint)   | `2d` [get](#get)                                                      | `3d` [peeklog](#peeklog)                                                                     | `4d` [timerange](#timerange)                | `5d` [bitor](#bitor)
`0e` [14](#smallint)        | `1e` [30](#smallint)   | `2e` [put](#put)                                                      | `3e` [txid](#txid)                                                                           | `4e` [prv](#prv)                            | `5e` [bitxor](#bitxor)
`0f` [15](#smallint)        | `1f` [31](#smallint)   | `2f` [depth](#depth)                                                  | `3f` [finalize](#finalize)                                                                   | `4f` [ext](#ext)                            | `5f+` [pushdata](#pushdata)

In the individual instruction descriptions that follow, certain
failure conditions are implicit. In particular, if the description
says (for example) “pops two ints from the stack,” the instruction is
understood to fail execution if fewer than two items are on the stack,
or if either is not an int.

Unless otherwise stated, references to “the stack” mean “the current
contract stack.”

When an instruction incurs data creation or copy costs, this is
indicated with language like “[Creates string](#string-cost) `h`” or
“[Copies](#copy-cost) `item`” and a link to the section on runlimit
costs.

### Numeric and boolean instructions

#### smallint

**0|...|19** → _(0|...|19)_

Pushes int `n` equal to the instruction’s code to the contract stack.

Note: Instructions `0x00` and `0x01` can be used to push
[boolean](#boolean) values `false` and `true`.

#### int

_x_ **int** → _n_

1. Pops a string `x` from the contract stack.
2. Decodes the prefix as a
   [LEB128](https://en.wikipedia.org/wiki/LEB128)-encoded unsigned
   64-bit integer `u`, ignoring any trailing bytes.
3. Interprets the 64 bits of `u` as a signed two's complement 64-bit
   integer `n`.
4. Pushes `n` to the contract stack.

Fails execution when `x` is not a valid LEB128 encoding of an integer.

Note: due to lack of clarity in the original specification and an accidental
implementation, [transaction version 3](#versioning) allows integers to
contain arbitrary trailing data after the valid LEB128 encoding.
However, encoding integers with such trailing data is discouraged
and may be forbidden in the future versions of TxVM.

#### add

_a b_ **add** → _a+b_

Pops two ints `a` and `b` from the stack, adds them, and pushes their
sum `a + b` to the stack.

Fails execution on overflow.

#### neg

_a_ **neg** → _-a_

Pops an int `a` from the stack, pushes the negated `a` to the stack.

Fails execution when `-a` overflows (i.e., `a` is `-2^63`).

#### mul

_a b_ **mul** → _a·b_

Pops two ints `a` and `b` from the stack, multiplies them, and pushes
their product `a · b` to the stack.

Fails execution on overflow.

#### div

_a b_ **div** → _a÷b_

Pops two ints `a` and `b` from the stack, divides them truncated
toward 0, and pushes their quotient `a ÷ b` to the stack.

Fails execution when:
* `a÷b` overflows;
* `b = 0`.

#### mod

_a b_ **mod** → _a mod b_

Pops two ints `a` and `b` from the stack, computes their remainder `a
% b`, and pushes it to the stack.

The integer quotient `q = a b div` and remainder `r = a b mod` satisfy
the following relationships: `a = q*b + r` and `|r| < |b|`

Fails execution when:
* `b = 0`;
* `a = -2^63` and `b = -1`.

#### gt

_a b_ **gt** → _bool_

1. Pops two ints `a` and `b` from the stack.
2. If `a` is greater than `b`, pushes int `1` to the stack.
3. Otherwise, pushes int `0`.

#### not

_p_ **not** → _bool_

1. Pops a [boolean](#boolean) `p` from the stack.
2. If `p` is `0`, pushes int `1`.
3. Otherwise, pushes int `0`.

#### and

_p q_ **not** → _bool_

1. Pops two [booleans](#boolean) `p` and `q` from the stack.
2. If both `p` and `q` are true, pushes int `1`.
3. Otherwise, pushes int `0`.

#### or

_p q_ **not** → _bool_

1. Pops two [booleans](#boolean) `p` and `q` from the stack.
2. If both `p` and `q` are [false](#boolean), pushes int `0`.
3. Otherwise, pushes int `1`.


### Crypto instructions

#### vmhash

_x f_ **vmhash** → _h_

1. Pops strings `f` and `x` from the contract stack.
2. [Creates string](#string-cost) `h` by computing a [VMHash](#vmhash): `h = VMHash(f,x)`.
3. Pushes the resulting string `h` to the contract stack.

#### sha256

_x_ **sha256** → _h_

1. Pops a string `x` from the contract stack.
2. [Creates string](#string-cost) `h` by computing SHA2-256: `h = SHA2-256(f,x)`.
3. Pushes the resulting string `h` to the contract stack.

#### sha3

_x_ **sha3** → _h_

1. Pops a string `x` from the contract stack.
2. [Creates string](#string-cost) `h` by computing SHA3-256: `h = SHA3(f,x)`.
3. Pushes the resulting string `h` to the contract stack.

#### checksig

_msg pubkey sig scheme_ **checksig** → _bool_

1. Pops [plain data item](#plain-data) `scheme` and strings `sig`, `pubkey` and `msg` from the contract stack.
2. If the string `sig` is empty, pushes int `0` to the contract stack.
3. If the string `sig` is not empty:
    1. Reduces `vm.runlimit` by 2048.
    2. If `scheme` is an int `0`:
        1. Fails execution if `pubkey` is not 32 bytes long.
        2. Fails execution if `sig` is not 64 bytes long.
        3. Performs an [Ed25519](https://tools.ietf.org/html/rfc8032) signature check with `pubkey` as the public key, `msg` as the message, and `sig` as the signature.
        4. If signature check fails, fail the VM execution.
    3. If `scheme` is any other value and `vm.extension` is `false`, fails execution.
    4. Pushes int `1` to the contract stack.

Note 1: Message is the first argument to simplify construction of
multi-signature predicates.

Note 2: As an optimization, the implementation of `checksig` may
immediately return a boolean result by checking signature length and
performing verification of all signatures in the transaction in a
batch mode.


### Stack instructions

#### roll

_x[n] x[n-1] ... x[0] n_ **roll** → _x[n-1] ... x[0] x[n]_

1. Pops an int `n` from the contract stack.
2. Reduces `vm.runlimit` by `n`.
3. Looks past `n` items from the top, and moves the next item to the top of the contract stack.

Fails execution if:
* `n` is negative;
* stack has fewer than `n+1` items remaining.

Note: `0 roll` is a no-op, `1 roll` swaps the top two items.

#### bury

_x[n] ... x[1] x[0] n_ **bury** → _x[0] x[n] ... x[1]_

1. Pops an int `n` from the contract stack.
2. Reduces `vm.runlimit` by `n`.
3. On the contract stack, moves the top item past the `n` items below
   it.

Fails execution if:
* `n` is negative;
* stack has fewer than `n+1` items remaining.

Note: `0 bury` is a no-op, `1 bury` swaps the top two items.

#### reverse

_x[n-1] ... x[0] n_ **reverse** → _x[0] ... x[n-1]_

1. Pops a int `n` from the contract stack.
2. Reduces `vm.runlimit` by `n`.
3. On the contract stack reverses the top `n` items in-place.

Note: `0 reverse` and `1 reverse` are no-ops, `2 reverse` swaps the
top two items.

#### get

Argument stack: _item_ **get** → ø

Contract stack: **get** → _item_

1. Pops an item from the argument stack.
2. Pushes that item to the contract stack.

#### put

Contract stack: _item_ **put** → ø

Argument stack: **put** → _item_

1. Pops an item from the contract stack.
2. Pushes that item to the argument stack.

#### depth

**depth** → _count_

1. Counts the number of items `n` on the argument stack.
2. Pushes the [int](#int) `n` to the contract stack.

#### prv

**prv** → ø

Fails execution.

Note: This instruction is intended to convey TxVM code to a secure CPU
enclave or other private execution context.

#### ext

_item_ **ext** → ø

Drops [plain data item](#plain-data) `item`.

Fails execution if the `vm.extension` flag is `false`.

Note: `x ext` acts as a NOP which can be assigned some functionality
in the future. If `x` is a [smallint](#smallint), `x ext` becomes a
compact 2-byte instruction with code `x`. `x` can also be a string or
a tuple containing both the instruction code and the actual argument
for that instruction.


### Control flow instructions

#### verify

_cond_ **verify** → ø

1. Pops a boolean `cond` from the contract stack.
2. Halts VM execution if it is equal to 0.

#### jumpif

_cond offset_ **jumpif** → ø

1. Pops an integer `offset` from the contract stack.
2. Pops a boolean `cond` from the contract stack.
3. If `cond` is false, does nothing.
4. If `cond` is true:
    1. Adds `offset` to the current [program counter](#running-programs).
    2. Fails if the resulting program counter is negative.
    3. Fails if the resulting program counter is greater than the length of the current program.

Note 1: The program counter has already been advanced and points to
the instruction after the `jumpif` before `offset` is added to it.

Note 2: Normally the program using `jumpif` would be written as
`<cond> <offset> jumpif`, but for convenience the TxVM assembly
language permits symbolic jumps using this syntax:

    <cond> jumpif:$<destination>

where `<destination>` is the name of a label somewhere in the
program. The label itself is marked as `$<destination>` among the
instructions. Example:

    <cond> jumpif:$xyz  ... $xyz ...

Note 3: Unconditional jumps can be implemented as:

    1 jumpif:$<destination>

The TxVM assembly language abbreviates this as `jump:$<destination>`.

#### exec

_args... program_ **exec** → _results..._

1. Pops a string `program` from the contract stack.
2. [Runs](#running-programs) `program`.

#### call

Caller’s contract stack: _contract|wrappedcontract_ **call** → ø

Argument stack: _args..._ **call** → _results..._

1. Pops a [contract](#contracts) or
   [wrapped contract](#wrapped-contracts) `contract` from the contract
   stack. Changes the type to [contract](#contracts) if needed.
2. Saves the values of `vm.currentcontract` and `vm.caller`.
3. Sets `vm.caller` to `vm.currentcontract.seed`.
4. Sets `vm.currentcontract` to `contract`.
5. [Runs](#running-programs) `contract.program`.
6. Fails execution if `vm.unwinding` is false, but the contract stack
   is not empty.
7. If `vm.unwinding` is true, it is set to false.
8. Sets `vm.currentcontract` and `vm.caller` to the values saved at
   step 2.

Note: A contract that finishes execution normally (without
[output](#output), [wrap](#wrap) or [yield](#yield)) must have an
empty contract stack. Such a contract is discarded in step 8.

#### yield

Contract stack: _stack items..._ _program_ **yield** → _stack items..._

Argument stack: **yield** → _contract_

1. Pops a string `program` from the contract stack.
2. Sets `contract.program` to `program`.
3. Pushes `vm.currentcontract` to the argument stack.
4. Sets `vm.unwinding` to true.

#### wrap

Contract stack: _stack items..._ _program_ **wrap** → _stack items..._

Argument stack: **wrap** → _wrappedcontract_

1. Fails execution if any item on the contract stack is not
   [portable](#portable-types).
2. Pops a string `program` from the contract stack.
3. Sets `contract.program` to `program`.
4. Changes the type of `vm.currentcontract` to
   [wrapped contract](#wrapped-contracts).
5. Pushes the wrapped contract to the argument stack.
6. Sets `vm.unwinding` to true.

#### input

_snapshot_ **input** → _contract_

1. Pops a [tuple](#tuple) `snapshot` from the current contract stack.
2. [Creates entry](#entry-cost) `c` of type [contract](#contracts) such
   that `snapshot` is the [conversion](#conversion) of `c`.
3. Pushes `c` to the current contract stack.
4. [Creates tuple](#tuple-cost) `in = {"I", vm.currentcontract.seed,
   inputid}` where `inputid` is the [snapshot ID](#snapshot-id) of
   `c`: `VMHash("SnapshotID", serialize(snapshot))`.
5. Writes `in` to the transaction log.

Fails execution if `vm.finalized` is true.

#### output

Contract stack: _stack items..._ _program_ **output** → _stack items..._

1. Fails execution if any item on the contract stack is not
   [portable](#portable-types).
2. Pops a string `program` from the contract stack.
3. Sets `vm.currentcontract.program` to `program`.
4. [Creates string](#string-cost) `snapshotstring =
   serialize(convert(vm.currentcontract))` using
   [serialization](#serialization) and [conversion](#conversion)
   algorithms.
5. [Creates tuple](#tuple-cost) `out = {"O", vm.caller, outputid}`
   where `outputid` is the [snapshot ID](#snapshot-id) of the current
   contract: `VMHash("SnapshotID", snapshotstring)`.
6. Writes `out` to the transaction log.
7. Sets `vm.unwinding` to true.

Fails execution if `vm.finalized` is true.

#### contract

_program_ **contract** → _contract_

1. Pops string `program` from the contract stack.
2. [Creates entry](#entry-cost) `contract` of type
   [contract](#contracts) with the given `program`, an empty contract
   stack, and a [contract seed](#contract-seed) computed from
   `program`.
3. Pushes `contract` to the current contract stack.

#### seed

_contract|wrappedcontract_ **seed** → _contract|wrappedcontract seed_

1. Looks at the [contract](#contracts) or
   [wrapped contract](#wrapped-contracts) `contract` on top of the
   contract stack.
2. [Copies](#copy-cost) `contract.seed` as string `seed`.
3. Pushes `seed` to the contract stack.

#### self

**self** → _seed_

1. [Copies](#copy-cost) `vm.currentcontract.seed` as string `seed`.
2. Pushes `seed` to the contract stack.

Note: The contract seed of the [witness program](#witness-program)
(i.e., the top-level contract) is an all-zero 32-byte string.

#### caller

**caller** → _seed_

1. [Copies](#copy-cost) `vm.caller` as string `seed`.
2. Pushes `seed` to the contract stack.

Note: `vm.caller` for the [witness program](#witness-program) (i.e.,
the top-level contract) is an all-zero 32-byte string.

#### contractprogram

**contractprogram** → _program_

1. [Copies](#copy-cost) `vm.currentcontract.program` as string `prog`.
2. Pushes `prog` to the contract stack.

#### timerange

_min max_ **timerange** → ø

1. Pops an integer `max` from the contract stack.
2. Pops an integer `min` from the contract stack.
3. [Creates tuple](#tuple-cost) `{"R", vm.currentcontract.seed, min,
   max}` and writes it to the transaction log.

Fails execution if `vm.finalized` is true.

#### log

_item_ **log** → ø

1. Pops a [plain data item](#plain-data) `item` from the contract stack.
2. [Creates tuple](#tuple-cost) `{"L", vm.currentcontract.seed, item}`
   and writes it to the transaction log.

Fails execution if `vm.finalized` is true.

#### peeklog

_i_ **peeklog** → _item_

1. Pops integer `i` from the contract stack.
2. [Copies](#copy-cost) `item` from index `i` in the transaction log
   (zero-based).
3. Pushes `item` to the contract stack.

Fails execution if index `i` is out of bounds (negative or ≥ than the length of
the log).

#### txid

**txid** → _txid_

1. Computes the [transaction ID](#transaction-id) from the items in
   the transaction log **without runlimit cost**.
2. [Copies](#copy-cost) the transaction ID to a string `txid`.
3. Pushes `txid` to the contract stack.

Fails execution if `vm.finalized` is false.

#### finalize

_value_ **finalize** → ø

1. Pops a [value](#values) `value` from the contract stack.
2. Fails execution if `value` is not a zero-amount value.
3. Fails execution if `vm.finalized` is true.
4. Sets `vm.finalized` to true.
5. [Creates tuple](#tuple-cost) `{"F", vm.currentcontract.seed,
   vm.version, value.anchor}` and writes it to the transaction log.

Note: after the transaction has been finalized, no additional items
can be added to the log.


### Value instructions

#### nonce

_blockid exp_ **nonce** → _a_

1. Pops int `exp` from the contract stack.
2. Pops a string `blockid` from the contract stack.
3. [Creates tuple](#tuple-cost) `nonce = {"N", vm.caller,
   vm.currentcontract.seed, blockid, exp}`.
4. [Creates tuple](#tuple-cost) `timerange = {"R", vm.currentcontract.seed, 0,
   exp}`.
5. [Creates entry](#entry-cost) `a` of type [value](#values):
    * `a.amount = 0`
    * `a.assetid = 0x000000...` (32 bytes, all zeroes)
    * `a.anchor = VMHash("Nonce", serialize(nonce))` (see
      [Encoding](#encoding) section).
6. Writes `nonce` to the transaction log.
7. Writes `timerange` to the transaction log.
8. Pushes `a` to the contract stack.

Fails execution if `vm.finalized` is true.

**Discussion**

A nonce serves two purposes:

1. It provides a unique anchor for an issuing transaction if other
   anchors are not available (i.e., from other values already on the
   stack);
2. It binds the entire transaction to a particular blockchain,
   protecting not only against cross-blockchain replays, but also
   potential blockchain forks due to compromise of the old
   block-signing keys.  This is a necessary feature for stateless
   signing devices that rely on blockchain proofs.

The `blockid` that is copied to the log in step 3 will be checked
against the IDs of “recent” blocks when the transaction is applied to
the blockchain. It must match a recent block, or the initial block of
the blockchain, or be a string of 32 zero-bytes.

Additionally, the `nonce` tuple is checked for uniqueness against
“recent” nonces.

To perform these checks, validators must keep a set of recent nonces
and a set of recent block headers available. For scalability and to
reduce resource demands on the network, these sets must be limited in
size. So block signers can and should impose reasonable limits on the
value of `exp` (which is the time by which the transaction must be
included in a block or become invalid).

As a special case for long-living pre-signed transactions, the
protocol allows a nonce to use the initial block’s ID regardless of
the `refscount` limit specified in the
[block headers](blockchain.md#block-header).

Another special case is an all-zero `blockid`.  This makes the nonce
replayable on another chain, but still unique within any one chain.

#### merge

_a b_ **merge** → _c_

1. Pops an item `b` of type [value](#values) from the contract stack.
2. Pops an item `a` of type [value](#values) from the contract stack.
3. [Creates entry](#entry-cost) `c` of type [value](#values):
    * `c.amount = a.amount + b.amount`
    * `c.assetid = a.assetid`
    * `c.anchor = VMHash("Merge", a.anchor || b.anchor)`
4. Pushes `c` to the contract stack.

Fails execution if `a.assetid` differs from `b.assetid`.

#### split

_a amount_ **split** → _b c_

1. Pops an integer `amount` from the contract stack.
2. Pops a [value](#values) `a` from the contract stack.
3. [Creates entry](#entry-cost) `b` of type [value](#values):
    * `b.amount = a.amount - amount`
    * `b.assetid = a.assetid`
    * `b.anchor = VMHash("Split1", a.anchor)`
4. [Creates entry](#entry-cost) `c` of type [value](#values):
    * `c.amount = amount`
    * `c.assetid = a.assetid`
    * `c.anchor = VMHash("Split2", a.anchor)`
5. Pushes `b`, then `c` to the contract stack.

Fails execution if:
* `amount` is negative;
* `amount` is greater than `a.amount`.

Note: It is possible to create a value whose amount is zero, with _<val>
0 split_ or _<val> amount split_. The `issue` and `finalize`
instructions consume a zero-valued amount for the unique anchor it
contains.

#### issue

_avalue amount assettag_ **issue** → _value_

1. Pops from the contract stack:
    1. string `assettag`,
    2. integer `amount`,
    3. [value](#values) `avalue`, which must have an amount of zero.
2. Computes [asset ID](#asset-id) `assetid` with the tag `assettag` and current contract’s seed `vm.currentcontract.seed`.
3. [Creates entry](#entry-cost) `v` of type [value](#values):
    * `v.amount = amount`
    * `v.assetid = assetid`
    * `v.anchor = avalue.anchor`
4. Pushes `v` to the contract stack.
5. [Creates tuple](#tuple-cost) `{"A", vm.caller, v.amount, v.assetid, v.anchor}` and writes it to the transaction log.

Fails execution if:
* `vm.finalized` is true;
* `avalue` is not a zero-amount value;
* `amount` is negative.

Note: issuance uses contract seed instead of the current executing program (or `vm.currentcontract.program`) to allow issuance of the same asset ID from different states of the contract. Conversely, to allow issuance of more than one asset ID by the same contract, several `assettag` strings can be used to generate distinct asset IDs.

#### retire

_value_ **retire** → ø

1. Pops an item `value` of type [value](#values) from the contract stack.
2. [Creates tuple](#tuple-cost) `{"X", vm.currentcontract.seed, value.amount, value.assetid, value.anchor}` and writes it to the transaction log.

Fails execution if `vm.finalized` is true.

#### amount

_value_ **amount** → _value amount_

1. Looks at [value](#values) `v` at the top of the contract stack.
2. Pushes `v.amount` to the contract stack.

#### assetid

_value_ **assetid** → _value assetID_

1. Looks at [value](#values) `v` at the top of the contract stack.
2. [Copies](#copy-cost) `v.assetid` as string `assetid`.
3. Pushes `assetid` to the contract stack.

#### anchor

_value_ **anchor** → _value anchor_

1. Looks at [value](#values) `v` at the top of the contract stack.
2. [Copies](#copy-cost) `v.anchor` as string `anchor`.
3. Pushes `anchor` to the contract stack.


### Data instructions

#### eq

_x y_ **eq** → _bool_

Tests two plain data items for equality.

1. Pops item `y` from the contract stack.
2. Pops item `x` from the contract stack.
3. If they have differing types, pushes int `0` to the stack.
4. If they have the same type:
    1. if they are tuples, pushes int `0` to the stack;
    2. otherwise, if they are equal, pushes int `1` to the stack;
    3. otherwise, pushes int `0` to the stack.

Note: [entry types](#entry-types) are not allowed to be compared directly with `eq`
in order to avoid dropping them from the stack without authorization.

#### dup

_item_ **dup** → _item item_

1. Peeks at the top item on the contract stack, `item`.
2. [Copies](#copy-cost) the `item`.
3. Pushes a copy of `item` to the contract stack.

Fails if `item` is not a [plain data item](#plain-data).

#### drop

_item_ **drop** → ø

1. Pops an item `item` from the contract stack.

Fails if `item` is neither a [plain data item](#plain-data) nor a
zero-amount [value](#values).

#### peek

_n_ **peek** → _item_

1. Pops an int `n` from the contract stack.
2. Looks at the `n`th item, `item` on the contract stack.
3. [Copies](#copy-cost) `item` and pushes it to the contract stack.

Fails if `item` is not a [plain data item](#plain-data).

Note: `0 peek` is equivalent to [dup](#dup).

#### tuple

_x[0] ... x[n-1] n_ **tuple** → _{ x[0], ..., x[n-1] }_

1. Pops an integer `n` from the contract stack.
2. Pops `n` plain data items from the contract stack.
3. [Creates tuple](#tuple-cost) `t` with these items. The topmost item
   on the stack becomes the last item in the tuple.
4. Pushes `t` to the contract stack.

#### untuple

_{x[0] ... x[n-1]}_ **untuple** → _x[0] ... x[n-1] n_

1. Pops a tuple `tuple` from the contract stack.
2. Reduces `vm.runlimit` by `n`, the length of the tuple.
3. Pushes each of the fields in `tuple` to the stack in order (so that
   the last item in the tuple ends up on top of the stack).
4. Pushes int `n`, the length of the tuple, to the contract stack.

#### len

_item_ **len** → _length_

1. Pops a string or tuple `item` from the contract stack.
2. Pushes int `n` to the contract stack:
    1. If `item` is a tuple, `n` is the the number of items in that tuple.
    2. If `item` is a string, `n` is the length of that string in bytes.

#### field

_tuple i_ **field** → _contents_

1. Pops an integer `i` from the top of the contract stack.
2. Pops [tuple](#tuple) `tuple`.
3. [Copies](#copy-cost) the `item` stored in the `i`th field of
   `tuple`.
4. Pushes `item` to the contract stack.

Fails if `i` is negative or greater than or equal to the number of
fields in `tuple`.

#### encode

_item_ **encode** → _string_

1. Pops a [plain data item](#plain-data) `item` from the contract stack.
2. [Creates string](#string-cost) `s` being a [serialized](#serialization) copy of `item`.
3. Pushes `s` to the contract stack

Note: use `peeklog encode` to sign portions of the transaction log.

#### cat

_a b_ **cat** → _a||b_

1. Pops string `b` from the contract stack.
2. Pops string `a` from the contract stack.
3. [Creates string](#string-cost) `a||b` as concatenation of `a` and `b`.
4. Pushes the result `a||b` to the contract stack.

#### slice

_str start end_ **slice** → _str[start:end]_

1. Pops two integers, `end`, then `start`, from the contract stack.
2. Pops a string `str` from the contract stack.
3. [Creates string](#string-cost) `str[start:end]` (with the first
   character being the one at index `start`, and the last character
   being the one before index `end`).
4. Pushes the resulting string `str[start:end]` to the contract stack.

Fails execution if:
* `start > end`;
* `start < 0`;
* `end > len(str)`.

#### bitnot

_a_ **bitnot** → _~a_

1. Pops a string `a` from the contract stack.
2. [Creates string](#string-cost) `~a` by inverting the bits of `a`.
3. Pushes the resulting string `~a` to the contract stack.

#### bitand

_a b_ **bitand** → _a&b_

1. Pops two strings `a` and `b` from the contract stack.
2. [Creates string](#string-cost) by performing a “bitwise and”
   operation on `a` and `b`.
3. Pushes the result `a & b` to the contract stack.

Fails execution if `len(a) != len(b)`.

#### bitor

_a b_ **bitor** → _a|b_

1. Pops two strings `a` and `b` from the contract stack.
2. [Creates string](#string-cost) by performing a “bitwise or”
   operation on `a` and `b`.
3. Pushes the result `a | b` to the contract stack.

Fails execution if `len(a) != len(b)`.

#### bitxor

_a b_ **bitxor** → _a^b_

1. Pops two strings `a` and `b` from the contract stack.
2. [Creates string](#string-cost) by performing a “bitwise xor”
   operation on `a` and `b`.
3. Pushes the result `a ^ b` to the contract stack.

Fails execution if `len(a) != len(b)`.

#### pushdata

**(0x5f+)immediatedata** → _string_

1. [Creates string](#string-cost) `string` from the bytes
   `immediatedata` following the instruction code.
2. Pushes `string` to the contract stack.

All instructions with code equal or greater than **0x5f** (**95** in
decimal) are instructions pushing a string to the contract stack.  The
string being pushed follows immediately after the instruction code.
The length of the string is `n - 95`, where `n` is the instruction
code.

Note 1: Code 0x5f pushes an empty string, code 0x60 is followed by a
1-byte string that is pushed to the stack, code 0x61 is followed by a
2-byte string, etc.

Note 2: Strings from 0 to 32 bytes in length require a 1-byte
instruction code (from 0x5f through 0x7f). Strings of 33 bytes and
longer use multi-byte opcodes because higher numbers occupy more than
1 byte in [LEB128](https://en.wikipedia.org/wiki/LEB128) encoding.



## Examples

### Deferred programs

Programs can be deferred simply by creating a contract out of a
program string.

    [<conditions...>] contract

(In TxVM assembly language, a sequence of instructions enclosed in
square brackets denotes the bytecode string resulting from assembling
those instructions.)

Normally, a deferred program is created within some contract and needs
to be passed out to the caller by [putting](#put) it on the argument
stack:

    [<conditions...>] contract put

For example, a contract that runs before a transaction is
[finalized](#finalize), but that requires a signature check against
the [transaction ID](#transaction-id) (available only after
finalization), can defer a signature check for later like so:

    <normal contract actions...> [txid <pubkey> get 0 checksig verify] contract put

The normal contract actions execute, then a new contract containing
the signature check is created and returned to the caller. That new
contract can be executed (via [call](#call)) at any time after
`finalize`. There is no danger of skipping the signature check, since
the contract remains on the stack until it is invoked and runs to
completion, and stacks must be cleared for the transaction to be
valid.

Note: It’s also possible to return the signature check program as a
bare string (not a contract):

    <normal contract actions...> [txid <pubkey> get 0 checksig verify] put

The caller would invoke this with [exec](#exec) rather than
[call](#call). But because this is a plain string of bytes rather than
a contract, there is no prohibition on simply dropping it from the
stack, so there is no guarantee in this case that the signature check
won’t be skipped.

### Pay To Public Key

In **Pay To Public Key** (P2PK) programs, some value is locked in a
contract when first called. The next call of the contract unlocks the
value and emits a deferred program checking the signature of the
transaction ID.

    <value> put [get [put [txid <pubkey> get 0 checksig verify] contract put] output] contract call

Explaining this example from the inside out:

    ... [txid <pubkey> get 0 checksig verify] contract ...

This is the deferred signature check from [the previous section](#deferred-programs).

    ... [put [txid <pubkey> get 0 checksig verify] contract put] output ...

This persists the contract to the global blockchain state (with
`output`) and suspends its execution. When it next runs (via `input`)
it will invoke this new program, which does two things:
* returns an item from the contract’s stack to the argument stack (via `put`);
* constructs a deferred signature check and returns that to the argument stack too.

    <value> put [get [...] output] contract call

This `put`s a value from the contract stack to the argument
stack. Then it constructs a contract and `call`s it. The constructed
contract `get`s the value from the argument stack and then persists
itself (together with the value it just consumed) to the global
blockchain state.

### Signature programs

**Signature programs** (P2SP) differ from
[P2PK programs](#pay-to-public-key) in that another program is
signed instead of the transaction ID.

The value is locked in a contract on the first call. The second call
releases the value and defers a signature check. The signature check
is against a program string that is executed.

P2SP program:

    [get dup <pubkey> get 0 checksig verify exec]

The first `get` consumes a program string from the argument stack. The
second `get` consumes a signature.

Usage:

    <value> put [get [get dup <pubkey> get 0 checksig verify exec] output] contract call

Breaking that down:

    <value> put                # move value to the argstack
    [                          #
        get                    # get value to the contract stack
        [                      #
            get dup            # gets <prog> <prog>
            <pubkey>           # pushes <pubkey>
            get                # gets <sig>
            0 checksig verify  # checks signature of a program (copy prog from below pubkey and sig)
            exec               # executes the program
        ]                      # saves the next state of the contract that checks the signature
        output                 # publishes the contract
    ] contract call            # creates a contract from the code string and calls it to let it store the value

The value can be unlocked later with a signature `sig` and a signed
program `prog`. Program `prog` will execute on a contract stack that
has `value`. If `prog` is:

    txid <X> eq verify

then this is equivalent to the simple P2PK case above: instead of a
signature covering transaction ID `X`, it covers a short program
saying “the transaction ID must be `X`.” But the signature program may
include other conditions too.

Note that for extra safety against key reuse, the value’s anchor
should also be covered by the signature. Here is a safer P2SP program
signing `program||value.anchor` instead of `program`:

                  contract stack                                      arg stack
                  --------------                                      ---------
                  [... <value>]                                       [... prog sig]
    anchor        [... <value> value.anchor]                          [... prog sig]
    get dup       [... <value> value.anchor prog prog]                [... sig]
    2 roll        [... <value> prog prog value.anchor]                [... sig]
    cat           [... <value> prog (prog||value.anchor)]             [... sig]
    <pubkey>      [... <value> prog (prog||value.anchor) pubkey]      [... sig]
    get           [... <value> prog (prog||value.anchor) pubkey sig]
    0 checksig    [... <value> prog <checksig-result>]
    verify        [... <value> prog]
    exec

### Single-asset transfer

The following example unlocks 3 outputs, re-allocates values and
creates 2 new outputs using [P2SP](#signature-programs) redeemed with
txid-signing programs.

All values have the same asset ID.

    # Unlock 3 outputs, move each value from argstack, but keep
    # deferred contracts in the argstack until tx is finalized.

    [[txid <txid> eq verify] contract put put]  # shared signature program with embedded txid, always on top after each input

    <sig1> put dup put
        {"C", <id1>, [get dup <pubkey1> get 0 checksig verify exec], {"V", <amount1>, <anchor1>}} input
    call get 1 roll

    <sig2> put dup put
        {"C", <id2>, [get dup <pubkey2> get 0 checksig verify exec], {"V", <amount2>, <anchor2>}} input
    call get 1 roll

    <sig3> put dup put
        {"C", <id3>, [get dup <pubkey3> get 0 checksig verify exec], {"V", <amount3>, <anchor3>}} input
    call get 1 roll

    drop # drop the signature program

    merge merge        # merge all unlocked values, all deferred programs are left on the argstack

    # Split the merged amount into two new amounts and lock with the new public keys:

    <amount4> split
    put [get [get dup <pubkey4> get 0 checksig verify exec] output] contract call

    <amount5> split
    put [get [get dup <pubkey4> get 0 checksig verify exec] output] contract call

    # Assuming amount4+amount5 == amount1+amount2+amount3, the
    # previous split should have left a zero value on the stack. This
    # is consumed by finalize for its anchor.

    finalize # use the remaining zero value to anchor the finalized transaction

    # pop all deferred contracts from the argstack
    get call
    get call
    get call

### Secure nonce

The goal of a [nonce](#nonce) is to create a new globally unique
anchor in the transaction. Recent nonces are remembered in the
blockchain state and each new one is checked for uniqueness against
the existing ones. (The amount of storage is bounded by a mandatory
expiration time.)

Do not create nonces with simply a randomly generated string:

    [<random> drop <blockchainid> <exp> nonce put] contract call  # THIS IS NOT SAFE

Such nonces are vulnerable to _sniping_: before the transaction is
confirmed, another transaction may steal that nonce and get committed
to a block ahead of it, making the original transaction invalid.

In the best case, the transaction simply has to be re-built, re-signed
and re-submitted.  In the worst case, a chain of unconfirmed
transactions spending the current one will break, potentially leading
to a loss of funds (depending on specifics of a higher-level
protocol).

The safe way to make a nonce is to generate a random public key and
sign the transaction ID with it:

    [
       [txid <pubkey> get checksig verify] contract put
        <blockchainid> <exp> nonce put
    ] contract call

That contract emits a deferred program and a new anchor. The deferred
program checks the transaction signature with a random public key that
simultaneously acts as a source of entropy for the nonce.

### Nonce and issuance

If a nonce has to be generated within the issuance contract, we can
avoid doing an additional signature verification by using
[nonce](#nonce) in the same contract that does [issue](#issue):

    [... nonce ... issue] contract  # safe, but subject to race conditions

Unfortunately, the contract seed must be stable in order to have a
reusable asset ID, but unique for the nonce. (Expiration time entropy
may be not enough to avoid race conditions.)

The way around is to bind [nonce](#nonce) to the issuance contract not
internally, but externally by checking that the [caller](#caller)
contract seed matches the built-in one. This enables us to randomize
the nonce-wrapping contract with a plain random string:

    [
        <random 16 bytes> drop
        <issuancecontractseed> caller eq verify
        <blockchainid> <exp> nonce put
    ] contract
    put
    [
        get call get # get the nonce-generating contract, call it, and get the anchor out of it
        ...
        issue
    ] contract

Nonce is wrapped it its own contract that has a built-in random string
with 128 bits of entropy, and a check that it is called from a correct
issuance program (so it cannot be sniped).  That contract is then
passed to the actual issuance contract which calls it and gets the
anchor out of it.

Note: the issuance contract itself must be protected from sniping by,
for instance, verifying a signature over the transaction ID.

### More examples

TBD:

* How to perform "read" action (reuse anchor)?
* How to make offer (1 AAPL → 150 USD)?
* How to make collateralized loan?
* How to make a p2sh clause?
* How to make a merkleized clause?
* How to make stateful bond/coupons contract?
* How to make treasury/bond/coupons coordination?
* How to make a fully issuer-defined instrument, governing the entire lifecycle? Use `wrap`.
* How to make two independent contracts get unlocked/destroyed atomically? Use contract that does that and authorized via `caller`.
