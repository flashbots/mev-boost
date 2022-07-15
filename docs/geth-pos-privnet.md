# Running PoS private testnet with geth

It’s surprisingly complex to set up, and there are little up-to-date resources. This guide may be incomplete, but should cover most of it. 
If you have any improvements, please create a PR 🙏

Resources used: [https://medium.com/@pradeep_thomas/how-to-setup-your-own-private-ethereum-network-f80bc6aea088](https://medium.com/@pradeep_thomas/how-to-setup-your-own-private-ethereum-network-f80bc6aea088), https://github.com/skylenet/ethereum-genesis-generator, [https://notes.ethereum.org/@parithosh/H1MSKgm3F](https://notes.ethereum.org/@parithosh/H1MSKgm3F), [https://dev.to/q9/how-to-run-your-own-beacon-chain-e70](https://dev.to/q9/how-to-run-your-own-beacon-chain-e70), [https://lighthouse-book.sigmaprime.io/setup.html](https://lighthouse-book.sigmaprime.io/setup.html) ([https://github.com/sigp/lighthouse/tree/unstable/scripts/local_testnet](https://github.com/sigp/lighthouse/tree/unstable/scripts/local_testnet))

Special thanks to Mateusz ([@mmrosum](https://twitter.com/mmrosum)) who put the initial version of this document together.

### No tl;dr, should take around 30 minutes to get it right

### Prepare PoW chain
you will also need to install `go-ethereum` to run the PoW chain.
1. Clone https://github.com/skylenet/ethereum-genesis-generator
2. `cp -r config-example data`
3. Modify data/el/genesis-config.yaml:
    1. Set chain id to `4242`
    2. Remove clique altogether or set `enabled: false`
4. Run `docker run -it -u $UID -v $PWD/data:/data -p 127.0.0.1:8000:8000 skylenet/ethereum-genesis-generator:latest el`
5. Check the genesis file in data/el/geth.json - verify clique is not present.  it is probably generated anyways - delete the corresponding JSON entry.
6. Generate a valid account using your favorite tool, take note of the address.  here are 3 options:
    1. `ethkey generate` + geth’s —nodekey
    2. geth console + eth.newAccount
    3. create a new address in something like metamask, and copy the private key.  inside a geth console session (`geth --datadir ~/.ethereum/local-testnet/testnet/geth-node-1 console`), run `web3.personal.importRawKey("<Private Key>","<Password>")`
7. Initialize geth from the json: `geth init --datadir ~/.ethereum/local-testnet/testnet/geth-node-1 ~/path/to/ethereum-genesis-generator/data/el/geth.json`
8. Check the node starts to mine and kill it quickly
You only have 100 blocks until fork is enabled and 400 blocks until node stops mining
`geth --datadir ~/.ethereum/local-testnet/testnet/geth-node-1 --networkid 4242 --http --http.port 8545 --http.api personal,eth,net,web3,engine,debug --discovery.dns "" --port 30303 --mine --miner.etherbase=<address> --miner.threads 1 --miner.gaslimit 1000000000 --authrpc.jwtsecret ~/.ethereum/local-testnet/testnet/geth-node-1/geth/jwtsecret --authrpc.addr localhost --authrpc.port 8551 --authrpc.vhosts localhost --unlock "<address>" --password <(echo "<password>") --allow-insecure-unlock > ~/.ethereum/miner.log 2>&1`
where `<address>` is the public key of the wallet you created in step (6)

### Prepare PoS chain

1. Clone https://github.com/sigp/lighthouse
2. Go to `scripts/local_testnet`
    1. Modify vars.env:
        1. Set `ETH1_NETWORK_MNEMONIC`, `DEPOSIT_CONTRACT_ADDRESS`, `GENESIS_FORK_VERSION` to be the same as in PoW’s config (GENESIS_FORK_VERSION is in `ethereum-genesis-generator/data/cl/config.yaml`)
        2. Set `GENESIS_DELAY` to 30
        3. Set `ALTAIR_FORK_EPOCH` to 1
        4. Add `MERGE_FORK_EPOCH=1`
        5. Adjust `SECONDS_PER_SLOT`, `SECONDS_PER_ETH1_BLOCK`, `BN_COUNT` to your preference
            1. There seem to be some issues with multiple beacon nodes using the same EL, if it does not work set `BN_COUNT` to 1
        6. Do not change `VALIDATOR_COUNT`, `GENESIS_VALIDATOR_COUNT` to less than 64
        7. modify VC_ARGS line to `VC_ARGS="--suggested-fee-recipient <address>"`, where <address> is the same public key that you registered in the `geth` command #8 above
    2. Modify scripts/local_testnet/beacon_node.sh:
        1. Add merge options to the end of the `exec lighthouse` command at the bottom: `--eth1 --merge --terminal-total-difficulty-override=60000000 --eth1-endpoints http://127.0.0.1:8545/ --execution-endpoints http://127.0.0.1:8551/ --http-allow-sync-stalled --execution-jwt ~/.ethereum/local-testnet/testnet/geth-node-1/geth/jwtsecret` .  (don't forget to add `\` between newlines, if any.  confirm that the jwtsecret path is the one used by `geth` - you may need to expand the `~`)
        2. Allow all subnets, with the line `SUBSCRIBE_ALL_SUBNETS="--subscribe-all-subnets"`
    3. Modify scripts/local_testnet/setup.sh:
        1. Add `--merge-fork-epoch $MERGE_FORK_EPOCH`
    4. Modify start_local_testnet.sh:
        1. Remove/comment ganache [`https://github.com/sigp/lighthouse/blob/stable/scripts/local_testnet/start_local_testnet.sh#L93`](https://github.com/sigp/lighthouse/blob/stable/scripts/local_testnet/start_local_testnet.sh#L93)
3. install lighthouse and lcli:
    1. make
    2. make install-lcli  

### Run and hope for the best

1. Run geth, wait until you see blocks in the logs
2. Run lighthouse’s `start_local_testnet.sh`. If fails make sure to read the log. If it complains about lack of funds check that geth mines blocks and retry.
3. Monitor the logs, PoS will kick in once `mergeForkBlock` is reached but you should see beacon and validator nodes being active before that (just not finalizing any epochs and not producing blocks and not voting before `mergeForkBlock`)
4. Once geth reaches `terminal_total_difficulty` it stops mining eth1 blocks (`60000000` ~12 min) and should be used by beacon nodes to create PoS payloads.
