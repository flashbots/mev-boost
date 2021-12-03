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

### milestone 2 - security & reputation

cleanup consensus client and add security fallbacks

#### middleware behavior
- [ ] add middleware module for verifying previous relay payload validity and accuracy with hard or statistical blacklist (may require modifications to execution client)
- [ ] add middleware module for subscribing to 3rd party relay monitoring service

#### required client modifications
- in event of middleware crash, consensus client must be able to bypass the middleware to reach a local or remote execution client

### milestone 3 - authentication & privacy (optional)

add authentication and p2p comms mechanisms to prevent validator deanonymization

#### middleware behavior
- [ ] middleware signs `feeRecipient` message and gossips over p2p at regular interval
- [ ] middleware gossips signed block + initial payload header over p2p

#### required client modifications
- consensus client must implement [Proposal Promises](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-2-Proposal-Promises)
- consensus client must implement [New Gossipsub Topics](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-3-New-Gossipsub-Topics)

### milestone 4 - configurations (optional)

add optional configurations to provide alternative guarantees

- [ ] consider adding direct `engine_forkchoiceUpdatedV1` call to relay for syncing state
- [ ] consider returning full payload directly to validator as optimization
- [ ] consider adding merkle proof of payment to shift verification requirements to the relay

## API Docs

### relay_proposePayloadV1

#### Request

- method: `relay_proposePayloadV1`
- params:
  1. [`SignedBeaconBlockHeader`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#signedbeaconblockheader)

#### Response

- result: `object`
  - `status`: `enum` - `"VALID" | "INVALID" | "SYNCING"`
  - `latestValidHash`: `DATA|null`, 32 Bytes - the hash of the most recent _valid_ block in the branch defined by payload and its ancestors
  - `validationError`: `String|null` - a message providing additional details on the validation error if the payload is deemed `INVALID`
- error: code and message set in case an exception happens while executing the payload.

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

### engine_getPayloadV1

#### Request

- method: `engine_getPayloadV1`
- params:
  1. `payloadId`: `DATA`, 8 Bytes - Identifier of the payload build process

#### Response

- result: [`ExecutionPayloadV1`](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#ExecutionPayloadV1)
- error: code and message set in case an exception happens while getting the payload.

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
