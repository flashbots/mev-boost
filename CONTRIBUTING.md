# Contributing guide

Welcome to the Flashbots collective! We just ask you to be nice when you play with us.

Please start by reading our [code of conduct](CODE_OF_CONDUCT.md).

## Set up

Install a few dev dependencies for `make lint`:

```bash
go install github.com/mgechev/revive@v1.1.3
go install mvdan.cc/gofumpt@v0.3.1
go install honnef.co/go/tools/cmd/staticcheck@v0.3.0
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0
```

Look at the [README for instructions to install the dependencies and build `mev-boost`](README.md#installing)

Alternatively, run mev-boost without build step:

```bash
go run . -h

# Run mev-boost pointed at our Kiln builder+relay
go run . -kiln -relays https://0xb5246e299aeb782fbc7c91b41b3284245b1ed5206134b0028b81dfb974e5900616c67847c2354479934fc4bb75519ee1@builder-relay-kiln.flashbots.net
```

Note that you'll need to set the correct genesis fork version (either manually with `-genesis-fork-version` or a helper flag `-mainnet`/`-kiln`/`-ropsten`/`-sepolia`).

If the test or target application crashes with an "illegal instruction" exception, run/rebuild with CGO_CFLAGS environment variable set to `-O -D__BLST_PORTABLE__`. This error also happens if you are on an ARM-based system, including the Apple M1/M2 chip.

## Test

```bash
make test
make lint
make run-mergemock-integration
```

### Testing with test-cli

test-cli is a utility to run through all the proposer requests against mev-boost+relay. See also the [test-cli readme](cmd/test-cli/README.md).

### Testing with mergemock

Mergemock is fully integrated: https://github.com/protolambda/mergemock

Make sure you've setup and built mergemock first, refer to its [README](https://github.com/flashbots/mergemock#quick-start) but here's a quick setup guide:

```bash
git clone https://github.com/protolambda/mergemock.git
cd mergemock
go build . mergemock
wget https://gist.githubusercontent.com/lightclient/799c727e826483a2804fc5013d0d3e3d/raw/2e8824fa8d9d9b040f351b86b75c66868fb9b115/genesis.json
openssl rand -hex 32 | tr -d "\n" > jwt.hex
```

Then you can run an integration test with mergemock, spawning both a mergemock relay+execution engine and a mergemock consensus client pointing to mev-boost, which in turn points to the mergemock relay:

```bash
cd mev-boost
make run-mergemock-integration
```

The path to the mergemock repo is assumed to be `../mergemock`, you can override like so:

```bash
make MERGEMOCK_DIR=/PATH-TO-MERGEMOCK-REPO run-mergemock-integration
```

to run mergemock in dev mode:

```bash
make MERGEMOCK_BIN='go run .' run-mergemock-integration
```

## Code style

Start by making sure that your code is readable, consistent, and pretty.
Follow the [Clean Code](https://flashbots.notion.site/Clean-Code-13016c5c7ca649fba31ae19d797d7304) recommendations.

## Send a pull request

- Your proposed changes should be first described and discussed in an issue.
- Open the branch in a personal fork, not in the team repository.
- Every pull request should be small and represent a single change. If the problem is complicated, split it in multiple issues and pull requests.
- Every pull request should be covered by unit tests.

We appreciate you, friend <3.

---

For the checklist and guide to releasing a new version, see [RELEASE.md](RELEASE.md).