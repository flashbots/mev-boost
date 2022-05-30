package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/sirupsen/logrus"

	"github.com/flashbots/mev-boost/server"
)

var log = logrus.WithField("service", "cmd/test-cli")

func doGenerateValidator(filePath string, gasLimit uint64, feeRecipient string) {
	v := newRandomValidator(gasLimit, feeRecipient)
	err := v.SaveValidator(filePath)
	if err != nil {
		log.WithError(err).Fatal("Could not save validator data")
	}
	log.WithField("file", filePath).Info("Saved validator data")
}

func doRegisterValidator(v validatorPrivateData, boostEndpoint string, builderSigningDomain boostTypes.Domain) {
	message, err := v.PrepareRegistrationMessage(builderSigningDomain)
	if err != nil {
		log.WithError(err).Fatal("Could not prepare registration message")
	}
	err = server.SendHTTPRequest(context.TODO(), *http.DefaultClient, http.MethodPost, boostEndpoint+"/eth/v1/builder/validators", message, nil)

	if err != nil {
		log.WithError(err).Fatal("Validator registration not successful")
	}

	log.WithError(err).Info("Registered validator")
}

func doGetHeader(v validatorPrivateData, boostEndpoint string, beaconNode Beacon, engineEndpoint string, proposerSigningDomain boostTypes.Domain) boostTypes.GetHeaderResponse {
	// Mergemock needs to call forkchoice update before getHeader, for non-mergemock beacon node this is a no-op
	err := beaconNode.onGetHeader()
	if err != nil {
		log.WithError(err).Fatal("onGetHeader hook failed")
	}

	currentBlock, err := beaconNode.getCurrentBeaconBlock()
	if err != nil {
		log.WithError(err).Fatal("could not retrieve current block hash from beacon endpoint")
	}

	var currentBlockHash string
	var emptyHash common.Hash
	// Beacon does not return block hash pre-bellatrix, fetch it from the engine if that's the case
	if currentBlock.BlockHash != emptyHash {
		currentBlockHash = currentBlock.BlockHash.String()
	} else if currentBlockHash == "" {
		block, err := getLatestEngineBlock(engineEndpoint)
		if err != nil {
			log.WithError(err).Fatal("could not get current block hash")
		}
		currentBlockHash = block.Hash.String()
	}

	uri := fmt.Sprintf("%s/eth/v1/builder/header/%d/%s/%s", boostEndpoint, currentBlock.Slot+1, currentBlockHash, v.Pk.String())

	var getHeaderResp boostTypes.GetHeaderResponse
	if err := server.SendHTTPRequest(context.TODO(), *http.DefaultClient, http.MethodGet, uri, nil, &getHeaderResp); err != nil {
		log.WithError(err).WithField("currentBlockHash", currentBlockHash).Fatal("Could not get header")
	}

	if getHeaderResp.Data.Message == nil {
		log.Fatal("Did not receive correct header")
	}
	log.WithField("header", *getHeaderResp.Data.Message).Info("Got header from boost")

	ok, err := boostTypes.VerifySignature(getHeaderResp.Data.Message, proposerSigningDomain, getHeaderResp.Data.Message.Pubkey[:], getHeaderResp.Data.Signature[:])
	if err != nil {
		log.WithError(err).Fatal("Could not verify builder bid signature")
	}
	if !ok {
		log.Fatal("Incorrect builder bid signature")
	}

	return getHeaderResp
}

func doGetPayload(v validatorPrivateData, boostEndpoint string, beaconNode Beacon, engineEndpoint string, proposerSigningDomain boostTypes.Domain) {
	header := doGetHeader(v, boostEndpoint, beaconNode, engineEndpoint, proposerSigningDomain)

	blindedBeaconBlock := boostTypes.BlindedBeaconBlock{
		Slot:          0,
		ProposerIndex: 0,
		ParentRoot:    boostTypes.Root{},
		StateRoot:     boostTypes.Root{},
		Body: &boostTypes.BlindedBeaconBlockBody{
			RandaoReveal:           boostTypes.Signature{},
			Eth1Data:               &boostTypes.Eth1Data{},
			Graffiti:               boostTypes.Hash{},
			ProposerSlashings:      []*boostTypes.ProposerSlashing{},
			AttesterSlashings:      []*boostTypes.AttesterSlashing{},
			Attestations:           []*boostTypes.Attestation{},
			Deposits:               []*boostTypes.Deposit{},
			VoluntaryExits:         []*boostTypes.VoluntaryExit{},
			SyncAggregate:          &boostTypes.SyncAggregate{},
			ExecutionPayloadHeader: header.Data.Message.Header,
		},
	}

	signature, err := v.Sign(&blindedBeaconBlock, proposerSigningDomain)
	if err != nil {
		log.WithError(err).Fatal("could not sign blinded beacon block")
	}

	payload := boostTypes.SignedBlindedBeaconBlock{
		Message:   &blindedBeaconBlock,
		Signature: signature,
	}
	var respPayload boostTypes.GetPayloadResponse
	if err := server.SendHTTPRequest(context.TODO(), *http.DefaultClient, http.MethodPost, boostEndpoint+"/eth/v1/builder/blinded_blocks", payload, &respPayload); err != nil {
		log.WithError(err).Fatal("could not get payload")
	}

	if respPayload.Data == nil {
		log.Fatal("Did not receive correct payload")
	}
	log.WithField("payload", *respPayload.Data).Info("got payload from mev-boost")
}

func main() {
	generateCommand := flag.NewFlagSet("generate", flag.ExitOnError)
	registerCommand := flag.NewFlagSet("register", flag.ExitOnError)
	getHeaderCommand := flag.NewFlagSet("getHeader", flag.ExitOnError)
	getPayloadCommand := flag.NewFlagSet("getPayload", flag.ExitOnError)

	var validatorDataFile string
	envValidatorDataFile := getEnv("VALIDATOR_DATA_FILE", "./validator_data.json")
	generateCommand.StringVar(&validatorDataFile, "vd-file", envValidatorDataFile, "Path to validator data file")
	registerCommand.StringVar(&validatorDataFile, "vd-file", envValidatorDataFile, "Path to validator data file")
	getHeaderCommand.StringVar(&validatorDataFile, "vd-file", envValidatorDataFile, "Path to validator data file")
	getPayloadCommand.StringVar(&validatorDataFile, "vd-file", envValidatorDataFile, "Path to validator data file")

	var boostEndpoint string
	envBoostEndpoint := getEnv("MEV_BOOST_ENDPOINT", "http://127.0.0.1:18550")
	registerCommand.StringVar(&boostEndpoint, "mev-boost", envBoostEndpoint, "Mev boost endpoint")
	getHeaderCommand.StringVar(&boostEndpoint, "mev-boost", envBoostEndpoint, "Mev boost endpoint")
	getPayloadCommand.StringVar(&boostEndpoint, "mev-boost", envBoostEndpoint, "Mev boost endpoint")

	var beaconEndpoint string
	envBeaconEndpoint := getEnv("BEACON_ENDPOINT", "http://localhost:5052")
	getHeaderCommand.StringVar(&beaconEndpoint, "bn", envBeaconEndpoint, "Beacon node endpoint")
	getPayloadCommand.StringVar(&beaconEndpoint, "bn", envBeaconEndpoint, "Beacon node endpoint")

	var isMergemock bool
	getHeaderCommand.BoolVar(&isMergemock, "mm", false, "Use mergemock EL to fake beacon node")
	getPayloadCommand.BoolVar(&isMergemock, "mm", false, "Use mergemock EL to fake beacon node")

	var engineEndpoint string
	envEngineEndpoint := getEnv("ENGINE_ENDPOINT", "http://localhost:8545")
	getHeaderCommand.StringVar(&engineEndpoint, "en", envEngineEndpoint, "Engine endpoint")
	getPayloadCommand.StringVar(&engineEndpoint, "en", envEngineEndpoint, "Engine endpoint")

	var genesisValidatorsRootStr string
	envGenesisValidatorsRoot := getEnv("GENESIS_VALIDATORS_ROOT", "0x0000000000000000000000000000000000000000000000000000000000000000")
	getHeaderCommand.StringVar(&genesisValidatorsRootStr, "genesis-validators-root", envGenesisValidatorsRoot, "Root of genesis validators")
	getPayloadCommand.StringVar(&genesisValidatorsRootStr, "genesis-validators-root", envGenesisValidatorsRoot, "Root of genesis validators")

	var genesisForkVersionStr string
	envGenesisForkVersion := getEnv("GENESIS_FORK_VERSION", "0x00000000")
	registerCommand.StringVar(&genesisForkVersionStr, "genesis-fork-version", envGenesisForkVersion, "hex encoded genesis fork version")

	var bellatrixForkVersionStr string
	envBellatrixForkVersion := getEnv("BELLATRIX_FORK_VERSION", "0x02000000")
	getHeaderCommand.StringVar(&bellatrixForkVersionStr, "bellatrix-fork-version", envBellatrixForkVersion, "hex encoded bellatrix fork version")
	getPayloadCommand.StringVar(&bellatrixForkVersionStr, "bellatrix-fork-version", envBellatrixForkVersion, "hex encoded bellatrix fork version")

	var gasLimit uint64
	envGasLimitStr := getEnv("VALIDATOR_GAS_LIMIT", "30000000")
	envGasLimit, err := strconv.Atoi(envGasLimitStr)
	if err != nil {
		log.WithError(err).Fatal("invalid gas limit specified")
	}
	generateCommand.Uint64Var(&gasLimit, "gas-limit", uint64(envGasLimit), "Gas limit to register the validator with")

	var validatorFeeRecipient string
	envValidatorFeeRecipient := getEnv("VALIDATOR_FEE_RECIPIENT", "0x0000000000000000000000000000000000000000")
	generateCommand.StringVar(&validatorFeeRecipient, "feeRecipient", envValidatorFeeRecipient, "FeeRecipient to register the validator with")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [generate|register|getHeader|getPayload]:\n", os.Args[0])
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		generateCommand.Parse(os.Args[2:])
		doGenerateValidator(validatorDataFile, gasLimit, validatorFeeRecipient)
	case "register":
		registerCommand.Parse(os.Args[2:])
		builderSigningDomain := computeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersionStr, boostTypes.Root{}.String())
		doRegisterValidator(mustLoadValidator(validatorDataFile), boostEndpoint, builderSigningDomain)
	case "getHeader":
		getHeaderCommand.Parse(os.Args[2:])
		proposerSigningDomain := computeDomain(boostTypes.DomainTypeBeaconProposer, bellatrixForkVersionStr, genesisValidatorsRootStr)
		doGetHeader(mustLoadValidator(validatorDataFile), boostEndpoint, createBeacon(isMergemock, beaconEndpoint, engineEndpoint), engineEndpoint, proposerSigningDomain)
	case "getPayload":
		getPayloadCommand.Parse(os.Args[2:])
		proposerSigningDomain := computeDomain(boostTypes.DomainTypeBeaconProposer, bellatrixForkVersionStr, genesisValidatorsRootStr)
		doGetPayload(mustLoadValidator(validatorDataFile), boostEndpoint, createBeacon(isMergemock, beaconEndpoint, engineEndpoint), engineEndpoint, proposerSigningDomain)
	default:
		fmt.Println("Expected generate|register|getHeader|getPayload subcommand")
		os.Exit(1)
	}
}

func computeDomain(domainType boostTypes.DomainType, forkVersionHex string, genesisValidatorsRootHex string) boostTypes.Domain {
	genesisValidatorsRoot := boostTypes.Root(common.HexToHash(genesisValidatorsRootHex))
	forkVersionBytes, err := hexutil.Decode(forkVersionHex)
	if err != nil || len(forkVersionBytes) > 4 {
		fmt.Println("Invalid fork version passed")
		os.Exit(1)
	}
	var forkVersion [4]byte
	copy(forkVersion[:], forkVersionBytes[:4])
	return boostTypes.ComputeDomain(domainType, forkVersion, genesisValidatorsRoot)
}

func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
