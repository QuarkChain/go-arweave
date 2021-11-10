package arweave

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimple1Item(t *testing.T) {
	es := make([]*element, 1)
	for i := 0; i < len(es); i++ {
		var bs [32]byte
		bs[i] = byte(i)
		es[i] = &element{data: bs, note: *big.NewInt(int64(i + 1000))}
	}

	rh, tree := GenerateTree(es)
	for i := 0; i < len(es); i++ {
		target := big.NewInt(int64(i + 999))
		path := GeneratePath(rh, target, tree)
		leaf, startOff, endOff, err := ValidatePath(rh, target, big.NewInt(int64(len(es)+1000)), path)
		require.NoError(t, err)
		require.True(t, target.Cmp(endOff) < 0)
		require.True(t, target.Cmp(startOff) >= 0)

		var bs [32]byte
		bs[i] = byte(i)
		require.EqualValues(t, leaf, bs, "leaf incorrect")
	}

}

func TestSimple3Item(t *testing.T) {
	es := make([]*element, 3)
	for i := 0; i < len(es); i++ {
		var bs [32]byte
		bs[i] = byte(i)
		es[i] = &element{data: bs, note: *big.NewInt(int64(i + 1000))}
	}

	rh, tree := GenerateTree(es)
	for i := 0; i < len(es); i++ {
		target := big.NewInt(int64(i + 999))
		path := GeneratePath(rh, target, tree)
		leaf, startOff, endOff, err := ValidatePath(rh, target, big.NewInt(int64(len(es)+1000)), path)
		require.NoError(t, err)
		require.True(t, target.Cmp(endOff) < 0)
		require.True(t, target.Cmp(startOff) >= 0)

		var bs [32]byte
		bs[i] = byte(i)
		require.EqualValues(t, leaf, bs, "leaf incorrect")
	}
}

func TestSimpleEvenItem(t *testing.T) {
	es := make([]*element, 64*1024)
	for i := 0; i < len(es); i++ {
		var bs [32]byte
		copy(bs[:], big.NewInt(int64(i)).Bytes())
		es[i] = &element{data: bs, note: *big.NewInt(int64(i + 1))}
	}

	rh, tree := GenerateTree(es)
	for i := 0; i < 100; i++ {
		target := big.NewInt(int64(rand.Uint64() % (64 * 1024)))
		path := GeneratePath(rh, target, tree)
		leaf, startOff, endOff, err := ValidatePath(rh, target, big.NewInt(64*1024), path)
		require.NoError(t, err)
		require.True(t, target.Cmp(endOff) < 0)
		require.True(t, target.Cmp(startOff) >= 0)

		var bs [32]byte
		copy(bs[:], target.Bytes())
		require.EqualValues(t, leaf, bs, "leaf incorrect")
	}
}

// TODO: Test sample root
