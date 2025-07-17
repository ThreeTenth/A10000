package core

import (
	"a10000/utils"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Block 区块
type Block struct {
	Index        int64  // 区块高度
	Timestamp    int64  // 区块创建时间戳
	Transactions string // 区块的数据
	PreviousHash string // 上一个区块的 Hash
	Hash         string // 当前区块的 Hash
}

func (b *Block) String() []byte {
	formatStr := fmt.Sprintf("%d%d%s%s", b.Index, b.Timestamp, b.Transactions, b.PreviousHash)
	return []byte(formatStr)
}

// Blockchain 区块链
type Blockchain struct {
	List []*Block
}

// GenesisBlock 创世区块
func (ch *Blockchain) GenesisBlock() {
	ch.List = append(ch.List, CreateBlock(0, "", "0"))
}

// AddBlock 向区块链中添加一个区块
func (ch *Blockchain) AddBlock(b *Block) error {
	lastBlock := ch.List[len(ch.List)-1]

	if b.Index != lastBlock.Index+1 {
		return errors.New("无效的区块: Index 错误")
	}

	if b.PreviousHash != lastBlock.Hash {
		return errors.New("无效的区块: PreviousHash 错误")
	}

	ch.List = append(ch.List, b)

	return nil
}

func CreateBlockchain() *Blockchain {
	var ch Blockchain
	ch.GenesisBlock()
	return &ch
}

// CreateBlock 创建一个区块
func CreateBlock(index int64, transactions string, previousHash string) *Block {
	var b Block

	b.Index = index
	b.Timestamp = utils.GetUTCTimestamp()
	b.Transactions = transactions
	b.PreviousHash = previousHash

	bytes := sha256.Sum256(b.String())
	b.Hash = hex.EncodeToString(bytes[:])

	return &b
}
