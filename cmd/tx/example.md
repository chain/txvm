# Creating realistic blockchain data

This document describes how to use Chain’s command-line tools to
produce realistic blockchain data.

## Creating the initial block

Start by creating block 1, the initial block of a blockchain. This
requires choosing a _quorum_ and a set of _signers_ to validate each
block. The quorum is an integer from 0 to the number of signers. It
takes that many signatures to validate a block. The signers are
specified by their public keys. Validating signatures are produced
from the corresponding private keys.

Let’s choose a quorum of 1 and generate a single random public/private
keypair for signing blocks.

```sh
ed25519 gen | tee blocksigner.prv | ed25519 pub >blocksigner.pub
```

Now the private key for the blocksigner is stored in `blocksigner.prv`
and the public key is stored in `blocksigner.pub`. (Naturally, if
these are to be used for anything important, the private key should be
kept very secure.)

Generate the initial block like this:

```sh
block new -quorum 1 `hex <blocksigner.pub` >initial-block
```

Get the block’s ID (which is also the blockchain ID) like this:

```sh
block hash <initial-block >blockchain-id
```

The history of the blockchain is encapsulated in state snapshots,
updated after each block. We can create our initial state snapshot
like this:

```sh
bcstate -block initial-block > initial-snapshot
```

## Creating the first transaction

The new blockchain initially contains no value, so the first
transaction must issue some.

To issue some value, it’s necessary to choose an amount and an asset
type to issue. The asset type is defined by the set of _authorizers_
and an optional _tag_. The authorizers are the signers who can
authorize issuance, just as a set of signers is needed to validate a
block.

Let’s choose a quorum of 1 and another random keypair for our asset
authorizer. Let’s further choose to omit the optional tag.

```sh
ed25519 gen | tee assetsigner.prv | ed25519 pub >assetsigner.pub
```

Now we can start to build a transaction that issues (let’s say) 100
units of that asset.

```sh
tx build issue -blockchain `hex <blockchain-id` -quorum 1 -prv `hex <assetsigner.prv` -pub `hex <assetsigner.pub` -amount 100
```

However, this does not produce a complete transaction. (Running the
command above gives the error “non-signature-check item on stack after
finalize.”) The 100 units it issues is left “on the stack” by this
transaction. It should instead be bundled into an “output contract”
from which it can be spent in a future transaction.

To move those 100 units into an output contract, it’s necessary to
choose another quorum and set of signers to identify who’s allowed to
spend it in the future. For simplicity let’s again choose a quorum of
1 and a single new keypair; and in keeping with convention, let’s call
the recipient of these funds “Alice.”

```sh
ed25519 gen | tee alice.prv | ed25519 pub >alice.pub
```

Putting those 100 units into an output contract also requires being
able to name the asset type they belong to. This asset ID can be
computed from the quorum, public key, and tag we chose above.

```sh
assetid 1 `hex <assetsigner.pub` >asset-id
```

Now we can add an output contract to the transaction.

```sh
tx build issue -blockchain `hex <blockchain-id` -quorum 1 -prv `hex <assetsigner.prv` -pub `hex <assetsigner.pub` -amount 100 output -quorum 1 -pub `hex <alice.pub` -amount 100 -assetid `hex <asset-id` >issue-100-to-alice
```

## Adding the transaction to a block

We can now create a new block for the blockchain containing this
transaction.

```sh
block build -snapout snapshot.2 issue-100-to-alice <initial-snapshot >block.2
```

This command also creates an updated state snapshot in the file
`snapshot.2`.

The output of `block build` is an _unsigned_ block. To sign it, run
the output through `block sign` like this:

```sh
block build -snapout snapshot.2 issue-100-to-alice <initial-snapshot >unsigned-block.2
block sign -prev `block header <initial-block | hex` `hex <blocksigner.prv` <unsigned-block.2 >block.2
```

If you now feed `unsigned-block.2` to `block validate` it will fail,
but `block.2` will succeed.

## Spending the earlier output contract

Now suppose Alice wants to send 20 of her 100 units to Bob.

Bob will need a quorum (let’s choose 1 again) and a keypair for receiving that transfer.

```sh
ed25519 gen | tee bob.prv | ed25519 pub >bob.pub
```

Alice will need to _input_ the earlier output contract. For this
she’ll need the amount, asset ID, and _anchor_ of that contract. To
discover the anchor, she can run:

```sh
tx log -witness <issue-100-to-alice
```

That produces output resembling this:

```
{'N', x'c47b8c5e753c80d74c4b33fdba41a038f1996e4d34317a60d4fbd8d9c9f3b85e', x'e2003a140131ec4f31b328701ef1242ee470ef2f72ad21b4f55135556f70f7de', x'e475a5942e2873c64b69a4bd39080a9d67921c5bf6846e5049e262a00d86ab4e', 1525913631509}
{'R', x'e2003a140131ec4f31b328701ef1242ee470ef2f72ad21b4f55135556f70f7de', 0, 1525913631509}
{'A', x'c47b8c5e753c80d74c4b33fdba41a038f1996e4d34317a60d4fbd8d9c9f3b85e', 100, x'c0c40e540441a843cb9a36bf00aa29677fcfb9e92628984636f57f7843a7c072', x'89f27908025eeca75eeb31a5821537e52892ca29016cd354736e6fe508ee2ca8'}
{'L', x'e2003a140131ec4f31b328701ef1242ee470ef2f72ad21b4f55135556f70f7de', ''}
{'L', x'47bb956c9e8844bf5d3cc3ed93d01e275353523142b6fb3999b5d0e11a958ffa', ''}
{'L', x'47bb956c9e8844bf5d3cc3ed93d01e275353523142b6fb3999b5d0e11a958ffa', ''}
{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'd017ec797d3ad4efb48d1f72bb7a67a1fc3e461cd5f3878b0e4c14dac2b4f249'}
{'R', x'0000000000000000000000000000000000000000000000000000000000000000', 0, 1525913631509}
{'L', x'0000000000000000000000000000000000000000000000000000000000000000', ''}
{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'53845470c98cfbe9b43d38937aa3b1c1274833f40ba02caad9635fdba781808e'}
```

Alice finds the anchor in the line beginning with an `O` for “output.”
The final item on that line
(d017ec797d3ad4efb48d1f72bb7a67a1fc3e461cd5f3878b0e4c14dac2b4f249 in
this example) is the hex-encoded anchor.

Now Alice can construct her transaction. It must input that earlier
contract, then output:
- 20 units to Bob
- the remaining 80 units back to Alice, as “change”

```sh
tx build input -quorum 1 -prv `hex <alice.prv` -pub `hex <alice.pub` -amount 100 -assetid `hex <asset-id` -anchor d017ec797d3ad4efb48d1f72bb7a67a1fc3e461cd5f3878b0e4c14dac2b4f249 output -quorum 1 -pub `hex <bob.pub` -amount 20 -assetid `hex <asset-id` output -quorum 1 -pub `hex <alice.pub` -amount 80 -assetid `hex <asset-id` >transfer-20-to-bob
```
