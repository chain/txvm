# Orderbook entry

An entry in an orderbook is an offer to sell some amount of some asset at a particular price.

In TxVM a seller can lock the offered amount in a contract that also records the asking price and the seller’s payment address
(public key).
The contract’s value can be released to anyone who runs the contract and supplies it with the seller’s requested payment
(which the contract sends to the seller’s address simultaneous to releasing its value to the buyer).

## Requirements

An offer contract must contain the Value for sale
(a specific amount of some asset type),
plus the amount and asset type of the price being asked.
It must also specify
(as a public key)
who receives the sale price.

For simplicity,
we’ll require buyers to consume the entire contract and pay the full asking price.
A version of this contract that allows the Value to be partially consumed for a proportional price
(with the balance paid into a new orderbook-offer contract)
is left as an exercise for the reader.

In addition to a “buy” clause that anyone can invoke by supplying the asking price,
the contract should also have a “cancel” clause that allows the seller
(and only the seller)
to recover the Value.

## Constructor

The constructor looks like this:

```
[get get get get [...main body...] output] contract
```

This produces a contract that consumes four arguments from the argument stack:
the Value,
the seller’s public key,
and the amount and asset-type of the asking price.
Let’s decide
(arbitrarily)
to keep the arguments in the contract in that order:
with the Value on the bottom and the asset-type of the asking price on top.

This constructor then `output`s the contract to the blockchain state with “...main body...” as the contract’s program.
The main body must contain the “buy” and “cancel” clauses.
Any future transaction may consume this contract using `input` and invoking one of those two clauses via `call`.

## The buy clause

The “buy” clause expects a Value on the argument stack
(payment for the Value in the contract).
It checks that the payment has the right amount and asset type,
and then it releases the contract’s Value to the argument stack and “sends” the payment to the seller.

(“Sending” here means creating a new contract to contain the Value until it can be unlocked with a signature matching the recipient’s public key.)

```
                            contract stack                                         arg stack
                            --------------                                         ---------
                            value seller priceamt pricetype                        payment
get                         value seller priceamt pricetype payment
assetid                     value seller priceamt pricetype payment paymenttype
2 roll                      value seller priceamt payment pricetype paymenttype
eq verify                   value seller priceamt payment
amount                      value seller priceamt payment paymentamt
2 roll                      value seller payment priceamt paymentamt
eq verify                   value seller payment
put put                     value                                                  payment seller
[...send...] contract call  value
put                                                                                value
```

Here,
`[...send...]` is a constructor that consumes a recipient pubkey and a Value from the argument stack,
then `output`s itself to the blockchain state with a program that releases the Value when a valid signature of that transaction’s ID is supplied.
For example:

```
[get get [put [txid swap get 0 checksig verify] yield] output]
```

(See
[Account](account.md)
for a discussion of the signature-checking clause used here.)

## The cancel clause

The “cancel” clause returns the Value to the seller.
It discards the price data from the contract and uses the same “send” contract as in the “buy” clause.
It also schedules a signature check to ensure that no one but the seller can cancel the offer.

```
                                         contract stack                   arg stack
                                         --------------                   ---------
                                         value seller priceamt pricetype
drop drop                                value seller
swap put put                                                              value seller
[...send...] contract call
[txid swap get 0 checksig verify] yield
```

## Selector

The main body of the contract is not complete until it can understand which clause the caller wishes to invoke.
The caller supplies a “selector” argument and the contract dispatches to the right clause based on that.
The dispatch logic together with the clauses constitutes the “...main body...” mentioned above.

For the dispatcher we
(arbitrarily)
choose 0 to mean “cancel” and any other value to mean “buy.”

```
                               contract stack                            arg stack
                               --------------                            ---------
                               value seller priceamt pricetype           ...otherargs... selector
get                            value seller priceamt pricetype selector  ...otherargs...
0 eq jumpif:$cancel            value seller priceamt pricetype           ...otherargs...

$buy
...buy clause goes here...
jump:$end

$cancel
...cancel clause goes here...

$end
```

## Discussion: An orderbook app

An orderbook app would monitor the blockchain looking for orderbook activity.
The contract we’ve developed here has a distinct “seed”
(a hash of its initial program),
as do all TxVM contracts.
As each new block arrives,
the app inspects the log of each transaction in the block.
The creation of new orderbook offers will show up in the log as new `O` records with the seed.
The fulfillment or cancellation of orderbook offers will show up as `I` records with the seed.

The seed determines how to interpret the contents of the contract’s stack,
and those contents can be inspected directly
(e.g.
via callback hooks)
as the VM executes the transaction.
In this way an orderbook app can maintain an up-to-date database of open orders.
