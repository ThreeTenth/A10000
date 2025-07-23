package core

import (
	"a10000/utils"
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
	return utils.Hash(b.String())
}

func (b *Block) String() []byte {
	formatStr := fmt.Sprintf("%d%d%v%s%d%d", b.Index, b.Timestamp, b.Transactions, b.PreviousHash, b.Nonce, b.Difficulty)
	return []byte(formatStr)
}

// Blockchain 区块链
type Blockchain struct {
	Blocks              []*Block            // 区块链
	PendingTransactions []*Transaction      // 待处理的交易
	Outputs             map[string]TxOutput // 区块链中余额不是直接存储的，而是通过 UTXO 计算得出. key: txid:index => value: TxOutput
}

func (ch *Blockchain) OutputKey(txid string, vout int) string {
	return fmt.Sprintf("%s:%d", txid, vout)
}

func (ch *Blockchain) AddTransaction(tx *Transaction) error {
	// 验证交易签名
	if err := tx.VerifySignature(); err != nil {
		return err
	}
	// 验证交易金额是否足够
	inputAmount := int64(0)
	for i := 0; i < len(tx.Inputs); i++ {
		output := ch.Outputs[ch.OutputKey(tx.Inputs[i].Txid, tx.Inputs[i].Vout)]
		inputAmount += output.Amount
	}
	outputAmount := int64(0)
	for i := 0; i < len(tx.Outputs); i++ {
		outputAmount += tx.Outputs[i].Amount
	}
	if inputAmount < outputAmount {
		return errors.New("无效的交易: 金额不足")
	}
	for _, input := range tx.Inputs {
		// 使用 ch.Outputs 来判断交易是否合法
		if _, ok := ch.Outputs[ch.OutputKey(input.Txid, input.Vout)]; !ok {
			return errors.New("无效的交易: 交易引用了不存在的输出")
		}

		// 验证交易是否已经入链
		// for _, block := range ch.Blocks {
		// 	for _, chainTx := range block.Transactions {
		// 		if chainTx.Exist(input) {
		// 			return errors.New("无效的交易: 交易已入链")
		// 		}
		// 	}
		// }

		// 验证交易是否已经存在
		for _, pendingTx := range ch.PendingTransactions {
			if pendingTx.Exist(input) {
				return errors.New("无效的交易: 交易已存在")
			}
		}
	}
	ch.PendingTransactions = append(ch.PendingTransactions, tx)
	return nil
}

// GenesisBlock 创世区块
// 创世区块的交易是 coinbase 交易
func (ch *Blockchain) GenesisBlock(coinbaseTx *Transaction) error {
	ch.Blocks = append(ch.Blocks, CreateBlock(0, []*Transaction{coinbaseTx}, "0"))

	for j := 0; j < len(coinbaseTx.Outputs); j++ {
		ch.Outputs[ch.OutputKey(coinbaseTx.ID, j)] = *coinbaseTx.Outputs[j]
	}

	return nil
}

// AddBlock 向区块链中添加一个区块
func (ch *Blockchain) AddBlock(b *Block) error {
	lastBlock := ch.Blocks[len(ch.Blocks)-1]

	if b.Index != lastBlock.Index+1 {
		return errors.New("无效的区块: Index 错误")
	}

	if b.PreviousHash != lastBlock.Hash {
		return errors.New("无效的区块: PreviousHash 错误")
	}

	if len(b.Transactions) == 0 {
		return errors.New("无效的区块: 交易数为 0")
	}

	coinbaseTx := b.Transactions[0]
	inputs := coinbaseTx.Inputs
	if (len(inputs) != 1) || coinbaseTx.Inputs[0].Txid != "" || coinbaseTx.Inputs[0].Vout != -1 {
		return errors.New("无效的区块: 区块的第一个交易必须是 coinbase 交易(01)")
	}

	for _, tx := range b.Transactions[1:] {
		if len(tx.Inputs) == 1 && tx.Inputs[0].Txid == "" && tx.Inputs[0].Vout == -1 {
			return errors.New("无效的区块: 区块不允许存在多个 coinbase 交易(02)")
		}
	}

	if err := b.Verification(); err != nil {
		return err
	}

	for i := 0; i < len(b.Transactions); i++ {
		tx := b.Transactions[i]
		if tx.Inputs[0].Vout != -1 { // Not a coinbase transaction
			for _, input := range tx.Inputs {
				delete(ch.Outputs, ch.OutputKey(input.Txid, input.Vout))
			}
		}
		for j := 0; j < len(tx.Outputs); j++ {
			ch.Outputs[ch.OutputKey(tx.ID, j)] = *tx.Outputs[j]
		}
		for j := 0; j < len(ch.PendingTransactions); j++ {
			pendingTx := ch.PendingTransactions[j]
			if pendingTx.ID == tx.ID {
				ch.PendingTransactions = append(ch.PendingTransactions[:j], ch.PendingTransactions[j+1:]...)
			}
		}

	}

	ch.Blocks = append(ch.Blocks, b)

	return nil
}

// 查找地址的余额
func (ch *Blockchain) FindUTXO(address string) map[string]TxOutput {
	utxo := make(map[string]TxOutput)
	// 查找 Outputs
	// 转换为 for range
	for key, output := range ch.Outputs {
		if output.IsFor(address) {
			utxo[key] = output
		}
	}
	return utxo
}

func CreateBlockchain() *Blockchain {
	var ch Blockchain
	ch.Blocks = make([]*Block, 0)
	ch.PendingTransactions = make([]*Transaction, 0)
	ch.Outputs = make(map[string]TxOutput)
	return &ch
}

// CreateBlock 创建一个区块
func CreateBlock(index int64, transactions []*Transaction, previousHash string) *Block {
	var b Block

	b.Index = index
	b.Timestamp = utils.GetUTCTimestamp()
	b.Transactions = transactions
	b.PreviousHash = previousHash
	b.Difficulty = 2 // 设置工作量证明的难度
	b.Nonce = 0
	b.Mining() // 计算 Nonce 和 Hash
	// b.Hash = b.CalculateHash()

	return &b
}
