# mev-boost

[![Test status](https://github.com/flashbots/mev-boost/workflows/Go/badge.svg)](https://github.com/flashbots/mev-boost/actions?query=workflow%3A%22Go%22)
[![Discord](https://img.shields.io/discord/755466764501909692)](https://discord.gg/7hvTycdNcK)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

A service that allows Ethereum Consensus Layer (CL) clients to outsource block construction to third party block builders in addition to execution clients. See [ethresearch post](https://ethresear.ch/t/mev-boost-merge-ready-flashbots-architecture/11177/) for the high level architecture and **[`docs/specification.md`](https://github.com/flashbots/mev-boost/blob/main/docs/specification.md)** for the specification and implementation details.

![mev-boost service integration overview](https://raw.githubusercontent.com/flashbots/mev-boost/main/docs/mev-boost-integration-overview.png)

v0.2 request flow:

```mermaid
sequenceDiagram
    participant consensus
    participant mev_boost
    participant relays
    Title: Block Proposal
    Note over consensus: sign fee recipient announcement
    consensus->>mev_boost: builder_registerValidator
    mev_boost->>relays: builder_registerValidator
    Note over consensus: wait for allocated slot
    consensus->>mev_boost: builder_getHeader
    mev_boost->>relays: builder_getHeader
    relays-->>mev_boost: builder_getHeader response
    Note over mev_boost: verify response matches expected
    Note over mev_boost: select best payload
    mev_boost-->>consensus: builder_getHeader response
    Note over consensus: sign the block
    consensus->>mev_boost: builder_getPayload
    Note over mev_boost: identify payload source
    mev_boost->>relays: builder_getPayload
    Note over relays: validate signature
    relays-->>mev_boost: builder_getPayload response
    Note over mev_boost: verify response matches expected
    mev_boost-->>consensus: builder_getPayload response
```

## Implementation Plan

See https://github.com/flashbots/mev-boost/wiki/The-Plan-(tm)

References:

* Specification: https://github.com/flashbots/mev-boost/blob/main/docs/specification.md
* https://ethresear.ch/t/mev-boost-merge-ready-flashbots-architecture/11177/
* https://hackmd.io/@paulhauner/H1XifIQ_t

## Build

```
make build
```

and then run it with:

```
./mev-boost
```

## Test

```
make test
```

## Lint

We use `revive` as a linter and [staticcheck](https://staticcheck.io/). You need to install them with

```bash
go install github.com/mgechev/revive@latest
go install honnef.co/go/tools/cmd/staticcheck@master
```

Lint and check the project:

```bash
make lint
```

## Running with mergemock

We are currently testing using a forked version of mergemock, see https://github.com/flashbots/mergemock

Make sure you've setup and built mergemock first, refer to its [README](https://github.com/flashbots/mergemock#quick-start) but here's a quick setup guide:

```
git clone -b v021-upstream https://github.com/flashbots/mergemock.git
cd mergemock
go build . mergemock
wget https://gist.githubusercontent.com/lightclient/799c727e826483a2804fc5013d0d3e3d/raw/2e8824fa8d9d9b040f351b86b75c66868fb9b115/genesis.json
```

Then you can run an integration test with mergemock, spawning both a mergemock execution engine and a mergemock consensus client as well as mev-boost:

```
cd mev-boost
make run-mergemock-integration
```

The path to the mergemock repo is assumed to be `../mergemock`, you can override like so:

```
make MERGEMOCK_DIR=/PATH-TO-MERGEMOCK-REPO run-mergemock-integration
```

to run mergemock in dev mode:

```
make MERGEMOCK_BIN='go run .' run-mergemock-integration
```
