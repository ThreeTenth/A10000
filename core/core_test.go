package core_test

import (
	"a10000/core"
	"testing"
)

func TestCreateBlock(t *testing.T) {
	tom, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate tom: %v", err)
	}

	ch := core.CreateBlockchain()
	genesisTx := core.NewCoinbaseTX(tom.Address(), 50)
	err = ch.GenesisBlock(genesisTx)
	if err != nil {
		t.Fatalf("Failed to create genesis block: %v", err)
	}
	index := len(ch.Blocks)
	previousHash := ch.Blocks[len(ch.Blocks)-1].Hash
	tomCoinbaseTx := core.NewCoinbaseTX(tom.Address(), 50)
	b := core.CreateBlock(int64(index), []*core.Transaction{tomCoinbaseTx}, previousHash)
	err = ch.AddBlock(b)

	t.Logf("Nonce: %d, Calculated Hash: %s, Expected Prefix: %d", b.Nonce, b.Hash, b.Difficulty)

	if err != nil {
		t.Error(err)
	}
}

func TestCalculateNonceAndDifficulty(t *testing.T) {
	// Create a block with difficulty 2
	b := &core.Block{
		Index:        1,
		Timestamp:    1234567890,
		Transactions: make([]*core.Transaction, 0),
		PreviousHash: "abc123",
		Difficulty:   1,
		Nonce:        0,
	}

	b.Mining()

	if len(b.Hash) == 0 {
		t.Error("Hash should not be empty after CalculateNonceAndDifficulty")
	}

	// Check that the hash has the required number of leading zeros
	prefix := b.Prefix()

	t.Logf("Nonce: %d, Calculated Hash: %s, Expected Prefix: %s", b.Nonce, b.Hash, prefix)

	if b.Hash[:b.Difficulty] != prefix {
		t.Errorf("Hash does not have required leading zeros: got %s, want prefix %s", b.Hash, prefix)
	}

	// Ensure Nonce is greater than zero (since it should have incremented)
	if b.Nonce == 0 {
		t.Error("Nonce should be greater than zero after PoW")
	}
}
