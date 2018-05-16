# Convertible assets (or, no wei!)

The native asset type on the Ethereum blockchain is the ether, whose
smallest subunit is the wei. There are a _quintillion_ wei in one
ether.

Modeling transactions denominated in ether presents a problem for
TxVM, where amounts are limited to numbers that can be expressed in 63
bits. 63 bits means TxVM cannot count higher than around 9.2
quintillion. So if the TxVM asset type representing ether counts in
weis, no transaction can represent more than about 9.2 ether, which
isn’t terribly much.  If the TxVM asset type representing ether counts
in some larger unit (like Gwei, which is a billion wei), then it’s
possible to represent very large amounts of ether but without perfect
precision. Fractional amounts of Gwei will be lost.

Luckily, there’s another option. TxVM allows us to define a pair of
related asset types, one a larger denomination of the other, like Gwei
and wei (or dollars and cents). A transaction can use a pair of
Values, one denominated in Gwei for the bulk of the total amount, and
one denominated in wei for the “change,” to represent large quantities
of ether with perfect precision.

For this model to be accurate, it must be possible to convert units of
one asset type into units of its “partner” asset type freely at a
fixed ratio: 1 billion wei for a Gwei (or 100 cents for a dollar).
Here’s how to do that in TxVM.

(Since the numbers involved are simpler, the rest of this document
will use dollars and cents in examples. But the same design works with
Gwei and wei.)

## Issuance and asset types

An asset type in TxVM is uniquely determined by the contract that
issues it, and the data that that contract supplies to the `issue`
instruction.

A vanilla TxVM asset uses a set of public keys and a caller-supplied
“tag” as arguments to the standard asset-issuance contract. Issuance
requires supplying a signature matching (some quorum of) the public
keys.

## Requirements

Our pair of “convertible asset types” must permit ordinary issuance
via signature authorization, but must also permit conversion of some
amount of one asset to the corresponding amount of the other
asset. This is also issuance: when converting cents to dollars, a new
dollar is issued for every 100 cents supplied.

In fact the 100 cents must not only be supplied, they must be
_destroyed_ when issuing the new dollar, in order to keep the total
amount in circulation constant. (Otherwise we’re in the realm of the
other kind of issuance, requiring signatures to increase the supply of
the asset.) Destroying value is done with the `retire` instruction.

So we need a new issuance contract that can operate in one of two
modes: normal issuance, in which signatures permit the creation of
arbitrarily many new units; and conversion, in which some amount of
the partner asset is destroyed while dictating how many new units of
this asset to create.

We’ll treat the case of normal issuance as solved. You can see how
Chain does issuance in
[the standard asset-issuance contract](https://github.com/chain/txvm/blob/main/protocol/txbuilder/standard/issue.go)
in the chain/txvm GitHub repo.

For conversion, we’ll need a way for the issuance contract to test
whether the supplied Value (to be destroyed) belongs to the partner
asset type. It will also have to determine the correct amount to issue
based on the amount destroyed.

## The asset tag

The `issue` instruction consumes an arbitrary string called the “asset
tag.” That’s combined with the identity of the current contract (the
contract containing the `issue` instruction) to get the ID of the
asset to issue.

The standard Chain issuance contract builds the asset tag from the set
of public keys and an arbitrary caller-supplied string. Let’s call
that string the _pre-tag_. (Some of the documentation confusingly
refers to that as a tag too, even though it’s different from the asset
tag consumed by `issue`. For this document we’ll stick with
“pre-tag.”)

By choosing the right format for our pre-tag we can kill two birds
with one stone: allowing the issuance contract to identify values of
the “partner” asset type; and also specifying the conversion ratio
between the two asset types. To do this, let’s stipulate that the
pre-tag must be the _encoding_ (conversion to string format) of a TxVM
_tuple_ with this format:

```
{...arbitrary items..., M, N}
```

where “arbitrary items” can be any TxVM data needed to distinguish
different asset types from one another (a “pre-pre-tag,” if you will),
and M and N are integers expressing the conversion ratio. For two
asset types to be “partners,” they must have the same “arbitrary
items” prefix, and one must end with `{...,M,N}` while the other ends
with the inverse, `{...,N,M}`.

Let’s call this the “pre-tag tuple.” The `encode` instruction can turn
that tuple into the “pre-tag” (a string).

So, for instance, dollars and cents could be defined with the pre-tag
tuples `{'USD',100,1}` and `{'USD',1,100}`.

When performing a conversion, the issuance contract can take the
pre-tag tuple it gets as an argument, then swap its M and N fields to
produce the partner type’s pre-tag tuple. From that it can compute the
expected asset ID, then compare that with the actual asset ID of the
Value supplied for conversion.

The issuance contract can also use M and N to compute the amount to
issue based on the amount being retired.

## Pre-tag tuple subroutines

Let’s start with the code that computes the partner’s asset ID, given
the pre-tag tuple, the quorum threshold, and the tuple full of pubkeys
on the argument stack. (Remember that the quorum and the set of
pubkeys are part of the asset’s identity too, in addition to the
pre-tag tuple.)

The specification for how an asset ID is computed can be found
[here](https://github.com/chain/txvm/blob/main/specifications/txvm.md#asset-id).

```
self      # puts the current contract’s “seed” (a hash) on the stack, which is now: [seed]
get       # move the pre-tag tuple from the arg stack                               [seed tuple]
untuple   # unpack the tuple into its member items, plus its length                 [seed ...arbitrary items... M N len]
2 bury    # move len out of the way for a moment                                    [seed ...arbitrary items... len M N]
swap      # swap M and N                                                            [seed ...arbitrary items... len N M]
2 roll    # get len back                                                            [seed ...arbitrary items... N M len]
tuple     # turn len items back into a tuple                                        [seed partner-tuple]
get get   # move the quorum and pubkeys to the stack                                [seed partner-tuple quorum {pubkey,...}]
3 tuple   # combine those into a single tuple                                       [seed {partner-tuple,quorum,{pubkey,...}}]
encode    # turn that tuple into a string, the “asset tag”                          [seed asset-tag]
cat       # combine two strings into one                                            [(seed+asset-tag)]
'AssetID' # this “domain separator” is part of the asset ID calculation             [(seed+asset-tag) 'AssetID']
vmhash    # compute asset ID                                                        [assetID]
```

Now let’s take a look at the way to calculate the amount to issue,
given a Value to retire. We’ll have to multiply that Value’s amount by
M, ensure it leaves no remainder when divided by N, and then
divide. Let’s assume that the pre-tag tuple and the retire Value are
on the argument stack.

```
get         # move the pre-tag tuple to the contract stack, which is now: [tuple]
dup dup len # copy the tuple twice; get its length (consuming one copy)   [tuple tuple len]
1 sub       # subtract 1                                                  [tuple tuple len-1]
dup 2 bury  # make a copy and move it out of the way                      [tuple len-1 tuple len-1]
field       # extract the len-1’th item from tuple, which is N            [tuple len-1 N]
2 bury      # move it out of the way                                      [N tuple len-1]
1 sub       # subtract 1                                                  [N tuple len-2]
field       # extract the len-2’th item from tuple, which is M            [N M]
get         # get the Value to retire                                     [N M retireval]
amount      # get its amount                                              [N M retireval amount]
2 roll      # move M into place                                           [N retireval amount M]
mul         # multiply                                                    [N retireval amount*M]
dup         # make a copy                                                 [N retireval amount*M amount*M]
3 roll      # move N into place                                           [retireval amount*M amount*M N]
dup 2 bury  # make a copy and move it out of the way                      [retireval amount*M N amount*M N]
mod         # compute amount*M mod N                                      [retireval amount*M N (amount*M % N)]
0 eq verify # compare it to zero, fail if not equal                       [retireval amount*M N]
div         # divide to get the answer                                    [retireval amount*M/N]
```

The final version of our issuance contract will differ slightly from
the code shown here, since data items may not be where these examples
assumed they would be. (For instance, both subroutines shown here get
the pre-tag tuple from the argument stack, but in the final version
that will only happen once, with the value shared between the
subroutines.)

## Putting it together

The complete issuance contract begins by pulling a selector off the
argument stack, telling whether the caller wants a signature-based
issuance or a conversion.

```
get                     # move the selector from the arg stack to the contract stack
1 eq jumpif:$convert    # if it’s equal to 1, jump to the convert clause
```

If the selector is _not_ 1, it executes the code that follows. Here’s
where we would insert the standard asset-issuance contract (which,
again, can be found
[here](https://github.com/chain/txvm/blob/main/specifications/txvm.md#asset-id)). It
needs to be followed by:

```
jump:$end
```

in order to skip over the convert clause, which follows. The convert
clause expects the stack to contain pubkeys, quorum, pre-tag tuple,
and the Value to retire, in that order (with retireval on top).

```
$convert
get         # contract stack is now:                             [retireval]
assetid     # extract the asset ID of the actual Value to retire [retireval actualAssetID]
self        # “seed” of the current contract                     [retireval actualAssetID seed]
get         # pre-tag tuple                                      [retireval actualAssetID seed tuple]
dup 4 bury  # make a copy of the tuple and move it aside         [tuple retireval actualAssetID seed {...,M,N}]
untuple     # unpack the tuple into its member items plus length [tuple retireval actualAssetID seed ... M N len]
swap dup    # move N to the top of the stack and copy it         [tuple retireval actualAssetID seed ... M len N N]
2 roll dup  # move len to the top of the stack and copy that     [tuple retireval actualAssetID seed ... M N N len len]
3 bury      # move one copy of len aside                         [tuple retireval actualAssetID seed ... M len N N len]
4 add bury  # move one copy of N to stack depth len+4            [tuple N retireval actualAssetID seed ... M len N]
2 roll dup  # move M to the top of the stack and copy it         [tuple N retireval actualAssetID seed ... len N M M]
3 roll dup  # move len to the top of the stack and copy it       [tuple N retireval actualAssetID seed ... N M M len len]
2 bury      # move one copy of len aside                         [tuple N retireval actualAssetID seed ... N M len M len]
5 add bury  # move M to stack depth len+5                        [tuple M N retireval actualAssetID seed ... N M len]
tuple       # make the partner’s pre-tag tuple                   [tuple M N retireval actualAssetID seed {...,N,M}]
get         # move the quorum to the contract stack              [tuple M N retireval actualAssetID seed {...,N,M} quorum]
dup 7 bury  # make a copy and move it aside                      [tuple quorum M N retireval actualAssetID seed {...,N,M} quorum]
get         # move the pubkeys to the contract stack             [tuple quorum M N retireval actualAssetID seed {...,N,M} quorum {pubkey,...}]
dup 7 bury  # make a copy and move it aside                      [tuple quorum {pubkey,...} M N retireval actualAssetID seed {...,N,M} quorum {pubkey,...}]
3 tuple     # combine 3 items into a single tuple                [tuple quorum {pubkey,...} M N retireval actualAssetID seed {{...,N,M},quorum,{pubkey,...}}]
encode      # encode that tuple as an asset tag string           [tuple quorum {pubkey,...} M N retireval actualAssetID seed partner-tag]
cat         # concatenate                                        [tuple quorum {pubkey,...} M N retireval actualAssetID (seed+partner-tag)]
'AssetID’   # add hash “domain separator”                        [tuple quorum {pubkey,...} M N retireval actualAssetID (seed+partner-tag) 'AssetID']
vmhash      # compute expected partner asset ID                  [tuple quorum {pubkey,...} M N retireval actualAssetID expectedAssetID]
eq verify   # test actual == expected, fail if they don’t        [tuple quorum {pubkey,...} M N retireval]
```

That’s the end of testing that the retireval has the right asset
type. The convert clause continues with the arithmetic that computes
how much to issue.

```
amount      # get the amount from retireval                      [tuple quorum {pubkey,...} M N retireval retireamt]
3 roll mul  # move M to the top of the stack and multiply        [tuple quorum {pubkey,...} N retireval retireamt*M]
dup         # copy                                               [tuple quorum {pubkey,...} N retireval retireamt*M retireamt*M]
3 roll dup  # move N to the top of the stack and copy            [tuple quorum {pubkey,...} retireval retireamt*M retireamt*M N N]
2 bury      # move one copy aside                                [tuple quorum {pubkey,...} retireval retireamt*M N retireamt*M N]
mod         # compute remainder                                  [tuple quorum {pubkey,...} retireval retireamt*M N (retireamt*M % N)]
0 eq verify # compare it to zero, fail if it’s not               [tuple quorum {pubkey,...} retireval retireamt*M N]
div         # safe to divide, so compute issuance amount         [tuple quorum {pubkey,...} retireval issueamt]
swap        # move retireval to the top of the stack             [tuple quorum {pubkey,...} issueamt retireval]
splitzero   # get an “anchor” (a zero Value) for the issuance    [tuple quorum {pubkey,...} issueamt retireval anchor]
2 bury      # move anchor out of the way                         [tuple quorum {pubkey,...} anchor issueamt retireval]
retire      # destroy retireval                                  [tuple quorum {pubkey,...} anchor issueamt]
4 roll      # move pre-tag tuple to top of the stack             [quorum {pubkey,...} anchor issueamt tuple]
4 roll      # move quorum to top of the stack                    [{pubkey,...} anchor issueamt tuple quorum]
4 roll      # move pubkeys to top of the stack                   [anchor issueamt tuple quorum {pubkey,...}]
3 tuple     # combine 3 items into a single tuple                [anchor issueamt {tuple,quorum,{pubkey,...}}]
encode      # compute this asset’s asset-tag string              [anchor issueamt asset-tag]
issue put   # issue new Value, move it to the arg stack          []
```

Finally, the issuance contract needs a target for the earlier `jump:$end`.

```
$end
```
