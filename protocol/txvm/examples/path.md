# Path payments

This is an elaboration of [the Orderbook example](orderbook.md).  As
in that example, for simplicity we will consider only the case where
quantities offered for sale exactly equal quantities sought to
buy. It’s not too difficult to elaborate these examples to permit
partial fulfillment of orderbook orders.

In this example we address the case where one seller offers X
units of asset A in exchange for Y units of asset B, and another
offers Y units of asset B in exchange for Z units of asset
C. A “path payment” solution allows a buyer to buy asset A and pay
with asset C, routing their payment through the intermediary who can
convert the buyer’s payment into the seller’s asking price.

In general there may be a sequence of such intermediaries. This
example generalizes readily to that case.

## Requirements

The offers described in this example are contracts in the blockchain’s
UTXO set. Specifically they are exactly the contracts described in
[the Orderbook example](orderbook.md).

As further described in the discussion for the Orderbook example, an
app maintains a database of open orderbook offers. It is this app’s
responsibility to find the path matching the A-for-B seller with the
A-for-C buyer via the B-for-C intermediary.

To conclude this sequence of payments it is not necessary to develop
any new contracts, only to construct a transaction that `input`s and
satisfies the already-existing contracts involved.

## The transaction

Here is how a path payment transaction may be constructed.

```
                              contract stack    arg stack
                              --------------    ---------
{'C', ...ZofC...} input call                    [sigcheck ZofCvalue]
1 put                                           [sigcheck ZofCvalue 1]
{'C', ...YofB...} input call                    [sigcheck YofBvalue]
1 put                                           [sigcheck YofBvalue 1]
{'C', ...XofA...} input call                    [sigcheck XofAvalue]
<buyerpubkey> put                               [sigcheck XofAvalue buyerpubkey]
[...send...] contract call                      [sigcheck]
finalize                                        [sigcheck]
get                           [sigcheck]        []
<signature> put               [sigcheck]        [<sig>]
call                          []                []
```

First the transaction `input`s and calls a contract containing Z units
of asset C. Presumably this is an ordinary UTXO that produces a
signature-checking contract and a Value object on the argument stack
when called.  (In practice this may involve `input`ting and calling
multiple UTXO contracts until the threshold Z is reached, and
`output`ting any “change” back to the buyer.)

The signature-checking contract remains on the argument stack until
after the `finalize` instruction, as discussed in
[the Account example](account.md).

The selector 1 is placed on the argument stack to signal to the next
contract that we want to invoke its “buy” clause (see
[the Orderbook example](orderbook.md)).

Next, the orderbook contract offering Y units of B in exchange for Z
units of C is `input`ted and called. It consumes the selector 1 from
the argument stack, invoking its “buy” clause, which further consumes
the Value object containing Z units of asset C. It releases its Y
units of asset B to the argument stack.

Another 1 is placed on the argument stack, the selector for the next
contract.

The orderbook contract offering X units of A in exchange for Y units
of B is `input`ted and called. It consumes the selector 1, invoking
its “buy” clause, further consuming the Value object containing Y
units of asset B. It releases its X units of asset A to the argument
stack.

The transaction “sends” this Value to the buyer (as in
[the Orderbook example](orderbook.md)).

Now the transaction is `finalize`d. The signature-check contract is
moved from the argument stack and a signature placed there for it to
consume. Finally the signature-check contract is called and the
transaction is complete.
