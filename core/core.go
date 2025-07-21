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
	Index        int64          // 区块高度
	Timestamp    int64          // 区块创建时间戳
	Transactions []*Transaction // 区块的数据
	PreviousHash string         // 上一个区块的 Hash
	Hash         string         // 当前区块的 Hash
	// 请编写PoW相关的字段
	Nonce      int64 // 工作量证明的随机数
	Difficulty int64 // 工作量证明的难度
}

func (b *Block) Prefix() string {
	prefix := ""
	for i := int64(0); i < b.Difficulty; i++ {
		prefix += "0"
	}
	return prefix
}

// Mining 挖矿, 计算区块的 Nonce 和 Difficulty
// 注意: 需要实现工作量证明的算法
// 这里可以使用简单的算法，例如: Nonce 从 0 开始递增
// 直到找到一个满足条件的 Nonce，使得 Hash 的前 Difficulty 位为 0
// 例如: Difficulty = 4 时，Hash 的前四位必须为 0000
func (b *Block) Mining() {
	prefix := b.Prefix()

	for b.Hash == "" || b.Hash[:b.Difficulty] != prefix {
		b.Nonce++
		b.Hash = b.CalculateHash()
	}
}

func (b *Block) Verification() error {
	if b.Difficulty < 1 {
		return errors.New("无效的区块: Difficulty 错误")
	}

	hast := b.CalculateHash()
	if hast != b.Hash {
		return errors.New("无效的区块: Hash 错误")
	}

	// 判断 hash 的前 Difficulty 位是否为 0
	prefix := b.Prefix()

	if b.Hash[:b.Difficulty] != prefix {
		return errors.New("无效的区块: Hash 前缀错误")
	}

	return nil
}

// CalculateHash 计算区块的 Hash
// 计算方式为: sha256(区块的字符串表示)
// 区块的字符串表示为: Index + Timestamp + Transactions + PreviousHash
// 注意: 需要将计算结果转换为十六进制字符串
func (b *Block) CalculateHash() string {
	bytes := sha256.Sum256(b.String())
	return hex.EncodeToString(bytes[:])
}

func (b *Block) String() []byte {
	formatStr := fmt.Sprintf("%d%d%v%s%d%d", b.Index, b.Timestamp, b.Transactions, b.PreviousHash, b.Nonce, b.Difficulty)
	return []byte(formatStr)
}

// Blockchain 区块链
type Blockchain struct {
	List []*Block
}

// GenesisBlock 创世区块
func (ch *Blockchain) GenesisBlock() {
	ch.List = append(ch.List, CreateBlock(0, make([]*Transaction, 0), "0"))
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

	if err := b.Verification(); err != nil {
		return err
	}

	ch.List = append(ch.List, b)

	return nil
}

// FindUTXO 查找未花费的交易输出.
// 这里的 UTXO 是指未花费的交易输出,
// 即所有交易中 Output 为指定地址的交易.
// 返回一个包含所有未花费交易的切片.
// 注意: 需要遍历所有区块和交易.
func (ch *Blockchain) FindUTXO(address string) []*Transaction {
	utxo := make([]*Transaction, 0)
	for _, block := range ch.List {
		if block == nil || block.Transactions == nil {
			continue
		}
		// 遍历区块中的所有交易
		for _, tx := range block.Transactions {
			if tx.Output == address {
				// 只添加未花费的交易
				utxo = append(utxo, tx)
			}
		}
	}
	return utxo
}

func CreateBlockchain() *Blockchain {
	var ch Blockchain
	ch.GenesisBlock()
	return &ch
}

// CreateBlock 创建一个区块
func CreateBlock(index int64, transactions []*Transaction, previousHash string) *Block {
	var b Block

	b.Index = index
	b.Timestamp = utils.GetUTCTimestamp()
	b.Transactions = transactions
	b.PreviousHash = previousHash
	b.Difficulty = 1 // 设置工作量证明的难度
	b.Nonce = 0
	b.Mining() // 计算 Nonce 和 Hash
	// b.Hash = b.CalculateHash()

	return &b
}
