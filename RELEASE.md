# Releasing a new version of mev-boost

This is a guide on how to release a new version of mev-boost.

## Double-check the current status

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
* For possible reviewers, take a look at [recent contributors](https://github.com/flashbots/mev-boost/graphs/contributors)

## Release only with 4 eyes

* Always have two people preparing and publishing the final release

## Tagging a version and pushing the release

To create a new version (with tag), follow all these steps! They are necessary to have the correct build version inside, and work with `go install`.

* Update `Version` in `config/vars.go` - change it to the next version (eg. from `v2.3.1-dev` to `v2.3.1`), and commit
* Create a [signed git tag](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-tags): `git tag -s v2.3.1`
* Now push to main and push the tag: `git push && git push --tags`
* Update `Version` in `config/vars.go` to next patch with `dev` suffix (eg. `v2.3.2-dev`), and commit

Now check the Github CI actions for release activity: https://github.com/flashbots/mev-boost/actions - CI builds and pushes the Docker image, and prepares a new draft release in https://github.com/flashbots/mev-boost/releases. Open it, generate the description, review, add signoffs and testing, and publish.
