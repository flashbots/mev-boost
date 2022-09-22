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

# Releasing a new version of mev-boost

First of all, check that the git repository is in the final state, and all the tests and checks are running fine

Test the current

```bash
make lint
make test-race
go mod tidy
git status # should be no changes

# Start mev-boost with relay check, and call the mev-boost status endpoint
go run . -mainnet -relay-check -relays https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net,https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com,https://0xb3ee7afcf27f1f1259ac1787876318c6584ee353097a50ed84f51a1f21a323b3736f271a895c7ce918c038e4265918be@relay.edennetwork.io,https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@builder-relay-mainnet.blocknative.com -debug
curl localhost:18550/eth/v1/builder/status
```

## Prepare a release candidate build and Docker image

For example, creating a new release `v2.3.1-rc1`:

```bash
# create a new branch
git checkout -b v2.3.1-rc1

# set and commit the correct version as described below
...

# be careful when pushing the tag to Github, because Docker CI would publish this tag as :latest (TODO: make docker ci ignore -* version suffix)
...

# create and push the Docker image
make docker-image-portable
make docker-push-version

# now other parties can test the release candidate from Docker like this:
docker pull flashbots/mev-boost:v2.3.1-rc1
```

## Ask node operators to test this RC (on Goerli or Sepolia)

* Reach out to node operators to help test this release
* Collect their sign-off for the release

## Collect code signoffs

* Reach out to the parties that have reviewed the PRs and ask for a sign-off on the release

## Release only with 4 eyes

* Always have two people preparing and publishing the final release

## Tagging a version and pushing the release

To create a new version (with tag), follow all these steps! They are necessary to have the correct build version inside, and work with `go install`.

* Update `Version` in `config/vars.go` - change it to the next version (eg. from `v2.3.1-dev` to `v2.3.1`), and commit
* Create a git tag: `git tag v2.3.1`
* Now push to main and push the tag: `git push && git push --tags`
* Update `Version` in `config/vars.go` to next patch with `dev` suffix (eg. `v2.3.2-dev`), and commit

Now check the Github CI actions for release activity: https://github.com/flashbots/mev-boost/actions - CI builds and pushes the Docker image, and prepares a new draft release in https://github.com/flashbots/mev-boost/releases. Open it, generate the description, review, add signoffs and testing, and publish.
