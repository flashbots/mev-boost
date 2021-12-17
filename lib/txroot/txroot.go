package txroot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/pkg/errors"
)

const (
	mask0 = ^uint64((1 << (1 << iota)) - 1)
	mask1
	mask2
	mask3
	mask4
	mask5
)

const (
	bit0 = uint8(1 << iota)
	bit1
	bit2
	bit3
	bit4
	bit5
)

const _MaxTransactionsPerPayload uint64 = 1048576
const _MaxBytesPerTransaction uint64 = 1073741824

// TransactionsRoot computes the HTR for the Transactions property of the ExecutionPayload
// The code was largely copy/pasted from the code generated to compute the HTR of the entire
// ExecutionPayload.
func TransactionsRoot(txs [][]byte) ([32]byte, error) {
	hasher := CustomSHA256Hasher()
	listMarshaling := make([][]byte, 0)
	for i := 0; i < len(txs); i++ {
		rt, err := transactionRoot(txs[i])
		if err != nil {
			return [32]byte{}, err
		}
		listMarshaling = append(listMarshaling, rt[:])
	}

	bytesRoot, err := BitwiseMerkleize(hasher, listMarshaling, uint64(len(listMarshaling)), _MaxTransactionsPerPayload)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute  merkleization")
	}
	bytesRootBuf := new(bytes.Buffer)
	if err := binary.Write(bytesRootBuf, binary.LittleEndian, uint64(len(txs))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal length")
	}
	bytesRootBufRoot := make([]byte, 32)
	copy(bytesRootBufRoot, bytesRootBuf.Bytes())
	return MixInLength(bytesRoot, bytesRootBufRoot), nil
}

func transactionRoot(tx []byte) ([32]byte, error) {
	hasher := CustomSHA256Hasher()
	chunkedRoots, err := packChunks(tx)
	if err != nil {
		return [32]byte{}, err
	}

	maxLength := (_MaxBytesPerTransaction + 31) / 32
	bytesRoot, err := BitwiseMerkleize(hasher, chunkedRoots, uint64(len(chunkedRoots)), maxLength)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute merkleization")
	}
	bytesRootBuf := new(bytes.Buffer)
	if err := binary.Write(bytesRootBuf, binary.LittleEndian, uint64(len(tx))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal length")
	}
	bytesRootBufRoot := make([]byte, 32)
	copy(bytesRootBufRoot, bytesRootBuf.Bytes())
	return MixInLength(bytesRoot, bytesRootBufRoot), nil
}

// Pack a given byte array into chunks. It'll pad the last chunk with zero bytes if
// it does not have length bytes per chunk.
func packChunks(bytes []byte) ([][]byte, error) {
	numItems := len(bytes)
	var chunks [][]byte
	for i := 0; i < numItems; i += 32 {
		j := i + 32
		// We create our upper bound index of the chunk, if it is greater than numItems,
		// we set it as numItems itself.
		if j > numItems {
			j = numItems
		}
		// We create chunks from the list of items based on the
		// indices determined above.
		chunks = append(chunks, bytes[i:j])
	}

	if len(chunks) == 0 {
		return chunks, nil
	}

	// Right-pad the last chunk with zero bytes if it does not
	// have length bytes.
	lastChunk := chunks[len(chunks)-1]
	for len(lastChunk) < 32 {
		lastChunk = append(lastChunk, 0)
	}
	chunks[len(chunks)-1] = lastChunk
	return chunks, nil
}

// BitwiseMerkleize - given ordered BYTES_PER_CHUNK-byte chunks, if necessary utilize
// zero chunks so that the number of chunks is a power of two, Merkleize the chunks,
// and return the root.
// Note that merkleize on a single chunk is simply that chunk, i.e. the identity
// when the number of chunks is one.
func BitwiseMerkleize(hasher HashFn, chunks [][]byte, count, limit uint64) ([32]byte, error) {
	if count > limit {
		return [32]byte{}, errors.New("merkleizing list that is too large, over limit")
	}
	hashFn := NewHasherFunc(hasher)
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	return Merkleize(hashFn, count, limit, leafIndexer), nil
}

// MixInLength appends hash length to root
func MixInLength(root [32]byte, length []byte) [32]byte {
	var hash [32]byte
	h := sha256.New()
	h.Write(root[:])
	h.Write(length)
	// The hash interface never returns an error, for that reason
	// we are not handling the error below. For reference, it is
	// stated here https://golang.org/pkg/hash/#Hash
	// #nosec G104
	h.Sum(hash[:0])
	return hash
}

// Depth retrieves the appropriate depth for the provided trie size.
func Depth(v uint64) (out uint8) {
	// bitmagic: binary search through a uint32, offset down by 1 to not round powers of 2 up.
	// Then adding 1 to it to not get the index of the first bit, but the length of the bits (depth of tree)
	// Zero is a special case, it has a 0 depth.
	// Example:
	//  (in out): (0 0), (1 1), (2 1), (3 2), (4 2), (5 3), (6 3), (7 3), (8 3), (9 4)
	if v == 0 {
		return 0
	}
	v--
	if v&mask5 != 0 {
		v >>= bit5
		out |= bit5
	}
	if v&mask4 != 0 {
		v >>= bit4
		out |= bit4
	}
	if v&mask3 != 0 {
		v >>= bit3
		out |= bit3
	}
	if v&mask2 != 0 {
		v >>= bit2
		out |= bit2
	}
	if v&mask1 != 0 {
		v >>= bit1
		out |= bit1
	}
	if v&mask0 != 0 {
		out |= bit0
	}
	out++
	return
}

// Merkleize with log(N) space allocation
func Merkleize(hasher Hasher, count, limit uint64, leaf func(i uint64) []byte) (out [32]byte) {
	if count > limit {
		panic("merkleizing list that is too large, over limit")
	}
	if limit == 0 {
		return
	}
	if limit == 1 {
		if count == 1 {
			copy(out[:], leaf(0))
		}
		return
	}
	depth := Depth(count)
	limitDepth := Depth(limit)
	tmp := make([][32]byte, limitDepth+1)

	j := uint8(0)
	hArr := [32]byte{}
	h := hArr[:]

	merge := func(i uint64) {
		// merge back up from bottom to top, as far as we can
		for j = 0; ; j++ {
			// stop merging when we are in the left side of the next combi
			if i&(uint64(1)<<j) == 0 {
				// if we are at the count, we want to merge in zero-hashes for padding
				if i == count && j < depth {
					v := hasher.Combi(hArr, ZeroHashes[j])
					copy(h, v[:])
				} else {
					break
				}
			} else {
				// keep merging up if we are the right side
				v := hasher.Combi(tmp[j], hArr)
				copy(h, v[:])
			}
		}
		// store the merge result (may be no merge, i.e. bottom leaf node)
		copy(tmp[j][:], h)
	}

	// merge in leaf by leaf.
	for i := uint64(0); i < count; i++ {
		copy(h, leaf(i))
		merge(i)
	}

	// complement with 0 if empty, or if not the right power of 2
	if (uint64(1) << depth) != count {
		copy(h, ZeroHashes[0][:])
		merge(count)
	}

	// the next power of two may be smaller than the ultimate virtual size,
	// complement with zero-hashes at each depth.
	for j := depth; j < limitDepth; j++ {
		tmp[j+1] = hasher.Combi(tmp[j], ZeroHashes[j])
	}

	return tmp[limitDepth]
}
