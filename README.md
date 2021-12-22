# mev-boost

A middleware server written in Go, that sits between an ethereum PoS consensus client and an execution client. It allows consensus clients to outsource block construction to third party block builders as well as fallback to execution clients. See [ethresearch post](https://ethresear.ch/t/mev-boost-merge-ready-flashbots-architecture/11177/) for the high level architecture.

![architecture](/docs/architecture.png)

## Table of Contents
- [mev-boost](#mev-boost)
  - [Implementation Plan](#implementation-plan)
    - [milestone 1 - kintsugi testnet](#milestone-1---kintsugi-testnet)
    - [milestone 2 - security, authentication & reputation](#milestone-2---security-authentication--reputation)
    - [milestone 3 - privacy (optional)](#milestone-3---privacy-optional)
    - [milestone 4 - configurations (optional)](#milestone-4---configurations-optional)
  - [API Docs](#api-docs)
    - [builder_proposeBlindedBlockV1](#builder_proposeblindedblockv1)
    - [builder_getPayloadHeaderV1](#builder_getpayloadheaderv1)
    - [engine_executePayloadV1](#engine_executepayloadv1)
    - [engine_forkchoiceUpdatedV1](#engine_forkchoiceupdatedv1)
    - [Types](#types)
      - [SignedBlindedBeaconBlock](#signedblindedbeaconblock)
      - [BlindedBeaconBlock](#blindedbeaconblock)
      - [BlindedBeaconBlockBody](#blindedbeaconblockbody)
  - [Build](#build)
  - [Test](#test)
  - [Lint](#lint)
  - [Running with mergemock](#running-with-mergemock)

## Implementation Plan

A summary of consensus client changes can be found [here](https://hackmd.io/@paulhauner/H1XifIQ_t).

### milestone 1 - kintsugi testnet

simple middleware logic with minimal consensus client changes, simple networking, no authentication, and manual safety mechanism

#### middleware behavior

- [x] middleware sends `feeRecipient` to relay with direct `engine_forkchoiceUpdatedV1` request at beginning of block
- [x] middleware fetches signed payloads from relay using unauthenticated `getPayloadHeader` request
- [x] middleware selects best payload that matches expected `payloadId` and requests signature from consensus client, this requires passing header object to the consensus client and flagging that it should be returned to the middleware once signed
- [x] middleware returns signed block + initial payload header to relay with direct request

#### required client modifications

- consensus client must implement [blind transaction signing](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-1-Blind-Transaction-Signing)

### milestone 2 - security, authentication & reputation

cleanup consensus client and add security fallbacks

#### middleware behavior

- [ ] middleware requests authenticated `feeRecipient` message from consensus client and gossips over p2p at regular interval
- [ ] add middleware module for verifying previous relay payload validity and accuracy with hard or statistical blacklist (may require modifications to execution client)
- [ ] add middleware module for subscribing to 3rd party relay monitoring service

#### required client modifications

- in event of middleware crash, consensus client must be able to bypass the middleware to reach a local or remote execution client
- consensus client must implement [Proposal Promises](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-2-Proposal-Promises)

### milestone 3 - privacy (optional)

add p2p comms mechanisms to prevent validator deanonymization

#### middleware behavior

- [ ] middleware gossips signed block + initial payload header over p2p

#### required client modifications

- consensus client must implement [New Gossipsub Topics](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-3-New-Gossipsub-Topics)

### milestone 4 - configurations (optional)

add optional configurations to provide alternative guarantees

- [ ] consider adding direct `relay_forkchoiceUpdatedV1` call to relay for syncing state
- [ ] consider returning full payload directly to validator as optimization
- [ ] consider adding merkle proof of payment to shift verification requirements to the relay

## API Docs

Methods are prefixed using the following convention:
- `engine` prefix indicates calls made to the execution client. These methods are specified in the [execution engine APIs](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md).
- `builder` prefix indicates calls made to the mev-boost middleware.
- `relay` prefix indicates calls made to a relay.

### engine_executePayloadV1

See [engine_executePayloadV1](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_executepayloadv1).

### engine_forkchoiceUpdatedV1

See [engine_forkchoiceUpdatedV1](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_forkchoiceupdatedv1).

### builder_proposeBlindedBlockV1

#### Request

- method: `builder_proposeBlindedBlockV1`
- params:
  1. [`SignedBlindedBeaconBlock`](#signedblindedbeaconblock)

#### Response

- result: [`ExecutionPayloadV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayload)
- error: code and message set in case an exception happens while proposing the payload.

### builder_getPayloadHeaderV1

#### Request

- method: `builder_getPayloadHeaderV1`
- params:
  1. `payloadId`: `DATA`, 8 Bytes - Identifier of the payload build process

#### Response

- result: [`ExecutionPayloadHeaderV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)
- error: code and message set in case an exception happens while getting the payload.

### relay_getPayloadHeaderV1

#### Request

- method: `relay_getPayloadHeaderV1`
- params:
  1. `payloadId`: `DATA`, 8 Bytes - Identifier of the payload build process

#### Response

- result: [`SignedMEVPayloadHeader`](#signedmevpayloadheader)
- error: code and message set in case an exception happens while getting the payload.

### relay_proposeBlindedBlockV1

#### Request

- method: `relay_proposeBlindedBlockV1`
- params:
  1. [`SignedBlindedBeaconBlock`](#signedblindedbeaconblock)
  2. [`SignedMEVPayloadHeader`](#signedmevpayloadheader)

#### Response

- result: [`ExecutionPayloadV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayload)
- error: code and message set in case an exception happens while proposing the payload.

### Types

#### SignedMEVPayloadHeader

See [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#custom-types) for the definition of fields like `BLSSignature`

- message: [MEVPayloadHeader](#mevpayloadheader)
- signature: BLSSignature

#### MEVPayloadHeader

- payloadHeader: [`ExecutionPayloadHeaderV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)
- feeRecipientDiff: Quantity, 256 Bits - the change in balance of the feeRecipient address

#### SignedBlindedBeaconBlock

See[here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#custom-types) for the definition of fields like `BLSSignature`

- message: [BlindedBeaconBlock](#blindedbeaconblock)
- signature: BLSSignature

#### BlindedBeaconBlock

This is forked from [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#beaconblock) with `body` replaced with `BlindedBeaconBlockBody`

- slot: Slot
- proposer_index: ValidatorIndex
- parent_root: Root
- state_root: Root
- body: BlindedBeaconBlockBody

#### BlindedBeaconBlockBody

This is forked from [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#beaconblockbody) with `execution_payload` replaced with [execution_payload_header](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)

- randao_reveal: BLSSignature
- eth1_data: Eth1Data
- graffiti: Bytes32
- proposer_slashings: List[ProposerSlashing, MAX_PROPOSER_SLASHINGS]
- attester_slashings: List[AttesterSlashing, MAX_ATTESTER_SLASHINGS]
- attestations: List[Attestation, MAX_ATTESTATIONS]
- deposits: List[Deposit, MAX_DEPOSITS]
- voluntary_exits: List[SignedVoluntaryExit, MAX_VOLUNTARY_EXITS]
- sync_aggregate: SyncAggregate
- execution_payload_header: ExecutionPayloadHeader

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

We use `revive` as a linter. You need to install it with `go install github.com/mgechev/revive@latest`

```
make lint
```

## Running with mergemock

We are currently testing using a forked version of mergemock, see https://github.com/flashbots/mergemock

Make sure you've setup and built mergemock first, refer to its [README](https://github.com/flashbots/mergemock#quick-start) but here's a quick setup guide:

```
git clone https://github.com/flashbots/mergemock.git
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
