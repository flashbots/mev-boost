package txroot

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestTransactionsRoot(t *testing.T) {

	tests := []struct {
		name    string
		txs     [][]byte
		want    string
		wantErr bool
	}{
		{
			"should succeed with nil",
			nil,
			"0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1",
			false,
		},
		{
			"should succeed with 0 txs",
			[][]byte{},
			"0x7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1",
			false,
		},
		{
			"should succeed with 1 empty tx",
			[][]byte{{}},
			"0x1547db04bc3b5505b4ebd93c929e5007d9739d30041b154a639f8565d3ec3083",
			false,
		},
		{
			"should succeed with 1 tx",
			[][]byte{
				common.Hex2Bytes("02f862018002028288b894f1a54b075fb71768ac31b33fd7c61ad8f9f7dd188080c001a0ddf84854772f5e3f34ac57c9e2b862952a54e346d1d8509839d3c832e82298e5a012be6ba681d3553470f5b4ff4e8cf02712e96574c9e0bc8e2c2abbb7f3f581ab"),
			},
			"0x4a87485a8a9264aae0e1a71f6d013a262be34fe964a19642a9bf18cd01e4d971",
			false,
		},
		{
			"should succeed with 1 tx",
			[][]byte{
				common.Hex2Bytes("02f862018002028288b894f1a54b075fb71768ac31b33fd7c61ad8f9f7dd188080c001a0ddf84854772f5e3f34ac57c9e2b862952a54e346d1d8509839d3c832e82298e5a012be6ba681d3553470f5b4ff4e8cf02712e96574c9e0bc8e2c2abbb7f3f581ab"),
				common.Hex2Bytes("f85f01028288b894f1a54b075fb71768ac31b33fd7c61ad8f9f7dd18808025a0ab6b0068c4b5e704e031850b29c3820b4c7b95f1eeb06a177c3ad7fda3b5975fa058593b17aafebb814156e6a6883340b701759a2e06b6d2ab53b4158d6f3c9c33"),
			},
			"0x4f9fec9d7b418d8efe319ce8829198cac5384ca8a27b8dba8c61396eba2a9f01",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransactionsRoot(tt.txs)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransactionsRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			hash := common.BytesToHash(got[:]).String()
			if !reflect.DeepEqual(hash, tt.want) {
				t.Errorf("TransactionsRoot() = %v, want %v", hash, tt.want)
			}
		})
	}
}
