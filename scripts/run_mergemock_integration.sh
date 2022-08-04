#!/bin/bash
set -euo pipefail

PROJECT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)

#
# Default to these values for mergemock.
#
MERGEMOCK_DIR=${MERGEMOCK_DIR:-$PROJECT_DIR/../mergemock}
MERGEMOCK_BIN=${MERGEMOCK_BIN:-./mergemock}

#
# This function will ensure there are no lingering processes afterwards.
#
trap 'kill -9 $(jobs -p)' exit

#
# Start mev-boost.
#
$PROJECT_DIR/mev-boost -mainnet -relays http://0x821961b64d99b997c934c22b4fd6109790acf00f7969322c4e9dbf1ca278c333148284c01c5ef551a1536ddd14b178b9@127.0.0.1:28545 &
echo "Waiting for mev-boost to become available..."
while ! nc -z localhost 18550; do
  sleep 0.1
done

#
# Mock the relay.
#
pushd $MERGEMOCK_DIR >/dev/null
$MERGEMOCK_BIN relay --listen-addr 127.0.0.1:28545 --secret-key 1e64a14cb06073c2d7c8b0b891e5dc3dc719b86e5bf4c131ddbaa115f09f8f52 &
echo "Waiting for relay to become available..."
while ! nc -z localhost 28545; do
  sleep 0.1
done

#
# Mock the consensus.
#
$MERGEMOCK_BIN consensus --slot-time=4s --engine http://127.0.0.1:8551 --builder http://127.0.0.1:18550 --slot-bound 10 &
MERGEMOCK_CONSENSUS_PID=$!
popd >/dev/null

#
# The script will exit when this process finishes.
#
wait $MERGEMOCK_CONSENSUS_PID
