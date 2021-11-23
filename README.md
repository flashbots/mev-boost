# mev-boost

A middleware server written in Go, that sits between an ethereum PoS consensus client and an execution client. It allows consensus clients to outsource block construction to third party block builders as well as fallback to execution clients. See [ethresearch post](https://ethresear.ch/t/mev-boost-merge-ready-flashbots-architecture/11177/) for the high level architecture.

## Implementation Plan

### milestone 1 - kintsugi testnet

simple middleware logic with minimal consensus client changes, simple networking, no authentication, and manual safety mechanism

- [x] middleware sends `feeRecipient` to relay with direct `engine_forkchoiceUpdatedV1` request at beginning of block
- [x] middleware fetches signed payloads from relay using unauthenticated `getPayloadHeader` request
- [x] middleware selects best payload that matches expected `payloadId` and requests signature from consensus client, this requires passing header object to the consensus client and flagging that it should be returned to the middleware once signed
- [x] middleware returns signed block + initial payload header to relay with direct request

### milestone 2 - authentication & privacy

add authentication and p2p comms mechanisms

- [ ] middleware imports staking keys locally (dangerous)
- [ ] middleware signs `feeRecipient` message and gossips over p2p at regular interval
- [ ] middleware gossips signed block + initial payload header over p2p
- [ ] consider adding direct `engine_forkchoiceUpdatedV1` call to relay and returning full payload directly to validator as optimization

### milestone 3 - security & reputation

cleanup consensus client and add security fallbacks

- [ ] add signing domain to VC to avoid need to export keys to middleware
- [ ] add middleware bypass mechanism to BN as a fallback
- [ ] add middleware module for verifying previous relay payload validity and accuracy with hard or statistical blacklist (may require modifications to execution client)
- [ ] add middleware module for relaycops to allow outsourcing of relay monitoring to trusted 3rd party
- [ ] consider adding merkle proof of payment to shift verification requirements to the relay

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
