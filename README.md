# mev-boost

A middleware server written in Go, that sits between an ethereum PoS consensus client and an execution client. It allows consensus clients to outsource block construction to third party block builders as well as fallback to execution clients. See [ethresearch post](https://ethresear.ch/t/mev-boost-merge-ready-flashbots-architecture/11177/) for the high level architecture.

![block-proposal](/docs/block-proposal.png)

## Implementation Milestones

1. **[minimal viable kintsugi compatibility](./docs/milestone-1.md) (mid december 2021)**
simple middleware logic with minimal consensus client changes, simple networking, no authentication, and manual safety mechanism
3. **[security, authentication & reputation](./docs/milestone-2.md) (end of january 2021)**
security features necessary for production use of the software, see [mev-boost milestone 2 security features](https://hackmd.io/@flashbots/r1CuVl86Y)
5. **[p2p communications](./docs/milestone-3.md) (optional / tbd)**
p2p comms mechanisms for validator IP obfuscation
7. **[optional configurations](./docs/milestone-4.md) (optional / tbd)**
optional configurations for custom distributions

| Milestone  |  1  |  2  |  3  |  4  |
| ---------- |:---:|:---:|:---:|:---:|
| mev-boost  | ✅  |     |     |     |
| mergemock  | ✅  |     |     |     |
| lighthouse | ✅  |     |     |     |
| prysm      | ✅  |     |     |     |
| teku       |     |     |     |     |
| nimbus     |     |     |     |     |
| lodestar   |     |     |     |     |
| grandine   |     |     |     |     |

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
