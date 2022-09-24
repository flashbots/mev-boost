![mev-boost](https://user-images.githubusercontent.com/116939/179831878-dc6a0f76-94f4-46cc-bafd-18a3a4b58ea4.png)

#

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/mev-boost)](https://goreportcard.com/report/github.com/flashbots/mev-boost)
[![Test status](https://github.com/flashbots/mev-boost/workflows/Tests/badge.svg)](https://github.com/flashbots/mev-boost/actions?query=workflow%3A%22Tests%22)

## What is MEV-Boost?

`mev-boost` is open source middleware run by validators to access a competitive block-building market. MEV-Boost was built by Flashbots as an implementation of [proposer-builder separation (PBS)](https://ethresear.ch/t/proposer-block-builder-separation-friendly-fee-market-designs/9725) for proof-of-stake (PoS) Ethereum.

With MEV-Boost, validators can access blocks from a marketplace of builders. Builders produce blocks containing transaction orderflow and a fee for the block proposing validator. Separating the role of proposers from block builders promotes greater competition, decentralization, and censorship-resistance for Ethereum.

## How does MEV-Boost work?


PoS node operators must run three pieces of software: a validator client, consensus client, and an execution client. MEV-boost is a sidecar for the Consensus Client, a separate piece of open source software, which queries and outsources block-building to a network of builders. Block builders prepare full blocks, optimizing for MEV extraction and fair distribution of rewards. They then submit their blocks to relays.

Relays aggregate blocks from **multiple** builders in order to select the block with the highest fees. One instance of MEV-boost can be configured by a validator to connect to **multiple** relays. The Consensus Layer client of a validator proposes the most profitable block received from MEV-boost to the Ethereum network for attestation and block inclusion.


![mev-boost service integration overview](https://raw.githubusercontent.com/flashbots/mev-boost/main/docs/mev-boost-integration-overview.png)

## Who can run MEV-Boost?

MEV-Boost is a piece of software that any PoS Ethereum node operator (including solo validators) can run as part of their Beacon Client software. It is compatible with any Ethereum consensus client. Support and installation instructions for each client can be found [here](#installing).


---

See also:

* [boost.flashbots.net](https://boost.flashbots.net)
* [mev-boost Docker images](https://hub.docker.com/r/flashbots/mev-boost)
* [wiki](https://github.com/flashbots/mev-boost/wiki) & [troubleshooting guide](https://github.com/flashbots/mev-boost/wiki/Troubleshooting)
* [mev-boost relay source code](https://github.com/flashbots/mev-boost-relay)
* Specs:
  * [Builder API](https://ethereum.github.io/builder-specs)
  * [Flashbots Relay API](https://flashbots.notion.site/Relay-API-Spec-5fb0819366954962bc02e81cb33840f5)


# Table of Contents

- [Background](#background)
- [Installing](#installing)
- [Usage](#usage)
- [The Plan](#the-plan)
- [API](#api)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

# Background

MEV is a centralizing force on Ethereum. Unattended, the competition for MEV opportunities leads to consensus security instability and permissioned communication infrastructure between traders and block producers. This erodes neutrality, transparency, decentralization, and permissionlessness.

Proposer/block-builder separation (PBS) was initially proposed by Ethereum researchers as a response to the risk that MEV poses to decentralization of consensus networks. They have suggested that uncontrolled MEV extraction promotes economies of scale which are centralizing in nature, and complicate decentralized pooling.

Flashbots is a research and development organization working on mitigating the negative externalities of MEV. Flashbots started as a builder specializing in MEV extraction in proof-of-work Ethereum to democratize access to MEV and make the most profitable blocks available to all miners. >90% of miners are outsourcing some of their block construction to Flashbots today.

In the future, [proposer/builder separation](https://ethresear.ch/t/two-slot-proposer-builder-separation/10980) will be enshrined in the Ethereum protocol itself to further harden its trust model.

Read more in [Why run mev-boost?](https://writings.flashbots.net/writings/why-run-mevboost/) and in the [Frequently Asked Questions](https://github.com/flashbots/mev-boost/wiki/Frequently-Asked-Questions).

# Installing

`mev-boost` can run in any machine, as long as it is reachable by the beacon client. The most common setup is to install it in the same machine as the beacon client. Multiple beacon-clients can use a single mev-boost instance. The default port is 18550.

See also [Rémy Roy's guide](https://github.com/remyroy/ethstaker/blob/main/prepare-for-the-merge.md#installing-mev-boost) for comprehensive instructions on installing, configuring and running mev-boost.

## Binaries

Each release includes binaries from Linux, Windows and macOS (portable build, for amd and arm). You can find the latest release at
https://github.com/flashbots/mev-boost/releases


## From source

Requires [Go 1.18+](https://go.dev/doc/install).

### `go install`

Install mev-boost with `go install`:

```bash
go install github.com/flashbots/mev-boost@latest
mev-boost -help
```

### Clone & build

clone the repository and build it:

```bash
git clone https://github.com/flashbots/mev-boost.git
cd mev-boost
make build
make build-portable

# Show the help
./mev-boost -help
```

## From Docker image

We maintain a mev-boost Docker images at https://hub.docker.com/r/flashbots/mev-boost

- [Install Docker Engine](https://docs.docker.com/engine/install/)
- Pull & run the latest image:

```bash
# Get the default mev-boost image
docker pull flashbots/mev-boost:latest

# Get the portable mev-boost image
docker pull flashbots/mev-boost:latest-portable

# Run it
docker run flashbots/mev-boost -help
```

## Systemd configuration

You can run mev-boost with a systemd config (`/etc/systemd/system/mev-boost.service`) like this:

```ini
[Unit]
Description=mev-boost
Wants=network-online.target
After=network-online.target

[Service]
User=mev-boost
Group=mev-boost
WorkingDirectory=/home/mev-boost
Type=simple
Restart=always
RestartSec=5
ExecStart=/home/mev-boost/bin/mev-boost \
		-mainnet \
		-relay-check \
		-relays YOUR_RELAY_CHOICE

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

If mev-boost crashes with [`"SIGILL: illegal instruction"`](https://github.com/flashbots/mev-boost/issues/256) then you need to use a portable build:

You can either use a [portable Docker image](https://hub.docker.com/r/flashbots/mev-boost/tags), or install/build the portable build like this:

```bash
# using `go install`
CGO_CFLAGS="-O -D__BLST_PORTABLE__" go install github.com/flashbots/mev-boost@latest

# build from source
make build-portable
```


# Usage

A single mev-boost instance can be used by multiple beacon nodes. Note that aside from running mev-boost, you will need to configure your beacon node to connect to mev-boost and your validator to allow it to register with the relay. This configuration varies and a guide for each consensus client can be found on the [MEV-boost website](https://boost.flashbots.net/#block-356364ebd7cc424fb524428ed0134b21).


### Mainnet

Run mev-boost pointed at our [Mainnet Relay](https://boost-relay.flashbots.net/):

```
 ./mev-boost -mainnet -relay-check -relays https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net
```

### Goerli testnet

Run mev-boost pointed at our [Goerli Relay](https://builder-relay-goerli.flashbots.net/):

```
 ./mev-boost -goerli -relay-check -relays https://0xafa4c6985aa049fb79dd37010438cfebeb0f2bd42b115b89dd678dab0670c1de38da0c4e9138c9290a398ecd9a0b3110@builder-relay-goerli.flashbots.net
```

### Ropsten testnet

Run mev-boost pointed at our [Ropsten Relay](https://builder-relay-ropsten.flashbots.net/):

```
 ./mev-boost -ropsten -relay-check -relays https://0xb124d80a00b80815397b4e7f1f05377ccc83aeeceb6be87963ba3649f1e6efa32ca870a88845917ec3f26a8e2aa25c77@builder-relay-ropsten.flashbots.net
```

### Kiln testnet

Run mev-boost pointed at our [Kiln Relay](https://builder-relay-kiln.flashbots.net):

```bash
./mev-boost -kiln -relay-check -relays https://0xb5246e299aeb782fbc7c91b41b3284245b1ed5206134b0028b81dfb974e5900616c67847c2354479934fc4bb75519ee1@builder-relay-kiln.flashbots.net
```

### Sepolia testnet

Run mev-boost pointed at our [Sepolia Relay](https://builder-relay-sepolia.flashbots.net/):

```
 ./mev-boost -sepolia -relay-check -relays https://0x845bd072b7cd566f02faeb0a4033ce9399e42839ced64e8b2adcfc859ed1e8e1a5a293336a49feac6d9a5edb779be53a@builder-relay-sepolia.flashbots.net
```


#### `test-cli`

`test-cli` is a utility to execute all proposer requests against mev-boost+relay. See also the [test-cli readme](cmd/test-cli/README.md).


# API

`mev-boost` implements the latest [Builder Specification](https://github.com/ethereum/builder-specs).

```mermaid
sequenceDiagram
    participant consensus
    participant mev_boost
    participant relays
    Title: Block Proposal
    Note over consensus: validator starts up
    consensus->>mev_boost: registerValidator
    mev_boost->>relays: registerValidator
    Note over consensus: wait for allocated slot
    consensus->>mev_boost: getHeader
    mev_boost->>relays: getHeader
    relays-->>mev_boost: getHeader response
    Note over mev_boost: verify response matches expected
    Note over mev_boost: select best payload
    mev_boost-->>consensus: getHeader response
    Note over consensus: sign the header
    consensus->>mev_boost: submitBlindedBlock
    Note over mev_boost: identify payload source
    mev_boost->>relays: submitBlindedBlock
    Note over relays: validate signature
    relays-->>mev_boost: submitBlindedBlock response
    Note over mev_boost: verify response matches expected
    mev_boost-->>consensus: submitBlindedBlock response
```

# The Plan

`mev-boost` is the next step on our exploration towards a trustless and decentralized MEV market. It is a service developed in collaboration with the Ethereum developers and researchers.

The roadmap, expected deliveries and estimated deadlines are described in [the plan](https://github.com/flashbots/mev-boost/wiki/The-Plan-(tm)). Join us in this repository while we explore the remaining [open research questions](https://github.com/flashbots/mev-boost/wiki/Research#open-questions) with all the relevant organizations in the ecosystem.

# Maintainers

- [@metachris](https://github.com/metachris)
- [@Ruteri](https://github.com/Ruteri)
- [@elopio](https://github.com/elopio)

# Contributing

[Flashbots](https://flashbots.net) is a research and development collective working on mitigating the negative externalities of decentralized economies. We contribute with the larger free software community to illuminate the dark forest.

You are welcome here <3.

- If you have a question, feedback or a bug report for this project, please [open a new Issue](https://github.com/flashbots/mev-boost/issues).
- If you would like to contribute with code, check the [CONTRIBUTING file](CONTRIBUTING.md) for further info about the development environment.
- We just ask you to be nice. Read our [code of conduct](CODE_OF_CONDUCT.md).

# Security

If you find a security vulnerability on this project or any other initiative related to Flashbots, please let us know sending an email to security@flashbots.net.

## Audits

- [20220620](docs/audit-20220620.md), by [lotusbumi](https://github.com/lotusbumi).

# License

The code in this project is free software under the [MIT License](LICENSE).

Logo by [@lekevicius](https://twitter.com/lekevicius) on CC0 license.

---

Made with ☀️ by the ⚡🤖 collective.
