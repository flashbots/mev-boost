package types

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ssz "github.com/ferranbt/fastssz"
	"github.com/stretchr/testify/require"
)

type _SSZable interface {
	MarshalSSZ() ([]byte, error)
	MarshalSSZTo(buf []byte) (dst []byte, err error)
	UnmarshalSSZ(buf []byte) error
	SizeSSZ() (size int)
	HashTreeRoot() ([32]byte, error)
	HashTreeRootWith(hh *ssz.Hasher) (err error)
}

func TestExecutionPayloadHeader(t *testing.T) {
	baseFeePerGas := U256Str{}
	baseFeePerGas[31] = 0x08

	h := ExecutionPayloadHeader{
		ParentHash:       Hash{0x01},
		FeeRecipient:     Address{0x02},
		StateRoot:        Root{0x03},
		ReceiptsRoot:     Root{0x04},
		LogsBloom:        Bloom{0x05},
		Random:           Hash{0x06},
		BlockNumber:      5001,
		GasLimit:         5002,
		GasUsed:          5003,
		Timestamp:        5004,
		ExtraData:        hexutil.Bytes{0x07},
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        Hash{0x09},
		TransactionsRoot: Root{0x0a},
	}
	b, err := json.Marshal(h)
	require.NoError(t, err)

	expectedJSON := `{
        "parent_hash": "0x0100000000000000000000000000000000000000000000000000000000000000",
        "fee_recipient": "0x0200000000000000000000000000000000000000",
        "state_root": "0x0300000000000000000000000000000000000000000000000000000000000000",
        "receipts_root": "0x0400000000000000000000000000000000000000000000000000000000000000",
        "logs_bloom": "0x05000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "prev_randao": "0x0600000000000000000000000000000000000000000000000000000000000000",
        "block_number": "5001",
        "gas_limit": "5002",
        "gas_used": "5003",
        "timestamp": "5004",
        "extra_data": "0x07",
        "base_fee_per_gas": "8",
        "block_hash": "0x0900000000000000000000000000000000000000000000000000000000000000",
        "transactions_root": "0x0a00000000000000000000000000000000000000000000000000000000000000"
    }`
	require.JSONEq(t, expectedJSON, string(b))

	// Now unmarshal it back and compare to original
	h2 := new(ExecutionPayloadHeader)
	err = json.Unmarshal(b, h2)
	require.NoError(t, err)
	require.Equal(t, h.ParentHash, h2.ParentHash)

	p, err := h2.HashTreeRoot()
	require.NoError(t, err)
	rootHex := fmt.Sprintf("%x", p)
	require.Equal(t, "7b7fd346d2b66aab2efce23959d7f90f36ff31a944ba867ae1c2827f85b2fbe5", rootHex)
}

func TestBlindedBeaconBlock(t *testing.T) {
	parentHash := Hash{0xa1}
	blockHash := Hash{0xa1}
	feeRecipient := Address{0xb1}

	msg := &BlindedBeaconBlock{
		Slot:          1,
		ProposerIndex: 2,
		ParentRoot:    Root{0x03},
		StateRoot:     Root{0x04},
		Body: &BlindedBeaconBlockBody{
			Eth1Data: &Eth1Data{
				DepositRoot:  Root{0x05},
				DepositCount: 5,
				BlockHash:    Hash{0x06},
			},
			ProposerSlashings: []*ProposerSlashing{},
			AttesterSlashings: []*AttesterSlashing{},
			Attestations:      []*Attestation{},
			Deposits:          []*Deposit{},
			VoluntaryExits:    []*VoluntaryExit{},
			SyncAggregate:     &SyncAggregate{CommitteeBits{0x07}, Signature{0x08}},
			ExecutionPayloadHeader: &ExecutionPayloadHeader{
				ParentHash:       parentHash,
				FeeRecipient:     feeRecipient,
				StateRoot:        Root{0x09},
				ReceiptsRoot:     Root{0x0a},
				LogsBloom:        Bloom{0x0b},
				Random:           Hash{0x0c},
				BlockNumber:      5001,
				GasLimit:         5002,
				GasUsed:          5003,
				Timestamp:        5004,
				ExtraData:        hexutil.Bytes{0x0d},
				BaseFeePerGas:    IntToU256(123456789),
				BlockHash:        blockHash,
				TransactionsRoot: Root{0x0e},
			},
		},
	}

	// Get HashTreeRoot
	root, err := msg.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, "b3b6844756cbf0fdd996cb20a1439bfb59a640cdae1604dbd8a81c7c993a6a6b", fmt.Sprintf("%x", root))

	// Marshalling
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	// fmt.Println(string(b))

	expectedJSON := `{
        "slot": "1",
        "proposer_index": "2",
        "parent_root": "0x0300000000000000000000000000000000000000000000000000000000000000",
        "state_root": "0x0400000000000000000000000000000000000000000000000000000000000000",
        "body": {
            "randao_reveal": "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
            "eth1_data": {
                "deposit_root": "0x0500000000000000000000000000000000000000000000000000000000000000",
                "deposit_count": "5",
                "block_hash": "0x0600000000000000000000000000000000000000000000000000000000000000"
            },
            "graffiti": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "proposer_slashings": [],
            "attester_slashings": [],
            "attestations": [],
            "deposits": [],
            "voluntary_exits": [],
            "sync_aggregate": {
                "sync_committee_bits": "0x07000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
                "sync_committee_signature": "0x080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
            },
            "execution_payload_header": {
                "parent_hash": "0xa100000000000000000000000000000000000000000000000000000000000000",
                "fee_recipient": "0xb100000000000000000000000000000000000000",
                "state_root": "0x0900000000000000000000000000000000000000000000000000000000000000",
                "receipts_root": "0x0a00000000000000000000000000000000000000000000000000000000000000",
                "logs_bloom": "0x0b000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
                "prev_randao": "0x0c00000000000000000000000000000000000000000000000000000000000000",
                "block_number": "5001",
                "gas_limit": "5002",
                "gas_used": "5003",
                "timestamp": "5004",
                "extra_data": "0x0d",
                "base_fee_per_gas": "123456789",
                "block_hash": "0xa100000000000000000000000000000000000000000000000000000000000000",
                "transactions_root": "0x0e00000000000000000000000000000000000000000000000000000000000000"
            }
        }
    }`
	require.JSONEq(t, expectedJSON, string(b))

	// Now unmarshal it back and compare to original
	msg2 := new(BlindedBeaconBlock)
	err = json.Unmarshal(b, msg2)
	require.NoError(t, err)
	require.Equal(t, msg, msg2)

	// HashTreeRoot
	p, err := msg2.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, "b3b6844756cbf0fdd996cb20a1439bfb59a640cdae1604dbd8a81c7c993a6a6b", fmt.Sprintf("%x", p))
}

func TestExecutionPayload(t *testing.T) {
	parentHash := Hash{0xa1}
	blockHash := Hash{0xa1}
	feeRecipient := Address{0xb1}

	tx1hex := "0xcdc2b165e82ed1fe09aae28fccee2199946baf6b4503ca7e6f19aaa95a92b766dce6d968024a68d97ee178082928142430d4"
	tx1 := new(hexutil.Bytes)
	tx1.UnmarshalText([]byte(tx1hex))

	msg := &ExecutionPayload{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     Root{0x09},
		ReceiptsRoot:  Root{0x0a},
		LogsBloom:     Bloom{0x0b},
		Random:        Hash{0x0c},
		BlockNumber:   5001,
		GasLimit:      5002,
		GasUsed:       5003,
		Timestamp:     5004,
		ExtraData:     hexutil.Bytes{0x0d},
		BaseFeePerGas: IntToU256(123456789),
		BlockHash:     blockHash,
		Transactions:  []hexutil.Bytes{*tx1},
	}

	// Marshalling
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	fmt.Println(string(b))

	expectedJSON := `{
        "parent_hash": "0xa100000000000000000000000000000000000000000000000000000000000000",
        "fee_recipient": "0xb100000000000000000000000000000000000000",
        "state_root": "0x0900000000000000000000000000000000000000000000000000000000000000",
        "receipts_root": "0x0a00000000000000000000000000000000000000000000000000000000000000",
        "logs_bloom": "0x0b000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "prev_randao": "0x0c00000000000000000000000000000000000000000000000000000000000000",
        "block_number": "5001",
        "gas_limit": "5002",
        "gas_used": "5003",
        "timestamp": "5004",
        "extra_data": "0x0d",
        "base_fee_per_gas": "123456789",
        "block_hash": "0xa100000000000000000000000000000000000000000000000000000000000000",
        "transactions": [
            "0xcdc2b165e82ed1fe09aae28fccee2199946baf6b4503ca7e6f19aaa95a92b766dce6d968024a68d97ee178082928142430d4"
        ]
    }`
	require.JSONEq(t, expectedJSON, string(b))

	// Now unmarshal it back and compare to original
	msg2 := new(ExecutionPayload)
	err = json.Unmarshal(b, msg2)
	require.NoError(t, err)
	require.Equal(t, msg, msg2)
}

func TestBuilderSSZEncoding(t *testing.T) {
	tests := []_SSZable{
		&Eth1Data{}, &BeaconBlockHeader{}, &SignedBeaconBlockHeader{}, &ProposerSlashing{}, &Checkpoint{}, &AttestationData{}, &IndexedAttestation{}, &AttesterSlashing{}, &Attestation{}, &Deposit{}, &VoluntaryExit{}, &SyncAggregate{},
		&RegisterValidatorRequestMessage{},
		&ExecutionPayloadHeader{},
		&BlindedBeaconBlockBody{}, &BlindedBeaconBlock{},
		&BuilderBid{},
	}
	for _, test := range tests {
		// fmt.Printf("%T \n", test)
		buf1, err := test.MarshalSSZ()
		require.NoError(t, err)
		require.LessOrEqual(t, 0, len(buf1))

		_, err = test.MarshalSSZTo([]byte{})
		require.NoError(t, err)

		err = test.UnmarshalSSZ([]byte{})
		require.Error(t, err, err)

		err = test.UnmarshalSSZ(buf1)
		fmt.Println("size", uint64(len(buf1)))
		require.NoError(t, err, err)

		n := test.SizeSSZ()
		require.LessOrEqual(t, 0, n)

		_, err = test.HashTreeRoot()
		require.NoError(t, err)

		hh := ssz.DefaultHasherPool.Get()
		err = test.HashTreeRootWith(hh)
		require.NoError(t, err)
	}
}
