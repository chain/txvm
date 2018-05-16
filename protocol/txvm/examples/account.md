# Account

Broadly speaking there are two main approaches to recording state in a
blockchain.

The Chain protocol uses the “utxo” model, where the main items in the
blockchain’s persistent state are chunks of value, protected by
programs called contracts. A transaction that can satisfy a contract
can consume the value, usually transforming it into one or more new
chunks of value, each protected by some new contract. In the typical
case, the contract simply tests a digital signature.

The utxo model is so-called because each item in the state is the
result, or the _output_, of some transaction, and hasn’t yet been
spent. Utxo stands for “unspent transaction output.”

Some other blockchains organize their state around top-level account
objects. Each account holds a balance of some value and is protected
by a public/private keypair. Supplying a suitable signature allows
funds from the account to be used in contracts.

Accounts are a higher-level abstraction than utxos, meaning it should
be possible to model accounts with utxos. Here’s how that looks in
TxVM.

## Requirements

This is a very rudimentary account design. We need to be able to do
four things:

1. Create an account with an initial balance and the public key that
   secures it.
2. Deposit to the account.
3. Withdraw from the account.
4. Query the balance of the account.

## Create

The code to create an account will consume an initial balance (as a
Value object) and a public key from the argument stack, then hold them
in the contract stack as it `output`s itself to the blockchain state
to await the next operation. In TxVM assembly language that’s simply:

```
get get [main] output
```

Here, [main] is a placeholder for the code of the account contract’s
main body. We’ll develop that in the following sections.

First let’s choose a convention for the two items stored on the
account contract’s stack: let’s say that the value containing the
account’s balance is on top of the stack, and the pubkey is below it.

## Query the balance

A Value in TxVM has an amount and an asset type. For the query clause,
let’s return both to the caller by querying them from the Value on top
of the contract stack and `put`ting them onto the argument stack.

```
assetid put
amount put

```

## Deposit new funds

For this clause, the caller will supply a Value object on the argument
stack. We’ll consume it with `get` and merge it into the Value on top
of the contract stack.

This clause will fail if the two Values don’t have the same asset
type.

```
get merge
```

## Withdraw funds

For this clause, the caller will supply an integer on the argument
stack: the amount to withdraw. The account contract will return two
items: a Value with the requested amount; and a signature-check
contract.

The caller is obliged by the rules of TxVM to invoke the
signature-check contract with suitable arguments in order to “clear”
it from the stack. This ensures that no one but the authorized account
holder may make withdrawals.

Consuming the amount argument and returning the desired value is
simple. It’s just:

```
get split put
```

This leaves the account pubkey and a Value with the remaining balance
on the contract stack while putting the requested Value on the
argument stack. The `split` instruction will fail if the requested
amount is greater than the amount in the Value on the stack.

The interesting part is the signature-check contract. It should
require a signature, then check (with the `checksig` instruction) that
it matches the account’s pubkey and some message being signed.

What message should we require the caller to sign?

It should be chosen to make _replay_ and _race_ attacks impossible.

A replay attack is one where the same signature can be used in
multiple transactions. An eavesdropper can obtain the signature and then
use it to make unauthorized further withdrawals.

A race attack is one where the attacker makes a copy of the authentic
transaction, alters it to benefit him or herself, and manages (by luck
or other means) to get it submitted and published before the authentic
transaction.

A good way to prevent both attacks is to require that the signed
message be the transaction itself, or rather a hash of it. Every
transaction has a distinct hash, so the signature cannot be used for
any other transaction (no replays). And any alteration to the
transaction changes its hash, invalidating the signature (no races).

You might see a chicken-and-egg problem here: it’s necessary to get
the transaction’s hash in order to produce a signature; but adding the
signature to the transaction will change its hash, invalidating the
signature!

This paradox is the reason that TxVM includes the `finalize`
instruction. TxVM guarantees that no instruction added after
`finalize` will change its hash. (It also prohibits instructions that
would alter the outcome of the transaction.) So when constructing a
“withdraw” transaction, it’s possible to assemble it as far as the
`finalize` instruction; then get the hash; then produce a signature;
then add that to complete the transaction.

That also explains why the withdraw clause _returns_ a
signature-checking contract to invoke later, rather than _requiring_ a
signature and checking it itself. Invoking the signature-checking
contract must be deferred until after a `finalize` instruction runs,
so that the `txid` instruction it contains can produce the
transaction’s hash.

Here’s the full withdraw clause, with some discussion below.

```
                                                       #                stack begins as: [pubkey balance]
get                                                    # consume amount argument         [pubkey balance amount]
split                                                  # split balance into two Values   [pubkey newbalance requestedval]
put                                                    # move withdrawal to the argstack [pubkey newbalance]
swap dup                                               # make a copy of pubkey           [newbalance pubkey pubkey]
put                                                    # move it to the argstack         [newbalance pubkey]
swap                                                   # restore stack ordering          [pubkey newbalance]
[get [txid swap get 0 checksig verify] yield] contract # create a new contract           [pubkey newbalance sigchecker1]
call                                                   # call it (see below)             [pubkey newbalance]
```

The final steps of this clause deserve some explanation. `[...]
contract` means “assemble these instructions into bytecode and turn
them into a new contract.” The new contract goes onto the stack (as
“sigchecker1” in the depiction above). The next instruction, `call`,
takes it off the stack and runs it.

That nested contract does two things when it runs:

`get`

and

`[txid swap get 0 checksig verify] yield`

The `get` instruction moves the account’s pubkey from the argstack to
the contract stack. (Recall that the withdraw clause made a copy of
the pubkey and put it on the argstack.)

The `[...] yield` means “return to the caller, but leave this contract
on the argstack (together with the items on its contract stack), and
the next time it’s called, run these instructions.” So some time
later, after the transaction’s `finalize` instruction executes, the
caller can get the transaction’s hash, produce a signature, then add
further instructions to the transaction that put the signature on the
argument stack and invoke this remaining contract one more
time. Here’s what happens during that final call:

```
         #                  stack begins as: [pubkey]
txid     # add transaction hash to the stack [pubkey hash]
swap     # reorder                           [hash pubkey]
get      # move signature from argstack      [hash pubkey sig]
0        # select “ed25519” signature scheme [hash pubkey sig 0]
checksig # compute valid-signature boolean   [bool]
verify   # fail if bool is not true          []
```

It’s important to note that, even though the withdraw clause returns
the withdrawn Value before checking for an authorizing signature,
there is no way for that Value to escape without the signature being
present. The rules of TxVM require that all stacks are empty at the
end of a transaction, and the only way to get a contract off the stack
is to run it until _its_ stack is empty. So the signature-check
contract can’t be skipped, and if the transaction cannot complete, it
is excluded from the blockchain and nothing it does ever actually
happens.

## Putting it all together

The account contract must embody all these clauses. It will expect a
selector on the top of the argument stack, telling it which clause to
execute (and also how to interpret any other values on the argument
stack). After the selected clause executes, it will `output` itself
again to await the next transaction. Note that the contract stack is
always `[pubkey value]` at the beginning and the end of each clause.

```
get                      # consume the selector
dup                      # make a copy of the selector (eq consumes it)
1 eq jumpif:$deposit     # if it’s 1, execute the deposit clause
2 eq jumpif:$withdraw    # if it’s 2, execute the withdraw clause
                         # otherwise, execute the query clause
assetid put
amount put
jump:$finish

$deposit
drop                     # didn’t need that extra copy of the selector after all
get merge
jump:$finish

$withdraw
get split put
swap dup put
swap
[get [txid swap get 0 checksig verify] yield] contract
call
                         # fall through to the finish clause
$finish
contractprogram          # get a copy of the currently executing contract
output                   # output to blockchain state
```
