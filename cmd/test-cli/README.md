# test-cli

## Build

```
make build-cli
```

## Usage

```
./test-cli [generate|register|getHeader|getPayload] [-help]

generate [-vd-file] [-fee-recipient]
register [-vd-file] [-mev-boost]
getHeader [-vd-file] [-mev-boost] [-bn] [-en] [-mm]
getPayload [-vd-file] [-mev-boost] [-bn] [-en] [-mm]

Env & defaults:
	[generate]
	-gas-limit                               = 30000000
	-fee-recipient  VALIDATOR_FEE_RECIPIENT  = 0x0000000000000000000000000000000000000000

	[register]
	-vd-file   VALIDATOR_DATA_FILE = ./validator_data.json
	-mev-boost MEV_BOOST_ENDPOINT  = http://127.0.0.1:18550

	[getHeader|getPayload]
	-vd-file   VALIDATOR_DATA_FILE = ./validator_data.json
	-mev-boost MEV_BOOST_ENDPOINT  = http://127.0.0.1:18550
	-bn        BEACON_ENDPOINT     = http://localhost:5052
	-en        ENGINE_ENDPOINT     = http://localhost:8545  - only used if -mm is passed or pre-bellatrix
	-mm                                                     - mergemock mode, use mergemock defaults, only call execution and use fake slot in getHeader

	           GENESIS_VALIDATORS_ROOT = "0x0000000000000000000000000000000000000000000000000000000000000000" - per network genesis validators root
	                                     "0x99b09fcd43e5905236c370f184056bec6e6638cfc31a323b304fc4aa789cb4ad"   for kiln

	           FORK_VERSION            = "0x02000000" (Bellatrix) - per network fork version
```

## Run mev-boost

```
./mev-boost -relays https://builder-relay-kiln.flashbots.net
```

## Run beacon node

`test-cli` needs beacon node to know current block hash (getHeader) and slot (getPayload).
If running against mergemock, pass `-mm` and disregard `-bn`.

## Pre-bellatrix run execution layer

Beacon node does not reply with block hash, it has to be taken from the execution.
If running against mergemock, `test-cli` will take the block hash from execution and fake the slot.

## Generate validator data

```
./test-cli generate [-gas-limit GAS_LIMIT] [-fee-recipient FEE_RECIPIENT]
```

If you wish you can substitute the randomly generated validator data in the validator data file.

## Register the validator

```
./test-cli register [-mev-boost]
```

## Test getHeader

```
./test-cli getHeader [-mev-boost] [-bn beacon_endpoint] [-en execution_endpoint] [-mm]
```

The call will return relay's current header.

## Test getPayload

```
./test-cli getPayload [-mev-boost] [-bn beacon_endpoint] [-en execution_endpoint] [-mm]
```

The call will return relay's best execution payload.
Signature checks on the test relay are disabled to allow ad-hoc testing.
