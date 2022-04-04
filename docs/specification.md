# Version 0.2

This document specifies the Builder API methods that the Consensus Layer uses to interact with external block builders.

## Structures

### `ExecutionPayloadV1`

Mirror of [`ExecutionPayloadV1`][execution-payload].

### `BlindExecutionPayloadV1`

Equivalent to `ExecutionPayloadV1`, except `transactions` is replaced with `transactionsRoot`.
- `parentHash`: `DATA`, 32 Bytes
- `feeRecipient`:  `DATA`, 20 Bytes
- `stateRoot`: `DATA`, 32 Bytes
- `receiptsRoot`: `DATA`, 32 Bytes
- `logsBloom`: `DATA`, 256 Bytes
- `prevRandao`: `DATA`, 32 Bytes
- `blockNumber`: `QUANTITY`, 64 Bits
- `gasLimit`: `QUANTITY`, 64 Bits
- `gasUsed`: `QUANTITY`, 64 Bits
- `timestamp`: `QUANTITY`, 64 Bits
- `extraData`: `DATA`, 0 to 32 Bytes
- `baseFeePerGas`: `QUANTITY`, 256 Bits
- `blockHash`: `DATA`, 32 Bytes
- `transactionsRoot`: `DATA`, 32 Bytes

## Errors

The list of error codes introduced by this specification can be found below.

| Code | Message | Meaning |
| - | - | - |
| -32000 | Server error | Generic client error while processing request. |
| -32001 | Unknown hash | No block with the provided hash is known. |
| -32002 | Unknown validator | No known mapping between validator and feeRecipient. |
| -32003 | SSZ error | Unable to decode SSZ. |
| -32004 | Unknown block | Block does not match the provided blind header. |
| -32005 | Signature invalid | Provided signature is invalid. |
| -32600 | Invalid Request | The JSON sent is not a valid Request object. |
| -32601 | Method not found | The method does not exist / is not available. |
| -32602 | Invalid params | Invalid method parameter(s). |
| -32603 | Internal error | Internal JSON-RPC error. |
| -32700 | Parse error | Invalid JSON was received by the server. |

## Routines

### Signing

All signature operations should follow the [standard BLS operations][bls] interface defined in `consensus-specs`.

There are two types of data to sign over in the Builder API:
* In-protocol messages, e.g. [`BeaconBlock`][beacon-block], which should compute the signing root using [`computer_signing_root`][compute-signing-root] and use the domain specified for beacon block proposals.
* Builder API messages, e.g. [`builder_setFeeRecipientV1`](#builder_setFeeRecipientV1), which should compute the signing root using [`compute_signing_root`][compute-signing-root] and the domain `DomainType('0xXXXXXXXX')` (TODO: get a proper domain).

As `compute_signing_root` takes `SSZObject` as input, client software should convert in-protocol messages to their SSZ representation to compute the signing root and Builder API messages to the SSZ representations defined below. See [`consensus-specs`][consensus-specs] for the definitions of [`Bytes20`][bytes20], [`ExecutionPayloadHeader`][execution-payload-header], and [`uint256`][uint256].

#### `feeRecipient`

```python
feeRecipient = Bytes20
```

#### `builder_getExecutionPayloadV1` Response

```python
class Response(Container):
    payload: ExecutionPayloadHeader
    value: uint256
```

## Methods

### `builder_setFeeRecipientV1`

#### Request

- method: `builder_setFeeRecipientV1`
- params:
  1. `feeRecipient`: `DATA`, 20 Bytes - Address of account which should receive fees.
  2. `signature`: `DATA`, 96 Bytes - Validator signature over `feeRecipient`.

#### Response

- result: `null`
- error: code and message set in case an exception happens while getting the payload.

#### Specification
1. The builder **MUST** store `feeRecipient` in a map keyed by the recovered public key of the validator.

### `builder_getBlindExecutionPayloadV1`

#### Request

- method: `builder_getBlindExecutionPayloadV1`
- params:
  1. `hash`: `DATA`, 32 Bytes - Hash of block which the validator intends to use as the parent for its proposal.

#### Response

- result: `object`
    - `payload`: [`BlindExecutionPayloadV1`](#blindexecutionpayloadv1).
    - `value`: `DATA`, 32 Bytes - the change in wei balance of the `feeRecipient` account.
    - `signature`: `DATA`, 96 Bytes - BLS signature of the relayer over `payload` and `feeRecipientDiff`
- error: code and message set in case an exception happens while getting the payload.

#### Specification
1. Builder software **SHOULD** return the `payload` that increases the `feeRecipient`'s balance by the most.
2. Builder software **SHOULD** complete the request in under `1 second`.
3. Builder software **MUST** return `-32001: Unknown hash` if the block identified by `hash` does not exist.
4. Builder software **MUST** return `-32002: Unknown validator` if the recovered validator public key has not been mapped to a `feeRecipient`.

### `builder_getExecutionPayloadV1`

#### Request

- method: `builder_getExecutionPayloadV1`
- params:
  1. `block`: `DATA`, arbitray length
  2. `signature`: `DATA`, 96 Bytes

#### Response

- result: [`ExecutionPayloadV1`][#executionpayloadv1].
- error: code and message set in case an exception happens while proposing the payload.

#### Specification
1. Builder software **MUST** verify that `block` is an SSZ encoded [`BeaconBlock`][beacon-block]. If the block is encoded incorrectly, the builder **MUST** return `-32003: SSZ error`. If the block is encoded correctly, but does not match the `BlindExecutionPayloadV1` provided in `builder_getBlindExecutionPayloadV1`, the builder **SHOULD** return `-32004: Unknown block`. If the CL modifies the payload in such a way that it is still valid and the builder is able to unblind it, the builder **MAY** update the payload on it's end to reflect the CL's changes before returning it.
2. Builder software **MUST** verify that `signature` is a BLS signature over `block`, verifiable using [`verify_block_signature`][verify-block-signature] and the validator public key that is expected to propose in the given slot. If the signature is determined to be invalid or from a different validator than expected, the builder **MUST** return `-32005: Invalid signature`.

[consensus-specs]: https://github.com/ethereum/consensus-specs
[bls]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#bls-signatures
[compute-signing-root]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#compute_signing_root
[bytes20]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#aliases
[uint256]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#basic-types
[execution-payload-header]: https://github.com/ethereum/consensus-specs/blob/dev/specs/bellatrix/beacon-chain.md#executionpayloadheader
[execution-payload]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#executionpayloadv1
[hash-tree-root]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#merkleization
[beacon-block]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblock
[verify-block-signature]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beacon-chain-state-transition-function
