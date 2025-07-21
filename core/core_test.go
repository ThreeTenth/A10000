package core_test

import (
	"a10000/core"
	"testing"
)

func TestCreateBlock(t *testing.T) {
	ch := core.CreateBlockchain()
	prevBlock := ch.List[len(ch.List)-1]
	b := core.CreateBlock(prevBlock.Index+1, "很好", prevBlock.Hash)
	err := ch.AddBlock(b)

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
		Transactions: "test",
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
