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

// Transaction 交易
type Transaction struct {
	ID        string `json:"id"`        // 交易ID
	Input     string `json:"input"`     // 交易输入
	Output    string `json:"output"`    // 交易输出
	Sender    string `json:"sender"`    // 交易发送方公钥
	Recipient string `json:"recipient"` // 交易接收方公钥
	Amount    int64  `json:"amount"`    // 交易金额
	Data      string `json:"data"`      // 交易数据
	Timestamp int64  `json:"timestamp"` // 交易时间戳
	Signature string `json:"signature"` // 交易签名
}

func (t *Transaction) String() []byte {
	formatStr := fmt.Sprintf("%s%s%s%s%s%d%s%d", t.ID, t.Input, t.Output, t.Sender, t.Recipient, t.Amount, t.Data, t.Timestamp)
	return []byte(formatStr)
}

func (t *Transaction) Hash() string {
	bytes := sha256.Sum256(t.String())
	return hex.EncodeToString(bytes[:])
}

// 验证签名
func (t *Transaction) VerifySignature() error {
	// 序列化交易数据
	hashed := t.Hash()
	sender := t.Sender // 假设 Sender 是公钥的字符串表示
	// sender := w.PublicKey.X.Text(16) + w.PublicKey.Y.Text(16)
	signature := t.Signature
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

// NewTransaction 创建一个新的交易
func NewTransaction(id, input, output, sender, recipient string, amount int64, data string) *Transaction {
	return &Transaction{
		ID:        id,
		Input:     input,
		Output:    output,
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
		Data:      data,
		Timestamp: utils.GetUTCTimestamp(),
	}
}
