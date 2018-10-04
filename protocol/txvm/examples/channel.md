# Payment channel

A payment channel is a way for a group of blockchain participants to
transact quickly and privately among themselves by avoiding the
blockchain for all but the first and last of a series of transactions.

Members pay into a special contract that holds their funds for the
lifetime of the channel. They conduct a series of off-chain
transactions using only the amounts in the channel. Each transaction
affects the members’ balances. When, by mutual agreement, the members
“close” the channel, the updated balances are paid back to the members
on-chain. For the intervening transactions, blockchain settlement
times are avoided, and transaction details are not disclosed to other
parties.

A payment channel contract must include certain safeguards. It must be
impossible for a member to claim funds they don’t own, whether by
falsifying the balances when closing the channel or by closing the
channel at a point other than the latest in its history. A member must
also be unable to hold the other members’ funds hostage by refusing to
close the channel in a timely fashion.

For simplicity, this example is limited to two channel members
transacting with a single asset type. We’ll refer to the two members
as Alice and Bob per convention.

## Constructor

The constructor phase of our contract needs to take a Value object
(the amount of some asset held by the channel), plus Alice’s and Bob’s
public keys. The Value object is presumably the result of a `merge` of
some of Alice’s money and some of Bob’s.

```
<value> put
<alice key> put
<bob key> put
[get get get [...main body...] output] contract call
```

## Closing the channel

The main body of the contract has one job: closing the channel and
releasing the proper amounts to Alice and to Bob. This suggests its
arguments should be the amounts to send to Alice and Bob, which should
add up to the total held in the channel. The transaction should
require a signature from Alice and one from Bob to authorize the
closure and disbursement of funds.

```
                                contract stack                                             arg stack
                                --------------                                             ---------
                                value alicePubkey bobPubkey                                aliceAmount bobAmount
2 roll                          alicePubkey bobPubkey value                                aliceAmount bobAmount
get                             alicePubkey bobPubkey value bobAmount                      aliceAmount
split                           alicePubkey bobPubkey remainingValue bobValue              aliceAmount
swap                            alicePubkey bobPubkey bobValue remainingValue              aliceAmount
get                             alicePubkey bobPubkey bobValue remainingValue aliceAmount
split                           alicePubkey bobPubkey bobValue zeroValue aliceValue
put                             alicePubkey bobPubkey bobValue zeroValue                   aliceValue
3 roll                          bobPubkey bobValue zeroValue alicePubkey                   aliceValue
dup                             bobPubkey bobValue zeroValue alicePubkey alicePubkey       aliceValue
put                             bobPubkey bobValue zeroValue alicePubkey                   aliceValue alicePubkey
[...send...] contract call      bobPubkey bobValue zeroValue alicePubkey
put                             bobPubkey bobValue zeroValue                               alicePubkey
[...checksig...] contract call  bobPubkey bobValue zeroValue                               aliceSigChecker
drop                            bobPubkey bobValue                                         aliceSigChecker
put                             bobPubkey                                                  aliceSigChecker bobValue
dup put                         bobPubkey                                                  aliceSigChecker bobValue bobPubkey
[...send...] contract call      bobPubkey                                                  aliceSigChecker
put                                                                                        aliceSigChecker bobPubkey
[...checksig...] contract call                                                             aliceSigChecker bobSigChecker
```

Here, `[...send...]` is the contract described in
[the Orderbook example](orderbook.md) for locking up a payment with a
recipient’s public key, and `[...checksig...]` is the
signature-checking contract described in
[the Account example](account.md).

Note, this does not explicitly check that `aliceAmount+bobAmount`
equals the amount of the Value object in the contract, but it does
check _implicitly_. If `aliceAmount+bobAmount` exceeds the amount in
the Value, then one of the `split` instructions will fail. If
`aliceAmount+bobAmount` is too small, then the `zeroValue` above will
contain an amount other than zero, and the `drop` instruction will
fail.

## Unilateral closing

The channel-closing clause above works only when Alice and Bob
cooperate, each adding a signature to the transaction so they can both
recover their money from the channel. But suppose Alice is ready to
close the channel and can’t get Bob’s cooperation for some reason. She
should be able to make an assertion about the current balances in the
channel and close it herself, disbursing both her funds and Bob’s.

This suggests that each time Alice and Bob transact off-chain, they
exchange signatures on a message attesting to the current
balances. This message could be used in an alternative close-channel
clause that does not require fresh signatures on the full transaction,
as the clause above does. Thus Alice or Bob could unilaterally close
the channel at any time. In fact they should exchange signatures
attesting to the opening balances when creating the channel, in case
the channel needs to be unilaterally closed even before any
transactions take place.

Let’s use a TxVM tuple to represent the message attesting to the
current balances: `{aliceAmount, bobAmount}`. The TxVM `encode`
instruction can turn this into a string, and this string is what we’ll
expect Alice and Bob to sign.

## Latest statement

This approach introduces a new problem. There is no guarantee that the
signed statement closing the channel and disbursing the funds is the
_latest_ such statement. Alice may send some funds to Bob on Monday,
and Bob may send some funds to Alice on Tuesday... and then try to
close the channel with _Monday’s_ statement instead of Tuesday’s,
illicitly undoing the payment he is supposed to have made to Alice.

To fix this, we’ll add a transaction counter to the channel. Each
off-chain transaction must now increment the counter, and Alice and
Bob must each sign a message containing Alice’s balance, Bob’s
balance, and the current value of the counter. In other words the
string to sign is now the `encode` of the tuple `{aliceAmount,
bobAmount, counter}`.

If Bob tries to pull a fast one on Alice by unilaterally closing their
channel with an outdated statement, all Alice needs to do is show a
signed statement with a higher counter in it. This means that when a
channel is unilaterally closed, there must be a delay before the
channel’s funds are disbursed, giving Alice time to respond, if
necessary, with proof of Bob’s fraud.

## Uniqueness

There’s one more wrinkle to address to make this solution
robust. Suppose Alice and Bob had a different payment channel in the
past. If a statement from the old channel happened to have the right
counter in it, and the right `aliceAmount+bobAmount` total, and a
balance that favors Bob (that sneaky devil), he might try to reuse
that old statement to unilaterally close the _new_ channel (since it’s
already signed by Alice) and pay him more than his fair share.

To prevent this, let’s store a unique nonce in the channel contract
and require that every signature now cover the tuple `{aliceAmount,
bobAmount, counter, nonce}`. This ensures that signatures from old
channels cannot be reused in other channels. As discussed in
[the Cryptocurrency example](cryptocurrency.md), the TxVM way to do
this is by extracting the _anchor_ from a Value object.

Conveniently, we can use the anchor inside the Value that is passed to
the channel contract’s constructor. With that decided, we’re ready to
implement the unilateral-close clause.

## Implementing the unilateral-close clause

To recap, this clause must do the following things:

- Store a tuple of the form `{aliceAmount, bobAmount, counter, nonce}`;
- Accept two signatures and check that one is a signature over the
  tuple by Alice and the other is a signature over the tuple by Bob;
- Check that the nonce is equal to the anchor in the contract’s Value;
- Permit challenges to the stated balances for, let’s say, 24 hours;
- Permit disbursement only after that interval.

To handle the 24-hour interval, we’ll require the caller to supply a
timestamp in the near future (e.g., in the next five minutes). We’ll
use this as the “maxtime” in a `timerange` instruction, requiring the
transaction to appear in a block before then. We’ll also store the
timestamp on the contract’s stack so we can compute a “mintime” for a
`timerange` instruction in the disburse clause, prohibiting it from
running any earlier than that.

```
                     contract stack                                                        arg stack
                     --------------                                                        ---------
                     value alicePubkey bobPubkey                                           timestamp aliceSig bobSig tuple
get                  value alicePubkey bobPubkey tuple                                     timestamp aliceSig bobSig
dup encode           value alicePubkey bobPubkey tuple tupleStr                            timestamp aliceSig bobSig
dup                  value alicePubkey bobPubkey tuple tupleStr tupleStr                   timestamp aliceSig bobSig
3 roll               value alicePubkey tuple tupleStr tupleStr bobPubkey                   timestamp aliceSig bobSig
dup                  value alicePubkey tuple tupleStr tupleStr bobPubkey bobPubkey         timestamp aliceSig bobSig
4 bury               value alicePubkey bobPubkey tuple tupleStr tupleStr bobPubkey         timestamp aliceSig bobSig
get                  value alicePubkey bobPubkey tuple tupleStr tupleStr bobPubkey bobSig  timestamp aliceSig
0 checksig verify    value alicePubkey bobPubkey tuple tupleStr                            timestamp aliceSig
3 roll               value bobPubkey tuple tupleStr alicePubkey                            timestamp aliceSig
dup                  value bobPubkey tuple tupleStr alicePubkey alicePubkey                timestamp aliceSig
4 bury               value alicePubkey bobPubkey tuple tupleStr alicePubkey                timestamp aliceSig
get                  value alicePubkey bobPubkey tuple tupleStr alicePubkey aliceSig       timestamp
0 checksig verify    value alicePubkey bobPubkey tuple                                     timestamp
dup                  value alicePubkey bobPubkey tuple tuple                               timestamp
3 field              value alicePubkey bobPubkey tuple nonce                               timestamp
4 roll               alicePubkey bobPubkey tuple nonce value                               timestamp
anchor               alicePubkey bobPubkey tuple nonce value anchor                        timestamp
swap                 alicePubkey bobPubkey tuple nonce anchor value                        timestamp
5 bury               value alicePubkey bobPubkey tuple nonce anchor                        timestamp
eq verify            value alicePubkey bobPubkey tuple                                     timestamp
get dup              value alicePubkey bobPubkey tuple timestamp timestamp
0 swap               value alicePubkey bobPubkey tuple timestamp 0 timestamp
timerange            value alicePubkey bobPubkey tuple timestamp
[...next...] output
```

Here, `[...next...]` is the program for the next phase of the
contract’s lifecycle, in which either:

- The balances in `tuple` are challenged by providing a new tuple (and
  two valid signatures of it) containing the same nonce and a higher
  counter; or
- Twenty-four hours have elapsed and disbursement of the contract’s
  funds is triggered.

## Challenge

```
                        contract stack                                                                                                     arg stack
                        --------------                                                                                                     ---------
                        value alicePubkey bobPubkey tuple timestamp                                                                        aliceSig bobSig newTuple
swap                    value alicePubkey bobPubkey timestamp tuple                                                                        aliceSig bobSig newTuple
untuple                 value alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter nonce 4                                           aliceSig bobSig newTuple
drop drop               value alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter                                                   aliceSig bobSig newTuple
get dup                 value alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple newTuple                                 aliceSig bobSig
untuple                 value alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple aliceAmt bobAmt newCounter nonce         aliceSig bobSig
11 roll                 alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple aliceAmt bobAmt newCounter nonce value         aliceSig bobSig
anchor                  alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple aliceAmt bobAmt newCounter nonce value anchor  aliceSig bobSig
swap                    alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple aliceAmt bobAmt newCounter nonce anchor value  aliceSig bobSig
12 bury                 value alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple aliceAmt bobAmt newCounter nonce anchor  aliceSig bobSig
eq verify               value alicePubkey bobPubkey timestamp aliceAmt bobAmt oldCounter newTuple aliceAmt bobAmt newCounter               aliceSig bobSig
4 roll                  value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple aliceAmt bobAmt newCounter oldCounter               aliceSig bobSig
gt verify               value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple aliceAmt bobAmt                                     aliceSig bobSig
drop drop               value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple                                                     aliceSig bobSig
dup encode              value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr                                         aliceSig bobSig
dup                     value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr newTupleStr                             aliceSig bobSig
6 roll                  value alicePubkey timestamp aliceAmt bobAmt newTuple newTupleStr newTupleStr bobPubkey                             aliceSig bobSig
dup                     value alicePubkey timestamp aliceAmt bobAmt newTuple newTupleStr newTupleStr bobPubkey bobPubkey                   aliceSig bobSig
7 bury                  value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr newTupleStr bobPubkey                   aliceSig bobSig
get                     value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr newTupleStr bobPubkey bobSig            aliceSig
0 checksig verify       value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr                                         aliceSig
6 roll                  value bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr alicePubkey                                         aliceSig
dup                     value bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr alicePubkey alicePubkey                             aliceSig
7 bury                  value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr alicePubkey                             aliceSig
get                     value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple newTupleStr alicePubkey aliceSig
0 checksig verify       value alicePubkey bobPubkey timestamp aliceAmt bobAmt newTuple
3 bury                  value alicePubkey bobPubkey newTuple timestamp aliceAmt bobAmt
drop drop               value alicePubkey bobPubkey newTuple timestamp
contractprogram output
```

Notice that when this clause ends, the stack is in the same state as
when it began (but with an updated balances tuple), and the same
contract program is active. This means that the challenge clause can
be invoked any number of times on a channel where unilateral closure
has been initiated but not completed, provided the callers can produce
signed tuples with ever-increasing counters.

## Disbursement

All the information needed to disburse the channel’s funds is already
on the contract’s stack, so no arguments are required for this clause.
All this clause has to do is limit the mintime to `timestamp`+24
hours, then unpack the balances tuple and split the Value
appropriately between Alice and Bob, just as in
[Closing the channel](#closing-the-channel) above.

```
                            contract stack                                               arg stack
                            --------------                                               ---------
                            value alicePubkey bobPubkey newTuple timestamp
86400000 add                value alicePubkey bobPubkey newTuple timestamp+24hrs
0                           value alicePubkey bobPubkey newTuple timestamp+24hrs 0
timerange                   value alicePubkey bobPubkey newTuple
untuple                     value alicePubkey bobPubkey aliceAmt bobAmt counter nonce 4
drop drop drop              value alicePubkey bobPubkey aliceAmt bobAmt
4 roll                      alicePubkey bobPubkey aliceAmt bobAmt value
swap split                  alicePubkey bobPubkey aliceAmt remainingValue bobValue
put                         alicePubkey bobPubkey aliceAmt remainingValue                bobValue
2 roll                      alicePubkey aliceAmt remainingValue bobPubkey                bobValue
put                         alicePubkey aliceAmt remainingValue                          bobValue bobPubkey
[...send...] contract call  alicePubkey aliceAmt remainingValue
swap split                  alicePubkey zeroValue aliceValue
put                         alicePubkey zeroValue                                        aliceValue
drop                        alicePubkey                                                  aliceValue
put                                                                                      aliceValue alicePubkey
[...send...] contract call
```

## Putting it all together

The constructor for the channel contract has not changed. It’s still:

```
[get get get [...main body...] output] contract
```

This creates a contract that consumes three items from the argument
stack (a Value, Alice’s pubkey, and Bob’s pubkey), stores them on the
stack, sets the contract’s program to `[...main body...]` and outputs
the contract to the blockchain state.

When the contract is later reconstituted with `input`, it’s in order
to close the channel, either cooperatively or unilaterally, after some
number of off-chain transactions. So `[...main body...]` needs some
selector logic to choose the channel-closing mode. Let’s say the
caller supplies an argument of 0 to mean “cooperative” and anything
else to mean “unilateral.” That means `[...main body...]` is:

```
get
0 eq jumpif:$cooperative

...unilateral-close clause here...

$cooperative
...cooperative-close clause here...
```

Note: no `jump $end` is needed after the unilateral-close clause (to
prevent falling through into the cooperative-close clause) since that
clause ends in `output`, which terminates execution of the contract
program and returns control to the caller.

That `output` instruction changes the contract program to one we’ve
denoted as `[...next...]` above, so that the next time the contract is
re(-re)constituted with `input`, it’s in order to either:

- Challenge the balance tuple; or
- Disburse the funds.

So `[...next...]` also requires selector logic. We’ll choose an
argument of 0 to mean “disburse” and anything else to mean
“challenge.” That means `[...next...]` is:

```
get
0 eq jumpif:$disburse

...challenge clause here...

$disburse
...disburse clause here...
```

As before, no `jump:$end` is needed at the end of the challenge
clause, which also ends with `output`.
