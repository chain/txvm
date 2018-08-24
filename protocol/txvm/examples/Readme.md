# TxVM examples

This directory collects examples of contracts that can be created with TxVM.

In some cases,
later examples build upon or refer to work introduced in earlier examples.

Disclaimer:
The examples here are simplified for the sake of clarity.
More importantly,
they have not been thoroughly tested.
They make a good survey of what’s possible with TxVM and are a great way to get started learning how to write TxVM contracts,
but they must not be used to control any real value.

* [Account](account.md).
  How to layer an account model on top of TxVM’s UTXO model.
* [Convertible assets](convertible.md).
  How to define two related asset types that allow units of one to be freely exchanged for units of the other at a fixed ratio
  (think dollars and cents).
* [Cryptocurrency](cryptocurrency.md).
  How to define an asset type that limits and democratizes its own issuance,
  two requirements for a cryptocurrency.
* [Orderbook](orderbook.md).
  How to publish and fulfill offers to exchange units of one asset type for units of another.
* [Path payments](path.md).
  How to combine orderbook offers,
  routing a payment from a buyer with one currency to a seller whose offer requires a different currency.
* [Payment channels
  (a la Lightning)](channel.md).
  How to lock up some funds for a while so two or more parties can transact quickly and privately off-chain,
  settling back to the blockchain when they’re done.
* [Escrow payments](escrow.md).
  How to entrust payments to a third party,
  who authorizes their release when some condition is met.
* Collateralized loan TBD
* Merkleized clauses TBD
