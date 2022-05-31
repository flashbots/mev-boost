# test-cli

## Build

```
make build-cli
```

## Usage

```
./test-cli [generate|register|getHeader|getPayload] [-help]

generate [-vd-file] [-fee-recipient]
register [-vd-file] [-mev-boost] [-genesis-fork-version] [-genesis-validators-root]
getHeader [-vd-file] [-mev-boost] [-bn] [-en] [-mm] [-bellatrix-fork-version] [-genesis-validators-root]
getPayload [-vd-file] [-mev-boost] [-bn] [-en] [-mm] [-bellatrix-fork-version] [-genesis-validators-root]

Env & defaults:
	[generate]
	-gas-limit                               = 30000000
	-fee-recipient  VALIDATOR_FEE_RECIPIENT  = 0x0000000000000000000000000000000000000000

	[register]
	-vd-file                  VALIDATOR_DATA_FILE     = ./validator_data.json
	-mev-boost                MEV_BOOST_ENDPOINT      = http://127.0.0.1:18550
	-genesis-fork-version     GENESIS_FORK_VERSION    = "0x00000000" (mainnet) - network fork version
	                                                    "0x70000069" (kiln)

	[getHeader]
	-vd-file    VALIDATOR_DATA_FILE = ./validator_data.json
	-mev-boost  MEV_BOOST_ENDPOINT  = http://127.0.0.1:18550
	-bn         BEACON_ENDPOINT     = http://localhost:5052
	-en         ENGINE_ENDPOINT     = http://localhost:8545  - only used if -mm is passed or pre-bellatrix
	-mm                                                      - mergemock mode, use mergemock defaults, only call execution and use fake slot in getHeader

	-genesis-fork-version     GENESIS_FORK_VERSION    = "0x00000000" (mainnet) - network fork version
	                                                    "0x70000069" (kiln)
	[getPayload]
	-vd-file    VALIDATOR_DATA_FILE = ./validator_data.json
	-mev-boost  MEV_BOOST_ENDPOINT  = http://127.0.0.1:18550
	-bn         BEACON_ENDPOINT     = http://localhost:5052
	-en         ENGINE_ENDPOINT     = http://localhost:8545  - only used if -mm is passed or pre-bellatrix
	-mm                                                      - mergemock mode, use mergemock defaults, only call execution and use fake slot in getHeader

	-genesis-validators-root  GENESIS_VALIDATORS_ROOT = "0x0000000000000000000000000000000000000000000000000000000000000000" (mainnet) - network genesis validators root
	                                                    "0x99b09fcd43e5905236c370f184056bec6e6638cfc31a323b304fc4aa789cb4ad" (kiln)

	-genesis-fork-version     GENESIS_FORK_VERSION    = "0x00000000" (mainnet) - network fork version
	                                                    "0x70000069" (kiln)
	-bellatrix-fork-version   BELLATRIX_FORK_VERSION  = "0x02000000" (mainnet) - network bellatrix fork version
	                                                    "0x70000071" (kiln)
```

### Run mev-boost

```
./mev-boost -relays https://builder-relay-kiln.flashbots.net
```

### Run beacon node

`test-cli` needs beacon node to know current block hash (getHeader) and slot (getPayload).
If running against mergemock, pass `-mm` and disregard `-bn`.

### Pre-bellatrix run execution layer

Beacon node does not reply with block hash, it has to be taken from the execution.
If running against mergemock, `test-cli` will take the block hash from execution and fake the slot.

### Generate validator data

```
./test-cli generate [-gas-limit GAS_LIMIT] [-fee-recipient FEE_RECIPIENT]
```

If you wish you can substitute the randomly generated validator data in the validator data file.

### Register the validator

```
./test-cli register [-mev-boost]
```

### Test getHeader

```
./test-cli getHeader [-mev-boost] [-bn beacon_endpoint] [-en execution_endpoint] [-mm]
```

The call will return relay's current header.

### Test getPayload

```
./test-cli getPayload [-mev-boost] [-bn beacon_endpoint] [-en execution_endpoint] [-mm]
```

The call will return relay's best execution payload.
Signature checks on the test relay are disabled to allow ad-hoc testing.
