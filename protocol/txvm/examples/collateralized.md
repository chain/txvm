# Collateralized loans

A lender will usually require collateral to secure the money he or she lends:
something the borrower will surrender to them if they fail to meet the loan’s repayment terms.
This can be modeled as a contract in TxVM.
The contract locks up the borrower’s collateral while simultaneously delivering the loan from the lender to the borrower.
When the loan is repaid the collateral is returned to the borrower,
but if a deadline passes first,
the lender may claim the collateral.

The example presented here is simplified for pedagogical purposes.
It does not include such loan features as installments,
compounding interest,
grace periods,
etc.,
but it should be clear that modeling such features is certainly possible in TxVM.

## Requirements

Creation of a new collateralized loan contract requires the Value being loaned,
the Value being used as collateral,
and a handful of parameters:

1. the repayment amount
   (or the amount due);
2. the repayment deadline;
3. the public key of the lender;
4. the public key of the borrower.

A transaction that creates a collateralized loan contract will require signatures from both the lender
(to authorize movement of the funds being loaned)
and the borrower
(to authorize movement of the collateral).
Before signing,
both participants must inspect the parameters being used in the contract to ensure they are the right ones.

When created,
the contract won’t store the loan Value.
Instead,
it will note that Value’s asset type and store that;
then immediately “send” the Value to the borrower.
(It notes the loan’s asset type in order to check that the future repayment has the same type.)

After the contract is created,
it holds the borrower’s collateral.
There are two clauses that will release it:

1. Loan repayment releases it to the borrower;
2. Loan default releases it to the lender.

Both clauses end with no items left on the stack,
so when either runs,
the contract evanesces.

## Constructor

In most of our other examples,
the constructor simply consumes a bunch of arguments,
stores them on the new contract’s stack,
and `output`s the contract to the blockchain.
In this example the constructor has to do a little more work:
it consumes the arguments,
it stores the loan Value’s asset type,
and it sends the loan Value to the borrower,
and _then_ it `output`s the contract to the blockchain.

```
                            contract stack                                        arg stack
                            --------------                                        ---------
                                                                                  loan borrower collateral due deadline lender
get get get get get get     lender deadline due collateral borrower loan
assetid                     lender deadline due collateral borrower loan assetid
2 bury                      lender deadline due collateral assetid borrower loan
put                         lender deadline due collateral assetid borrower       loan
dup put                     lender deadline due collateral assetid borrower       loan borrower
[...send...] contract call  lender deadline due collateral assetid borrower
[...main...] output
```

Here,
`[...send...]` is our standard pay-to-pubkey contract,
and `[...main...]` is the main body of this contract,
embodying the two collateral-release clauses described above.

## Repayment

The repayment clause of the contract’s main body takes a Value as an argument.
It checks that its amount equals `due` and its asset type equals `assetid`,
then sends the repayment to the lender and the collateral to the borrower.

```
                            contract stack                                                      arg stack
                            --------------                                                      ---------
                            lender deadline due collateral assetid borrower                     payment
get                         lender deadline due collateral assetid borrower payment
amount                      lender deadline due collateral assetid borrower payment paymentAmt
5 roll                      lender deadline collateral assetid borrower payment paymentAmt due
eq verify                   lender deadline collateral assetid borrower payment
assetid                     lender deadline collateral assetid borrower payment paymentAsset
3 roll                      lender deadline collateral borrower payment paymentAsset assetid
eq verify                   lender deadline collateral borrower payment
put                         lender deadline collateral borrower                                 payment
3 roll                      deadline collateral borrower lender                                 payment
put                         deadline collateral borrower                                        payment lender
[...send...] contract call  deadline collateral borrower
swap put put                deadline                                                            collateral borrower
[...send...] contract call  deadline
drop
```

## Default

If the deadline passes with no repayment,
the loan is defaulted and the lender may claim the collateral in the contract.

```
                            contract stack                                   arg stack
                            --------------                                   ---------
                            lender deadline due collateral assetid borrower
drop drop                   lender deadline due collateral
2 roll 0                    lender due collateral deadline 0
timerange                   lender due collateral
put                         lender due                                       collateral
drop                        lender                                           collateral
put                                                                          collateral lender
[...send...] contract call
```

## Putting it all together

The main body of the loan contract must choose one of the clauses to run.
Let’s make it expect a selector on the argument stack that dispatches to the right clause.

```
get                         # consume the selector arg
1 eq jumpif:$repay          # if it’s 1, run the repayment clause
                            # otherwise, fall through to the default clause
$default
drop drop
2 roll 0
timerange
put
drop
put
[...send...] contract call
jump:$end

$repay
get
amount
5 roll
eq verify
assetid
3 roll
eq verify
put
3 roll
put
[...send...] contract call
swap put put
[...send...] contract call
drop

$end
```
