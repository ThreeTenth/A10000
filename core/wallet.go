package core

import (
	"a10000/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"strconv"
	"strings"
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

func (w *Wallet) Balance(uouto map[string]TxOutput) int64 {
	balance := int64(0)
	for _, output := range uouto {
		if output.IsFor(w.Address()) {
			balance += output.Amount
		}
	}
	return balance
}

func (w *Wallet) Address() string {
	return w.PublicKey.X.Text(16) + ":" + w.PublicKey.Y.Text(16)
}

func (w *Wallet) SignTransaction(tx *Transaction) error {
	for i := 0; i < len(tx.Inputs); i++ {
		input := tx.Inputs[i]

		// 计算交易的 Hash
		// 交易的 Hash + 交易输入的 Hash
		// 确保交易不会被修改
		hashed := tx.Hash() + input.Hash()

		// 使用私钥签名
		r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, []byte(hashed))
		if err != nil {
			return err
		}

		// 将签名转换为字符串
		input.Signature = r.Text(16) + ":" + s.Text(16)
	}
	return nil
}

func (w *Wallet) NewTransaction(uouto map[string]TxOutput, to string, amount int64, data string) (*Transaction, error) {
	if w.PrivateKey == nil || w.PublicKey == nil {
		return nil, errors.New("wallet is not initialized")
	}

	// 创建交易
	inputPubKey := w.Address() // 假设公钥的字符串表示

	inputAmount := int64(0)
	inputs := make([]*TxInput, 0)
	outputs := make([]*TxOutput, 0)

	for key, output := range uouto {
		if output.IsFor(inputPubKey) {
			pairs := strings.Split(key, ":")
			if len(pairs) != 2 {
				return nil, errors.New("invalid output key")
			}
			txID := pairs[0]
			// pairs[1] 转换为 int64, 原生代码
			vout, err := strconv.Atoi(pairs[1])
			if err != nil {
				return nil, errors.New("invalid output key(Vout)")
			}
			inputAmount += output.Amount
			inputs = append(inputs, &TxInput{Txid: txID, Vout: vout, Signature: "", PubKey: inputPubKey})
			if amount <= inputAmount {
				break
			}
		}
	}

	outputs = append(outputs, &TxOutput{
		Amount:     amount,
		PubKeyHash: utils.Hash([]byte(to)),
	})

	outputs = append(outputs, &TxOutput{
		Amount:     inputAmount - amount,
		PubKeyHash: utils.Hash([]byte(inputPubKey)),
	})

	transaction := &Transaction{
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: utils.GetUTCTimestamp(),
	}

	// 签名交易
	err := w.SignTransaction(transaction)
	if err != nil {
		return nil, err
	}
	transaction.ID = transaction.Hash()

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
