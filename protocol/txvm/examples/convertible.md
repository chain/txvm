# Convertible assets (or, no wei!)

The native asset type on the Ethereum blockchain is the ether,
whose smallest subunit is the wei.
There are a _quintillion_ wei in one ether.

Modeling transactions denominated in ether presents a problem for TxVM,
where amounts are limited to numbers that can be expressed in 63 bits.
63 bits means TxVM cannot count higher than around 9.2 quintillion.
So if the TxVM asset type representing ether counts in weis,
no transaction can represent more than about 9.2 ether,
which isn’t terribly much.
If the TxVM asset type representing ether counts in some larger unit
(like Gwei,
which is a billion wei),
then it’s possible to represent very large amounts of ether but without perfect precision.
Fractional amounts of Gwei will be lost.

Luckily,
there’s another option.
TxVM allows us to define a pair of related asset types,
one a larger denomination of the other,
like Gwei and wei
(or dollars and cents).
A transaction can use a pair of Values,
one denominated in Gwei for the bulk of the total amount,
and one denominated in wei for the “change,” to represent large quantities of ether with perfect precision.

For this model to be accurate,
it must be possible to convert units of one asset type into units of its “partner” asset type freely at a fixed ratio:
1 billion wei for a Gwei
(or 100 cents for a dollar).
Here’s how to do that in TxVM.

(Since the numbers involved are simpler,
the rest of this document will use dollars and cents in examples.
But the same design works with Gwei and wei.)

## Issuance and asset types

An asset type in TxVM is uniquely determined by the contract that issues it,
and the data that that contract supplies to the `issue` instruction.

A vanilla TxVM asset uses a set of public keys and a caller-supplied “tag” as arguments to the standard asset-issuance contract.
Issuance requires supplying a signature matching
(some quorum of)
the public keys.

## Requirements

Our pair of “convertible asset types” must permit ordinary issuance via signature authorization,
but must also permit conversion of some amount of one asset to the corresponding amount of the other asset.
This is also issuance:
when converting cents to dollars,
a new dollar is issued for every 100 cents supplied.

In fact the 100 cents must not only be supplied,
they must be _destroyed_ when issuing the new dollar,
in order to keep the total amount in circulation constant.
(Otherwise we’re in the realm of the other kind of issuance,
requiring signatures to increase the supply of the asset.)
Destroying value is done with the `retire` instruction.

So we need a new issuance contract that can operate in one of two modes:
normal issuance,
in which signatures permit the creation of arbitrarily many new units;
and conversion,
in which some amount of the partner asset is destroyed while dictating how many new units of this asset to create.

We’ll treat the case of normal issuance as solved.
You can see how Chain does issuance in
[the standard asset-issuance contract](https://github.com/chain/txvm/blob/main/protocol/txbuilder/standard/issue.go)
in the chain/txvm GitHub repo.

For conversion,
we’ll need a way for the issuance contract to test whether the supplied Value
(to be destroyed)
belongs to the partner asset type.
It will also have to determine the correct amount to issue based on the amount destroyed.

## The asset tag

The `issue` instruction consumes an arbitrary string called the “asset tag.” That’s combined with the identity of the current contract
(the contract containing the `issue` instruction)
to get the ID of the asset to issue.

The standard Chain issuance contract builds the asset tag from the set of public keys and an arbitrary caller-supplied string.
Let’s call that string the _pre-tag_.
(Some of the documentation confusingly refers to that as a tag too,
even though it’s different from the asset tag consumed by `issue`.
For this document we’ll stick with “pre-tag.”)

By choosing the right format for our pre-tag we can kill two birds with one stone:
allowing the issuance contract to identify values of the “partner” asset type;
and also specifying the conversion ratio between the two asset types.
To do this,
let’s stipulate that the pre-tag must be the _encoding_
(conversion to string format)
of a TxVM _tuple_ with this format:

```
{...arbitrary items..., M, N}
```

where “arbitrary items” can be any TxVM data needed to distinguish different asset types from one another
(a “pre-pre-tag,” if you will),
and M and N are integers expressing the conversion ratio.
For two asset types to be “partners,” they must have the same “arbitrary items” prefix,
and one must end with `{...,M,N}` while the other ends with the inverse,
`{...,N,M}`.

Let’s call this the “pre-tag tuple.” The `encode` instruction can turn that tuple into the “pre-tag”
(a string).

So,
for instance,
dollars and cents could be defined with the pre-tag tuples `{'USD',100,1}` and `{'USD',1,100}`.

When performing a conversion,
the issuance contract can take the pre-tag tuple it gets as an argument,
then swap its M and N fields to produce the partner type’s pre-tag tuple.
From that it can compute the expected asset ID,
then compare that with the actual asset ID of the Value supplied for conversion.

The issuance contract can also use M and N to compute the amount to issue based on the amount being retired.

## Pre-tag tuple subroutines

Let’s start with the code that computes the partner’s asset ID,
given the pre-tag tuple,
the quorum threshold,
and the tuple full of pubkeys on the argument stack.
(Remember that the quorum and the set of pubkeys are part of the asset’s identity too,
in addition to the pre-tag tuple.)

The specification for how an asset ID is computed can be found
[here](https://github.com/chain/txvm/blob/main/specifications/txvm.md#asset-id).

```
           contract stack                           arg stack
           --------------                           ---------
                                                    {pubkey...} quorum pretag-tuple
self       seed                                     {pubkey...} quorum               # put this contract’s “seed” (a hash) on the stack
get        seed pretag-tuple                        {pubkey...} quorum               # move pretag-tuple from arg stack to contract stack
untuple    seed ...arbitrary-items... M N len       {pubkey...} quorum               # unpack tuple into its member items plus its length
2 bury     seed ...arbitrary-items... len M N       {pubkey...} quorum               # move len out of the way for a moment
swap       seed ...arbitrary-items... len N M       {pubkey...} quorum               # swap M and N
2 roll     seed ...arbitrary-items... N M len       {pubkey...} quorum               # get len back
tuple      seed partner-tuple                       {pubkey...} quorum               # turn len items back into a tuple
get get    seed partner-tuple quorum {pubkey...}                                     # get remaining args
3 tuple    seed {partner-tuple,quorum,{pubkey...}}                                   # combine into a single tuple
encode     seed asset-tag                                                            # turn that tuple into a string, the “asset tag”
cat        (seed+asset-tag)                                                          # concatenate two strings
'AssetID'  (seed+asset-tag) 'AssetID'                                                # this “domain separator” is part of the asset ID calculation
vmhash     assetID                                                                   # compute asset ID
```

Now let’s take a look at the way to calculate the amount to issue,
given a Value to retire.
We’ll have to multiply that Value’s amount by M,
ensure it leaves no remainder when divided by N,
and then divide.
Let’s assume that the pre-tag tuple and the retire Value are on the argument stack.

```
             contract stack                             arg stack
             --------------                             ---------
                                                        retireval pretag-tuple
get          pretag-tuple                               retireval               # move the pre-tag tuple to the contract stack
dup dup len  pretag-tuple pretag-tuple len              retireval               # copy the tuple twice; get its length (consuming one copy)
1 sub        pretag-tuple pretag-tuple len-1            retireval               # subtract 1
dup 2 bury   pretag-tuple len-1 pretag-tuple len-1      retireval               # make a copy, move it out of the way
field        pretag-tuple len-1 N                       retireval               # get the len-1’th item from tuple, which is N
2 bury       N pretag-tuple len-1                       retireval               # move it out of the way
1 sub        N pretag-tuple len-2                       retireval               # subtract 1
field        N M                                        retireval               # get the len-2’th item from tuple, which is M
get          N M retireval                                                      # get the Value to retire
amount       N M retireval retireamt                                            # get its amount
2 roll       N retireval retireamt M                                            # move M into place
mul          N retireval retireamt*M                                            # multiply
dup          N retireval retireamt*M retireamt*M                                # make a copy
3 roll       retireval retireamt*M retireamt*M N                                # move N into place
dup 2 bury   retireval retireamt*M N retireamt*M N                              # make a copy, move it out of the way
mod          retireval retireamt*M N (retireamt*M % N)                          # compute amount*M mod N
0 eq verify  retireval retireamt*M N                                            # compare it to zero, fail if not equal
div          retireval (retireamt*M / N)                                        # compute amount*M / N
```

The final version of our issuance contract will differ slightly from the code shown here,
since data items may not be where these examples assumed they would be.
(For instance,
both subroutines shown here get the pre-tag tuple from the argument stack,
but in the final version that will only happen once,
with the value shared between the subroutines.)

## Putting it together

The complete issuance contract begins by pulling a selector off the argument stack,
telling whether the caller wants a signature-based issuance or a conversion.

```
get                     # move the selector from the arg stack to the contract stack
1 eq jumpif:$convert    # if it’s equal to 1, jump to the convert clause
```

If the selector is _not_ 1,
it executes the code that follows.
Here’s where we would insert the standard asset-issuance contract
(which,
again,
can be found
[here](https://github.com/chain/txvm/blob/main/specifications/txvm.md#asset-id)).
It needs to be followed by:

```
jump:$end
```

in order to skip over the convert clause,
which follows.
The convert clause expects the stack to contain pubkeys,
quorum,
pre-tag tuple,
and the Value to retire,
in that order
(with retireval on top).

```
            contract stack                                                                                         arg stack
            --------------                                                                                         ---------
                                                                                                                   {pubkey,...} quorum pretag-tuple retireval
$convert                                                                                                           {pubkey,...} quorum pretag-tuple retireval
get         retireval                                                                                              {pubkey,...} quorum pretag-tuple
assetid     retireval actualAssetID                                                                                {pubkey,...} quorum pretag-tuple
self        retireval actualAssetID seed                                                                           {pubkey,...} quorum pretag-tuple
get         retireval actualAssetID seed pretag-tuple                                                              {pubkey,...} quorum
dup 4 bury  pretag-tuple retireval actualAssetID seed pretag-tuple                                                 {pubkey,...} quorum
untuple     pretag-tuple retireval actualAssetID seed ... M N len                                                  {pubkey,...} quorum
swap dup    pretag-tuple retireval actualAssetID seed ... M len N N                                                {pubkey,...} quorum
2 roll dup  pretag-tuple retireval actualAssetID seed ... M N N len len                                            {pubkey,...} quorum
3 bury      pretag-tuple retireval actualAssetID seed ... M len N N len                                            {pubkey,...} quorum
4 add bury  pretag-tuple N retireval actualAssetID seed ... M len N                                                {pubkey,...} quorum
2 roll dup  pretag-tuple N retireval actualAssetID seed ... len N M M                                              {pubkey,...} quorum
3 roll dup  pretag-tuple N retireval actualAssetID seed ... N M M len len                                          {pubkey,...} quorum
2 bury      pretag-tuple N retireval actualAssetID seed ... N M len M len                                          {pubkey,...} quorum
5 add bury  pretag-tuple M N retireval actualAssetID seed ... N M len                                              {pubkey,...} quorum
tuple       pretag-tuple M N retireval actualAssetID seed partner-tuple                                            {pubkey,...} quorum
get         pretag-tuple M N retireval actualAssetID seed partner-tuple quorum                                     {pubkey,...}
dup 7 bury  pretag-tuple quorum M N retireval actualAssetID seed partner-tuple quorum                              {pubkey,...}
get         pretag-tuple quorum M N retireval actualAssetID seed partner-tuple quorum {pubkey,...}
dup 7 bury  pretag-tuple quorum {pubkey,...} M N retireval actualAssetID seed partner-tuple quorum {pubkey,...}
3 tuple     pretag-tuple quorum {pubkey,...} M N retireval actualAssetID seed {partner-tuple,quorum,{pubkey,...}}
encode      pretag-tuple quorum {pubkey,...} M N retireval actualAssetID seed partner-tag
cat         pretag-tuple quorum {pubkey,...} M N retireval actualAssetID (seed+partner-tag)
'AssetID'   pretag-tuple quorum {pubkey,...} M N retireval actualAssetID (seed+partner-tag) 'AssetID'
vmhash      pretag-tuple quorum {pubkey,...} M N retireval actualAssetID expectedAssetID
eq verify   pretag-tuple quorum {pubkey,...} M N retireval
```

That’s the end of testing that the retireval has the right asset type.
The convert clause continues with the arithmetic that computes how much to issue.

```
             contract stack                                                              arg stack
             --------------                                                              ---------
             pretag-tuple quorum {pubkey,...} M N retireval
amount       pretag-tuple quorum {pubkey,...} M N retireval retireamt
3 roll mul   pretag-tuple quorum {pubkey,...} N retireval retireamt*M
dup          pretag-tuple quorum {pubkey,...} N retireval retireamt*M retireamt*M
3 roll dup   pretag-tuple quorum {pubkey,...} retireval retireamt*M retireamt*M N N
2 bury       pretag-tuple quorum {pubkey,...} retireval retireamt*M N retireamt*M N
mod          pretag-tuple quorum {pubkey,...} retireval retireamt*M N (retireamt*M % N)
0 eq verify  pretag-tuple quorum {pubkey,...} retireval retireamt*M N
div          pretag-tuple quorum {pubkey,...} retireval issueamt
swap         pretag-tuple quorum {pubkey,...} issueamt retireval
splitzero    pretag-tuple quorum {pubkey,...} issueamt retireval zeroval
2 bury       pretag-tuple quorum {pubkey,...} zeroval issueamt retireval
retire       pretag-tuple quorum {pubkey,...} zeroval issueamt
4 roll       quorum {pubkey,...} zeroval issueamt pretag-tuple
4 roll       {pubkey,...} zeroval issueamt pretag-tuple quorum
4 roll       zeroval issueamt pretag-tuple quorum {pubkey,...}
3 tuple      zeroval issueamt {pretag-tuple,quorum,{pubkey,...}}
encode       zeroval issueamt asset-tag
issue        issuedval
put                                                                                      issuedval
```

Finally,
the issuance contract needs a target for the earlier `jump:$end`.

```
$end
```
