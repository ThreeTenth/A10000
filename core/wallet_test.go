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

	t.Logf("Wallet created with public key: %s", wallet.PublicKey.X.Text(16)+":"+wallet.PublicKey.Y.Text(16))

}

func TestSignTransaction(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	transaction, err := wallet.NewTransaction("recipient_public_key", 100, "test data")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	if transaction.Signature == "" {
		t.Fatal("Transaction signature should not be empty")
	}

	if err = transaction.VerifySignature(); err != nil {
		t.Fatal("Transaction signature verification failed", err)
	}

	t.Logf("Transaction created with ID: %s and signature: %s", transaction.ID, transaction.Signature)
}
