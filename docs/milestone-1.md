# Milestone 1 - kintsugi testnet

This initial milestone provides simple middleware logic with minimal consensus client changes, simple networking, no validator authentication, and manual safety mechanism

## Architecture

### Block Proposal

```sequence
participant consensus
participant mev_boost
participant execution
participant relays
Title: Block Proposal
Note over consensus: wait for allocated slot
consensus->mev_boost: engine_forkchoiceUpdatedV1
mev_boost->execution: engine_forkchoiceUpdatedV1
mev_boost->relays: engine_forkchoiceUpdatedV1
Note over mev_boost: begin polling
mev_boost->relays: relay_getPayloadHeaderV1
consensus->mev_boost: builder_getPayloadHeaderV1
mev_boost->execution: engine_getPayloadV1
Note over mev_boost: select best payload
mev_boost-->consensus: builder_getPayloadHeaderV1 response
Note over consensus: sign the block
consensus->mev_boost: builder_proposeBlindedBlockV1
Note over mev_boost: identify payload source
mev_boost->relays: relay_proposeBlindedBlockV1
Note over relays: validate signature
relays-->mev_boost: relay_proposeBlindedBlockV1 response
mev_boost-->consensus: builder_proposeBlindedBlockV1 response
```

## Specification

1. mev-boost must be initialized with a list of (`BLSPublicKey`, `RelayURI`) pairs representing trusted relays.
2. mev-boost must intercept [`engine_forkchoiceUpdatedV1`](#engine_forkchoiceupdatedv1) call from the BN -> EC and forward it to all connected relays to communicate `feeRecipient`.
3. mev-boost must begin polling connected relays for their [`SignedMEVPayloadHeader`](#signedmevpayloadheader) using [`relay_getPayloadHeaderV1`](#relay_getpayloadheaderv1) requests.
4. mev-boost must verify the returned [`SignedMEVPayloadHeader`](#signedmevpayloadheader) signature matches the BLS key associated with the IP of the relay and has a matching `payloadId`.
5. upon receiving a [`builder_getPayloadHeaderV1`](#builder_getpayloadheaderv1) request from the BN, mev-boost must return the [`ExecutionPayloadHeaderV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader) with the highest associated `feeRecipientBalance`. If no eligible payload is received from a relay, mev-boost must request and return a payload from the local execution client using [`engine_getPayloadV1`](#engine_getpayloadv1).
6. the BN must use the [`ExecutionPayloadHeaderV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader) received to assemble and sign a [`SignedBlindedBeaconBlock`](#signedblindedbeaconblock) and return it to mev-boost using [`builder_proposeBlindedBlockV1`](#builder_proposeblindedblockv1).
7. mev-boost must forward the [`SignedBlindedBeaconBlock`](#signedblindedbeaconblock) to all connected relays and attach the matching [`SignedMEVPayloadHeader`](#signedmevpayloadheader) using [`relay_proposeBlindedBlockV1`](#relay_proposeblindedblockv1) to inform the network of which relay created this payload.
8. if an [`ExecutionPayloadV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayload) is returned, mev-boost must verify that the root of the transaction list matches the expected transaction root from the [`SignedBlindedBeaconBlock`](#signedblindedbeaconblock) before returning it to the BN.

### required client modifications

- consensus client must implement [blind transaction signing](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-1-Blind-Transaction-Signing)

## API Docs

Methods are prefixed using the following convention:
- `engine` prefix indicates calls made to the execution client. These methods are specified in the [execution engine APIs](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md).
- `builder` prefix indicates calls made to the mev-boost middleware.
- `relay` prefix indicates calls made to a relay.

### engine_forkchoiceUpdatedV1

See [engine_forkchoiceUpdatedV1](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_forkchoiceupdatedv1).

### engine_getPayloadV1

See [engine_getPayloadV1](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_getpayloadv1).

### builder_getPayloadHeaderV1

#### Request

- method: `builder_getPayloadHeaderV1`
- params:
  1. `payloadId`: `DATA`, 8 Bytes - Identifier of the payload build process

#### Response

- result: [`ExecutionPayloadHeaderV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)
- error: code and message set in case an exception happens while getting the payload.

### builder_proposeBlindedBlockV1

#### Request

- method: `builder_proposeBlindedBlockV1`
- params:
  1. [`SignedBlindedBeaconBlock`](#signedblindedbeaconblock)

#### Response

- result: [`ExecutionPayloadV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayload)
- error: code and message set in case an exception happens while proposing the payload.

Technically, this call only needs to return the `transactions` field of [`ExecutionPayloadV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayload), but we return the full payload for simplicity.

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

Technically, this call only needs to return the `transactions` field of [`ExecutionPayloadV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayload), but we return the full payload for simplicity. 

### Types

#### SignedMEVPayloadHeader

See [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#custom-types) for the definition of fields like `BLSSignature`

- `message`: [`MEVPayloadHeader`](#mevpayloadheader)
- `signature`: `BLSSignature`

#### MEVPayloadHeader

- `payloadHeader`: [`ExecutionPayloadHeaderV1`](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)
- `feeRecipient`: `Data`, 20 Bytes - the fee recipient address requested by the validator
- `feeRecipientBalance`: `Quantity`, 256 Bits - the ending balance of the feeRecipient address

Note: the feeRecipient must match the suggestedFeeRecipient address provided in the [`PayloadAttributesV1`](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#payloadattributesv1) of the associated [`engine_forkchoiceUpdatedV1`](#engine_forkchoiceupdatedv1) request.

#### SignedBlindedBeaconBlock

See [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#custom-types) for the definition of fields like `BLSSignature`

- `message`: [`BlindedBeaconBlock`](#blindedbeaconblock)
- `signature`: `BLSSignature`

#### BlindedBeaconBlock

This is forked from [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#beaconblock) with `body` replaced with `BlindedBeaconBlockBody`

```py
class BlindedBeaconBlock(Container):
    slot: Slot
    proposer_index: ValidatorIndex
    parent_root: Root
    state_root: Root
    body: BlindedBeaconBlockBody
```

#### BlindedBeaconBlockBody

This is forked from [here](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#beaconblockbody) with `execution_payload` replaced with [execution_payload_header](https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/merge/beacon-chain.md#executionpayloadheader)

```py
class BlindedBeaconBlockBody(Container):
    randao_reveal: BLSSignature
    eth1_data: Eth1Data 
    graffiti: Bytes32
    proposer_slashings: List[ProposerSlashing, MAX_PROPOSER_SLASHINGS]
    attester_slashings: List[AttesterSlashing, MAX_ATTESTER_SLASHINGS]
    attestations: List[Attestation, MAX_ATTESTATIONS]
    deposits: List[Deposit, MAX_DEPOSITS]
    voluntary_exits: List[SignedVoluntaryExit, MAX_VOLUNTARY_EXITS]
    sync_aggregate: SyncAggregate
    execution_payload_header: ExecutionPayloadHeader
```

