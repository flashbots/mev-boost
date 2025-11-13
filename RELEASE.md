# Releasing a new version of mev-boost

This is a guide on how to release a new version of mev-boost.

The best days to release a new version are Monday to Wednesday. Never release on a Friday. Release preferred with another person present (four eyes principle).

Process:
1. Double-check the current build
2. Create a release candidate (RC)
3. Test the RC on testnets
4. Collect signoffs
5. Create the full release + announcement

## Double-check the current build

First of all, check that the git repository is in the final state, and all the tests and checks are running fine

Test the current

```bash
make lint
make test-race
go mod tidy
git status # should be no changes

# Start mev-boost with relay check and -relays
go run . -mainnet -relay-check -min-bid 0.12345 -debug -relays https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net,https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com,https://0xa1559ace749633b997cb3fdacffb890aeebdb0f5a3b6aaa7eeeaf1a38af0a8fe88b9e4b1f61f236d2e64d95733327a62@relay.ultrasound.money

# Start mev-boost with relay check and multiple -relay flags
go run . -mainnet -relay-check -debug -min-bid 0.12345 \
    -relay https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net \
    -relay https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com \
    -relay https://0xa1559ace749633b997cb3fdacffb890aeebdb0f5a3b6aaa7eeeaf1a38af0a8fe88b9e4b1f61f236d2e64d95733327a62@relay.ultrasound.money

# Call the status endpoint
curl localhost:18550/eth/v1/builder/status
```

## Make a release

Let's release `v1.9`:

1. Create a GitHub issue about the upcoming release ([example](https://github.com/flashbots/mev-boost/issues/524))
1. Tag a release candidate (RC), or alpha version: `v1.9-rc1`, and push the new tag to GitHub.
    - The [release CI](https://github.com/flashbots/mev-boost/actions) starts
building the binaries and the Docker image (which is then pushed to [Docker Hub](https://hub.docker.com/r/flashbots/mev-boost/tags)).
    - Binaries can be downloaded from the release CI summary website.
1. Test the RC in testnets. Iterate as needed, create more alpha versions / release candidates as needed.
2. When tests are complete, create the release: tag, GitHub release, `stable` branch update.

```bash
# Create the RC tag
git tag -s v1.9-rc1

# Push to Github, which kicks off the release CI
git push origin tags/v1.9-rc1

# When CI is done, the Docker image can be used
docker pull flashbots/mev-boost:v1.9a1

# Once testing is done, tag the full release
git tag -s v1.9
git tag -s v1.9.0

# Push to Github, which kicks off the release CI
git push origin tags/v1.9 tags/v1.9.0

# Finally, update the stable branch
git checkout stable
git merge tags/v1.9 --ff-only
```
