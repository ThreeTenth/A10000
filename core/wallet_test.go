package core_test

import (
	"a10000/core"
	"testing"
)

func TestCreateWallet(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if wallet.PrivateKey == nil || wallet.PublicKey == nil {
		t.Fatal("Wallet keys should not be nil")
	}

	t.Logf("Wallet created with public key: %s", wallet.Address())

}

func TestSignTransaction(t *testing.T) {
	tom, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate tom: %v", err)
	}
	alice, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate alice: %v", err)
	}
	anna, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate anna: %v", err)
	}

	// 该测试存在一个特殊 case:
	// 即在快速创建交易和打包区块时, 由于打包难度(block.Difficulty)比较低,
	// 所以会快速打包, 并开始下一次交易并打包,
	// 于是会导致 genesisTx 和 tomCoinbaseTx 的参数值(ID) 完全一致(inputs 一致, outputs 一致, 时间间隔极短, 所以时间戳也一致),
	// 这样会导致 ch.Outputs 的 key 已存在, 进而覆盖 genesisTx 而非新增键值对(tomCoinbaseTx.ID, tomCoinbaseTx).
	// 解决方案是增加挖矿难度或增加交易时间间隔, 以避免该问题.
	ch := core.CreateBlockchain()
	genesisTx := core.NewCoinbaseTX(tom.Address(), 50)
	err = ch.GenesisBlock(genesisTx)
	if err != nil {
		t.Fatalf("Failed to create genesis block: %v", err)
	}

	utxo := ch.FindUTXO(tom.Address())
	transactionTom2Alice, err := tom.NewTransaction(utxo, alice.Address(), 20, "test data")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	err = ch.AddTransaction(transactionTom2Alice)
	if err != nil {
		t.Fatalf("Failed to add transaction(tom to alice): %v", err)
	}

	index := len(ch.Blocks)
	previousHash := ch.Blocks[len(ch.Blocks)-1].Hash
	tomCoinbaseTx := core.NewCoinbaseTX(tom.Address(), 50)
	transactions := append([]*core.Transaction{tomCoinbaseTx}, ch.PendingTransactions...)
	b := core.CreateBlock(int64(index), transactions, previousHash)
	err = ch.AddBlock(b)
	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	utxo = ch.FindUTXO(alice.Address())

	transactionAlice2Anna, err := alice.NewTransaction(utxo, anna.Address(), 3, "test data")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	err = ch.AddTransaction(transactionAlice2Anna)
	if err != nil {
		t.Logf("Cann't add transaction(tom to anna): %v", err)
	}

	index = len(ch.Blocks)
	previousHash = ch.Blocks[len(ch.Blocks)-1].Hash
	tomCoinbaseTx = core.NewCoinbaseTX(tom.Address(), 50)
	transactions = append([]*core.Transaction{tomCoinbaseTx}, ch.PendingTransactions...)
	b = core.CreateBlock(int64(index), transactions, previousHash)
	err = ch.AddBlock(b)
	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	// 计算的余额
	utxo = ch.FindUTXO(tom.Address())
	tomBalance := tom.Balance(utxo)

	utxo = ch.FindUTXO(alice.Address())
	aliceBalance := alice.Balance(utxo)

	utxo = ch.FindUTXO(anna.Address())
	annaBalance := anna.Balance(utxo)

	if aliceBalance != 17 {
		t.Fatalf("Alice's balance is incorrect, expected 17, got %d", aliceBalance)
	}

	if annaBalance != 3 {
		t.Fatalf("Alice's balance is incorrect, expected 3, got %d", annaBalance)
	}

	if tomBalance != 130 {
		t.Fatalf("Tom's balance is incorrect, expected 130, got %d", tomBalance)
	}

	t.Logf("Tom's balance: %d, Alice's balance: %d, Anna's balance: %d", tomBalance, aliceBalance, annaBalance)
}

func TestSignTransactionNoInputs(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	ch := core.CreateBlockchain()
	uxto := ch.FindUTXO(wallet.Address())

	transaction, err := wallet.NewTransaction(uxto, "recipient_public_key", 100, "test data")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	if transaction.ID == "" {
		t.Fatal("Transaction ID should not be empty")
	}

	err = transaction.VerifySignature()
	if err.Error() == "no inputs" {
		t.Logf("Transaction signature verification failed: %v", err)
	} else {
		t.Fatalf("Transaction created with ID: %s, error: %v", transaction.ID, err)
	}

}
