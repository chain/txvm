# Escrow payments

In an escrow payment,
the payer sends his or her payment to a third party.
The third party is trusted by both the payer and the payee to release the payment to the payee when some agreed-on condition is met,
or to return it to the payer if the condition is not met by some deadline.

## Requirements

Our escrow contract takes five inputs:

1. The payment to secure;
2. The public key of the payee;
3. The public key of the payer,
   for returning the payment if necessary;
4. The public key of the trusted third party
   (or “escrow agent”);
5. The deadline after which payment can be reclaimed by the payer
   (if it hasn’t already been released to the payee).

It has three ways to release the payment it holds:

1. The escrow agent releases payment to the payee;
2. The escrow agent returns payment to the payer;
3. The payer reclaims payment after the deadline.

Each of these clauses ends with no items left on the contract stack,
meaning this contract disappears after one use.

Note that there is no way for the escrow agent to abscond with the funds.
They can only end up with the payer or the payee.

## Create

Creating the contract simply consumes the five required arguments,
stores them on the stack,
and outputs the contract to the blockchain.

```
get get get get get [...main...] output
```

Here,
`[...main...]` is a placeholder for the three clauses of the escrow contract’s main body.
Let’s develop the clauses one by one.

Let’s say that the order of arguments is the same as described above under Requirements
(payment,
payee,
payer,
agent,
deadline),
which means that the arguments will be stored on the contract stack with payment on the bottom and deadline on the top.

## Pay payee

The pay-payee clause requires a signature from the escrow agent.
The escrowed Value is moved into a new contract secured by the payee’s public key.

```
                            contract stack                      arg stack
                            --------------                      ---------
                            payment payee payer agent deadline
drop                        payment payee payer agent
put                         payment payee payer                 agent
[...checksig...] contract   payment payee payer sigChecker1     agent
call                        payment payee payer                 sigChecker2
drop                        payment payee                       sigChecker2
swap                        payee payment                       sigChecker2
put put                                                         sigChecker2 payment payee
[...send...] contract call                                      sigChecker2
```

Here,
`[...checksig...]` is the same signature-checking contract we’ve used in our other examples.
It consumes the agent’s pubkey from the argument stack,
then does a `yield`.
When later re-invoked
(after a `finalize`)
it will consume the agent’s signature from the argument stack and compare it against the agent’s pubkey and the transaction hash.

Also,
`[...send...]` is the same pay-to-pubkey contract we’ve used in our other examples.
It consumes the recipient’s pubkey and a Value object from the argument stack,
then outputs itself to the blockchain.
The Value can be unlocked in a later transaction with the recipient’s signature.

## Pay payer

The pay-payer clause is almost identical to pay-payee.
The escrow agent must supply a signature and the escrowed Value in the contract is moved into a new contract.
The only difference is that the new contract uses the payer’s pubkey instead of the payee’s.

```
                            contract stack                      arg stack
                            --------------                      ---------
                            payment payee payer agent deadline
drop                        payment payee payer agent
put                         payment payee payer                 agent
[...checksig...] contract   payment payee payer sigChecker1     agent
call                        payment payee payer                 sigChecker2
swap drop                   payment payer                       sigChecker2
swap                        payer payment                       sigChecker2
put put                                                         sigChecker2 payment payer
[...send...] contract call                                      sigChecker2
```

## Reclaim payment

Reclaim payment is like pay-payer,
except that instead of checking for the escrow agent’s signature,
the `deadline` on the contract stack is used as the “mintime” parameter to a `timerange` instruction.
This ensures the transaction is valid only in blocks whose timestamp is later than `deadline`.
We use 0 for the maxtime to mean “no maximum time.”

```
                            contract stack                        arg stack
                            --------------                        ---------
                            payment payee payer agent deadline
0                           payment payee payer agent deadline 0
timerange                   payment payee payer agent
drop                        payment payee payer
swap drop swap              payer payment
put put                                                           payment payer
[...send...] contract call
```

## Putting it all together

The escrow contract must contain all three clauses,
and we’ll require a selector argument on top of the stack that tells it which clause to execute.

```
get                        # consume the selector
dup                        # make a copy of the selector (eq below consumes it)
1 eq jumpif:$paypayee      # if it’s 1, execute the pay-payee clause
2 eq jumpif:$paypayer      # if it’s 2, execute the pay-payer clause
                           # otherwise, execute the reclaim clause

$reclaim
0
timerange
drop
swap drop swap
put put
[...send...] contract call
jump:$end

$paypayee
drop                       # discard the extra copy of the selector
drop
put
[...checksig...] contract
call
drop
swap
put put
[...send...] contract call
jump:$end

$paypayer
drop
put
[...checksig...] contract
call
swap drop
swap
put put
[...send...] contract call

$end
```
