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

	if err != nil {
		t.Error(err)
	}
}
