package core

import (
	"a10000/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
)

// Wallet 钱包
// 仅包含公私钥
// 用于签名交易和生成新的交易
// 仅作测试使用
// 实际应用中, 钱包应由独立的第三方服务提供
// 例如, 使用 MetaMask 等钱包服务
// 这里的 Wallet 仅用于演示如何使用 ecdsa 包进行签名
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey // 私钥
	PublicKey  *ecdsa.PublicKey  // 公钥
}

func (w *Wallet) SignTransaction(tx *Transaction) (string, error) {
	// 序列化交易数据
	hashed := tx.Hash()

	// 使用私钥签名
	r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, []byte(hashed))
	if err != nil {
		return "", err
	}

	// 将签名转换为字符串
	signature := r.Text(16) + ":" + s.Text(16)
	return signature, nil
}

func (w *Wallet) NewTransaction(recipient string, amount int64, data string) (*Transaction, error) {
	if w.PrivateKey == nil || w.PublicKey == nil {
		return nil, errors.New("wallet is not initialized")
	}

	// 生成交易ID
	id := utils.GenerateTransactionID()

	// 创建交易
	sender := w.PublicKey.X.Text(16) + ":" + w.PublicKey.Y.Text(16) // 假设公钥的字符串表示
	transaction := NewTransaction(id, "", "", sender, recipient, amount, data)

	// 签名交易
	signature, err := w.SignTransaction(transaction)
	if err != nil {
		return nil, err
	}
	transaction.Signature = signature

	return transaction, nil
}

// 生成一个新的钱包
func NewWallet() (*Wallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}
