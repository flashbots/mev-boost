# mev-boost

A middleware server written in Go, that sits between an ethereum PoS consensus client and an execution client. It allows consensus clients to outsource block construction to third party block builders as well as fallback to execution clients. See [ethresearch post](https://ethresear.ch/t/mev-boost-merge-ready-flashbots-architecture/11177/) for the high level architecture.

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

- [ ] consider adding direct `engine_forkchoiceUpdatedV1` call to relay for syncing state
- [ ] consider returning full payload directly to validator as optimization
- [ ] consider adding merkle proof of payment to shift verification requirements to the relay

## API Docs

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
  - NOTE: this slightly varies from the upstream `ExecutionPayloadV1`, in that the `transactions` field is optional and not meant to be used by the caller.
- error: code and message set in case an exception happens while getting the payload.

### engine_executePayloadV1

#### Request

- method: `engine_executePayloadV1`
- params:
  1. [`ExecutionPayloadV1`](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#ExecutionPayloadV1)

#### Response

- result: `object`
  - `status`: `enum` - `"VALID" | "INVALID" | "SYNCING"`
  - `latestValidHash`: `DATA|null`, 32 Bytes - the hash of the most recent _valid_ block in the branch defined by payload and its ancestors
  - `validationError`: `String|null` - a message providing additional details on the validation error if the payload is deemed `INVALID`
- error: code and message set in case an exception happens while executing the payload.

### engine_forkchoiceUpdatedV1

#### Request

- method: "engine_forkchoiceUpdatedV1"
- params:
  1. `forkchoiceState`: `Object` - instance of [`ForkchoiceStateV1`](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#ForkchoiceStateV1)
  2. `payloadAttributes`: `Object|null` - instance of [`PayloadAttributesV1`](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#PayloadAttributesV1) or `null`

#### Response

- result: `object`
  - `status`: `enum` - `"SUCCESS" | "SYNCING"`
  - `payloadId`: `DATA|null`, 8 Bytes - identifier of the payload build process or `null`
- error: code and message set in case an exception happens while updating the forkchoice or initiating the payload build process.

### Types

#### SignedBlindedBeaconBlock

See https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#custom-types for the definition of fields like `BLSSignature`

- message: BlindedBeaconBlock
- signature: BLSSignature

#### BlindedBeaconBlock

This is forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#beaconblock with `body` replaced with `BlindedBeaconBlockBody`

- slot: Slot
- proposer_index: ValidatorIndex
- parent_root: Root
- state_root: Root
- body: BlindedBeaconBlockBody

#### BlindedBeaconBlockBody

This is forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#beaconblockbody with `execution_payload` replaced with [execution_payload_header](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)

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

Make sure you've setup and built mergemock first, refer to its [README](https://github.com/protolambda/mergemock#quick-start)

```
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
