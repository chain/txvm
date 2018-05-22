# Cryptocurrency

Two properties of a successful cryptocurrency are:

* The supply of tokens is limited
* The mechanism for distributing new tokens is widely understood to be
  fair

In TxVM terms, this means defining an asset type whose issuance
contract can be used by anyone to create new tokens, but with limits
on how many may be created and when.

In this document we’ll develop an issuance contract that anyone can
use to create N new tokens as long as no tokens have been created in
the past N hours.

## Persistent storage

At once we’re confronted with the problem of remembering, from one
transaction to the next, the time of the latest issuance, for
calculating how many units the issuance contract should allow to be
created.

In TxVM, recording state between transactions is the province of
contracts. When a contract executes the `output` instruction, a hash
of its complete state, including the contents of its stack, is added
to the utxo set. When some later transaction reconstitutes that
contract with `input`, it can access the stack items that were
recorded earlier.

(Normally, `output` and `input` are used for storing and spending
Values, respectively, but that is just one specific use of TxVM
contracts, which turn out to be much more general.)

The standard TxVM issuance contract is stateless, but that doesn’t
mean this one has to be. Let’s say that our cryptocurrency issuance
contract must be persistent, and must keep on its stack the timestamp
of the latest issuance (in milliseconds since the epoch).

This means that the issuance contract must be constructed with an
initial timestamp before it can be used. Once it’s constructed, it
must `output` itself to the blockchain state. When some later
transaction wants to use it to issue tokens of the cryptocurrency,
that transaction must first `input` the contract from the blockchain
state before calling it.

Here’s how such a contract can be constructed with an initial
timestamp.

```
INITIAL put
[get [...BODY...] output] contract
call
```

Here, INITIAL is an integer, the initial timestamp. The first line
puts that value onto the argument stack.

The next line creates a contract by supplying a program as a sequence
of assembly-language instructions enclosed in square brackets (meaning
“the string of bytecode that these instructions assemble to”) and
invoking `contract` on it.

The final line invokes the new contract.  When it does, it performs
these actions:

```
get
```

This moves INITIAL from the argument stack to the new contract’s
stack.

```
[...BODY...] output
```

Here, BODY is the main body of the issuance contract (which we’ll
develop below). This line `output`s the contract, together with the
timestamp on its stack, to the blockchain state, specifying BODY as
the code to execute the next time the contract is called.  That BODY
will include an `issue` instruction but will also include logic that
prevents issuance of too many tokens too soon after the last issuance
denoted by the timestamp.

## There can be only one

One problem with this approach is that anyone may create an instance
of this contract with a false timestamp in it, allowing them to
circumvent the one-per-hour rule. What we need is some way to
designate a single instance of this contract as authentic and prevent
any others from working.

For this we can use TxVM _anchors_, which are the mechanism for
guaranteeing uniqueness on the blockchain. An anchor is a string that
lives inside every Value object.

There are three ways that Value objects get created:

* By the `issue` instruction. This instruction consumes a zero-Value
  as an argument, then rehashes its anchor to get the anchor for the
  new Value.
* By the `split` instruction, which rehashes the anchor in the
  original Value in two different ways, to get the anchors for the two
  new Values that result from the split; and by the `merge`
  instruction, which rehashes the anchors in the two original Values
  to get the anchor for the new Value that results.
* By the `nonce` instruction, which uses the blockchain ID (the hash
  of the initial block) plus a timestamp to create the hash for the
  new zero-Value it produces. Extra block-level rules ensure that
  `nonce` cannot create duplicate zero-Values.

In short there is no way to produce two Values with the same anchor.

So let’s redefine the issuance contract to store an anchor on its
stack in addition to a timestamp. To make sure it _is_ an anchor (and
not just any old string that could be forged), let’s have the
contract’s constructor phase consume a zero-Value and extract the
anchor string from that itself. And here’s the key: the anchor string
will double as the _asset tag_ argument to `issue`.

The asset tag is part of what determines the type of asset being
issued. The other thing that determines the asset type is the _seed_
(a hash) of the contract doing the issuing.

So if someone tries to create a duplicate issuance contract, they will
necessarily use a different zero-Value to construct it, thus getting a
different anchor string, thus using a different asset tag, thus
issuing a different asset — not the desired cryptocurrency.

If they write a new issuance contract that tries to mimic the one
we’ve designed here, such that it can present a forged asset tag to
the `issue` instruction, it will necessarily use different
instructions, this having a different contract seed, thus issuing a
different asset — not the desired cryptocurrency.

Here’s the new constructor phase for the contract. It consumes a
zero-Value and an initial timestamp from the argument stack. It
extracts and stores the anchor string from the zero-Value, but it also
keeps the zero-Value around as a source of _new_ zero-Values, which
the main body of the contract will need in order to perform issuances.

```
INITIAL put
BCID EXP nonce put
[get anchor get [...BODY...] output] contract
call
```

Here, BCID is the blockchain ID and EXP is a timestamp (in
epoch-milliseconds in the not-too-distant future) when block-level
uniqueness guarantees for the resulting zero-Value can expire.

## The timerange instruction

A TxVM transaction does not have a way to query the current
time. However, blockchain blocks have timestamps, and a transaction
can instruct a block to exclude it unless that timestamp lies within a
specific range. This is done with the `timerange` instruction, which
works by adding an entry to the _transaction log_. That log is
inspected by a block when considering whether to include a
transaction.

When issuing N tokens, we’ll add N hours to the timestamp in the
issuance contract, then use that as the lower bound for the
`timerange` instruction. We’ll leave the upper bound unspecified.

Thus, if the timestamp says “five hours ago” and we’re trying to issue
two tokens, we’ll update the timestamp to “three hours ago” and
specify a timerange that began then. This transaction should have no
trouble being published on the blockchain.

On the other hand, if the timestamp says “two hours ago” and we’re
trying to issue five tokens, we’ll update the timestamp to “three
hours in the future” and specify a timerange that doesn’t begin until
then. This transaction will have to wait before a block will include
it. And in the meantime, someone may beat it to the punch by issuing a
single token, which in this scenario can be done right away.

## The main body

At last we’re ready to write the main body of the issuance contract,
the part referred to as BODY above. The contract stack begins with the
items placed there during constructor phase. The argument stack
contains the number of tokens the caller wishes to issue.

```
                          contract stack                                         arg stack
                          --------------                                         ---------
                          [zeroval anchor timestamp]                             [amount]
get dup                   [zeroval anchor timestamp amount amount]               []
2 bury                    [zeroval anchor amount timestamp amount]               []
3600000 mul               [zeroval anchor amount timestamp hours]                []
add                       [zeroval anchor amount newtimestamp]                   []
dup 0                     [zeroval anchor amount newtimestamp newtimestamp 0]    []
timerange                 [zeroval anchor amount newtimestamp]                   []
3 roll                    [anchor amount newtimestamp zeroval]                   []
splitzero                 [anchor amount newtimestamp zeroval1 zeroval2]         []
3 roll                    [anchor newtimestamp zeroval1 zeroval2 amount]         []
4 roll dup                [newtimestamp zeroval1 zeroval2 amount anchor anchor]  []
3 bury                    [newtimestamp zeroval1 anchor zeroval2 amount anchor]  []
issue put                 [newtimestamp zeroval1 anchor]                         [issuedval]
2 roll                    [zeroval1 anchor newtimestamp]                         [issuedval]
contractprogram output
```

The next-to-last line resets the items on the stack to the order
expected for the next call to the contract (zeroval, anchor, timestamp).

The final line outputs the updated contract back to the blockchain
state, specifying that this same program should run the next time the
contract is called.

You might be wondering: why store both a zeroval and the anchor
string, when the anchor can be pulled out of a zeroval at any time
with the `anchor` instruction? The answer is that the zeroval changes
with every call to the contract. We need an unchanging copy of the
_original_ anchor string to use as an asset tag, so that calling the
contract issues tokens of the same asset type every time.

## Competing issuances

Each time the issuance contract is used, it is destroyed! ...and
replaced with an updated one. The updated one has new stack contents
and a new “snapshot hash” (not to be confused with the seed, a hash
that doesn’t change).

If two different transactions are both trying to issue cryptocurrency
tokens at the same time, they will both have to `input` the same
issuance contract before calling it.

Two transactions trying to `input` the same item from the blockchain
state is a classic double-spend. At most one of these transactions can
be added to the blockchain. Which one wins and which one loses is a
problem addressed by the blockchain network’s _consensus_ mechanism,
which is outside the scope of TxVM.

Whichever transaction loses can try again, but will first have to
learn (by monitoring the blockchain) the new identity and contents of
the issuance contract, recreating its transaction but with revised
arguments to `input`.
