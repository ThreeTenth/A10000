package core

import (
	"a10000/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type TxInput struct {
	Txid      string `json:"txid"`      // 引用的交易ID
	Vout      int    `json:"vout"`      // 引用的交易输出索引
	Signature string `json:"signature"` // 签名
	PubKey    string `json:"pubkey"`    // 公钥
}

func (in *TxInput) String() string {
	formatString := fmt.Sprintf("%s%d%s", in.Txid, in.Vout, in.PubKey)
	return formatString
}

func (in *TxInput) Bytes() []byte {
	return []byte(in.String())
}

func (in *TxInput) Hash() string {
	bytes := sha256.Sum256(in.Bytes())
	return hex.EncodeToString(bytes[:])
}

func (in *TxInput) VerifySignature(txHashed string) error {
	hashed := txHashed + in.Hash()
	sender := in.PubKey // 假设 sender 是公钥的字符串表示
	// sender := w.PublicKey.X.Text(16) + w.PublicKey.Y.Text(16)
	signature := in.Signature
	// 	signature := r.Text(16) + ":" + s.Text(16)
	parts := strings.Split(signature, ":")
	if len(parts) != 2 {
		return errors.New("invalid signature format") // 确保签名格式正确
	}
	r, err := new(big.Int).SetString(parts[0], 16)
	if !err {
		return errors.New("invalid signature r value") // 确保 r 值正确
	}
	s, err := new(big.Int).SetString(parts[1], 16)
	if !err {
		return errors.New("invalid signature s value") // 确保 s 值正确
	}
	// 将公钥字符串转换为 ecdsa.PublicKey
	parts = strings.Split(sender, ":")
	if len(parts) != 2 {
		return errors.New("invalid PublicKey format") // 确保公钥格式正确
	}
	x, err := new(big.Int).SetString(parts[0], 16)
	if !err {
		return errors.New("invalid signature PublicKey.X value") // 确保 x 值正确
	}
	y, err := new(big.Int).SetString(parts[1], 16)
	if !err {
		return errors.New("invalid signature PublicKey.y value") // 确保 x 值正确
	}
	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
	// 验证签名
	ok := ecdsa.Verify(publicKey, []byte(hashed), r, s)
	if !ok {
		return errors.New("signature verification failed") // 验证失败
	}
	return nil
}

type TxOutput struct {
	Amount     int64  `json:"value"`      // 金额
	PubKeyHash string `json:"pubkeyhash"` // 接收方公钥 hash
}

func (out *TxOutput) String() string {
	formatString := fmt.Sprintf("%d%s", out.Amount, out.PubKeyHash)
	return formatString
}

// 判断 pubKey 的 hash 是否一致
func (out *TxOutput) IsFor(pubKey string) bool {
	return out.PubKeyHash == utils.Hash([]byte(pubKey))
}

// Transaction 交易
// 区块链的交易规则和用户余额计算方式:
//
// 由数个交易输入和数个交易输出构成一个交易
// 交易输入引用了前一个交易的输出
// 交易输出可以被下一个交易的输入引用
// 交易输入的金额之和必须等于交易输出的金额之和
// 交易输入的金额之和大于交易输出的金额之和时，差额作为矿工费奖励给矿工
// 交易输入的金额之和小于交易输出的金额之和时，交易无效
// 交易输出中, 可以有多个接收方, 一般情况下, 至少有两个:
// 一个是交易对象, 一个是自己
// 给自己的这个 output 是用于找零的.
type Transaction struct {
	ID        string      `json:"id"`        // 交易 Hash
	Inputs    []*TxInput  `json:"inputs"`    // 交易输入
	Outputs   []*TxOutput `json:"outputs"`   // 交易输出
	Timestamp int64       `json:"timestamp"` // 交易时间戳
}

func (tx *Transaction) String() string {
	formatInputs := ""
	for _, input := range tx.Inputs {
		formatInputs += input.String()
	}
	formatOutputs := ""
	for _, output := range tx.Outputs {
		formatOutputs += output.String()
	}

	formatStr := fmt.Sprintf("%d%s%s", tx.Timestamp, formatInputs, formatOutputs)
	return formatStr
}

func (tx *Transaction) Bytes() []byte {
	return []byte(tx.String())
}

func (tx *Transaction) Hash() string {
	bytes := sha256.Sum256(tx.Bytes())
	return hex.EncodeToString(bytes[:])
}

func (tx *Transaction) Exist(in *TxInput) bool {
	for _, input := range tx.Inputs {
		if input.Txid == in.Txid && input.Vout == in.Vout {
			return true
		}
	}
	return false
}

// 验证签名
func (tx *Transaction) VerifySignature() error {
	if len(tx.Inputs) == 0 {
		return errors.New("no inputs")
	}
	if len(tx.Outputs) == 0 {
		return errors.New("no outputs")
	}
	hashed := tx.Hash()
	if tx.ID != hashed {
		return errors.New("invalid transaction hash")
	}
	for i := 0; i < len(tx.Inputs); i++ {
		if err := tx.Inputs[i].VerifySignature(hashed); err != nil {
			return err
		}
	}

	return nil
}

func NewCoinbaseTX(minerAddress string, amount int64) *Transaction {
	// 创建交易
	inputs := make([]*TxInput, 0)
	outputs := make([]*TxOutput, 0)

	inputs = append(inputs, &TxInput{
		Txid:      "",
		Vout:      -1,
		Signature: "",
		PubKey:    "",
	})

	outputs = append(outputs, &TxOutput{
		Amount:     amount,
		PubKeyHash: utils.Hash([]byte(minerAddress)),
	})

	transaction := &Transaction{
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: utils.GetUTCTimestamp(),
	}

	transaction.ID = transaction.Hash()

	return transaction
}
