# Version 0.2

This document specifies the Builder API methods that the Consensus Layer uses to interact with external block builders.

## Structures

### `ExecutionPayloadV1`

Mirror of [`ExecutionPayloadV1`][execution-payload].

### `BlindExecutionPayloadV1`

Equivalent to `ExecutionPayloadV1`, except `transactions` is replaced with `transactionsHash`. This is the [SSZ hash tree root][hash-tree-root] of `transactions`.

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
 - `transactionsHash`: `DATA`, 32 Bytes

## Errors

The list of error codes introduced by this specification can be found below.

| Code | Message | Meaning |
| - | - | - |
| -32000 | Server error | Generic client error while processing request. |
| -32001 | Unknown hash | No block with the provided hash is known. |
| -32002 | SSZ error | Unable to decode SSZ. |
| -32003 | Unknown block | Block does not match the provided blind header. |
| -32004 | Signature invalid | Provided signature is invalid. |
| -32600 | Invalid Request | The JSON sent is not a valid Request object. |
| -32601 | Method not found | The method does not exist / is not available. |
| -32602 | Invalid params | Invalid method parameter(s). |
| -32603 | Internal error | Internal JSON-RPC error. |
| -32700 | Parse error | Invalid JSON was received by the server. |

## Methods

### `builder_getBlindExecutionPayloadV1`

#### Request

- method: `builder_getBlindExecutionPayloadV1`
- params:
  1. `hash`: `DATA`, 32 Bytes - Hash of block which the validator intends to use as the parent for its proposal.

#### Response

- result: `object`
    - `payload`: [`BlindExecutionPayloadV1`](#blindexecutionpayloadv1).
    - `feeRecipientDiff`: `DATA`, 32 Bytes - the change in wei balance of the `feeRecipient` account.
    - `signature`: `DATA`, 96 Bytes - BLS signature of the relayer over `payload` and `feeRecipientDiff`
- error: code and message set in case an exception happens while getting the payload.

#### Specification
1. Builder software **SHOULD** return the `payload` which increases the `feeRecipient`'s balance by the most.
2. Builder software **SHOULD** complete the request in under `1 second`.
3. Builder software **MUST** return `-32001: Unknown hash` if the block identified by `hash` does not exist.

### `builder_getExecutionPayloadV1`

#### Request

- method: `builder_getExecutionPayloadV1`
- params:
  1. `header`: `DATA`, arbitray length
  2. `signature`: `DATA`, 96 Bytes

#### Response

- result: [`ExecutionPayloadV1`][#executionpayloadv1].
- error: code and message set in case an exception happens while proposing the payload.

#### Specification
1. Builder software **MUST** verify that `header` is an SSZ encoded [`BeaconBlockHeader`][beacon-block-header]. If the header is encoded incorrectly, the builder **MUST** return `-32002: SSZ error`. If the header is encoded correctly, but does not match the `BlindExecutionPayloadV1` provided in `builder_getBlindExecutionPayloadV1`, the builder **MUST** return `-32003: Unknown header`.
2. Builder software **MUST** verify that `signature` is a BLS signature over `header`, verifiable using [`verify_block_signature`][verify-block-signature] and the validator public key that is expected to propose in the given slot. If the signature is determined to be invalid or from a different validator than expected, the builder **MUST** return `-32004: Invalid signature`.

[execution-payload]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#executionpayloadv1
[hash-tree-root]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#merkleization
[beacon-block-header]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblockheader
[verify-block-signature]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beacon-chain-state-transition-function
