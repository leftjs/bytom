package test

import (
	"context"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/Masterminds/glide/cfg"
)

const (
	vmVersion    = 1
	blockVersion = 1
	assetVersion = 1
	maxNonce     = ^uint64(0)
)

// MockTxPool mock transaction pool
func MockTxPool() *protocol.TxPool {
	return protocol.NewTxPool()
}

// MockChain mock chain with genesis block
func MockChain(testDB dbm.DB) (*protocol.Chain, error) {
	store := leveldb.NewStore(testDB)
	txPool := MockTxPool()
	genesisBlock, err := GenerateGenesisBlock()
	if err != nil {
		return nil, err
	}

	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		return nil, err
	}
	if err := chain.SaveBlock(genesisBlock); err != nil {
		return nil, err
	}
	if err := chain.ConnectBlock(genesisBlock); err != nil {
		return nil, err
	}

	return chain, nil
}

// MockUTXO mock a utxo
func MockUTXO(controlProg *account.CtrlProgram) *account.UTXO {
	utxo := &account.UTXO{}
	utxo.OutputID = bc.Hash{V0: 1}
	utxo.SourceID = bc.Hash{V0: 2}
	utxo.AssetID = *consensus.BTMAssetID
	utxo.Amount = 1000000000
	utxo.SourcePos = 0
	utxo.ControlProgram = controlProg.ControlProgram
	utxo.AccountID = controlProg.AccountID
	utxo.Address = controlProg.Address
	utxo.ControlProgramIndex = controlProg.KeyIndex
	return utxo
}

// MockTx mock a tx
func MockTx(utxo *account.UTXO, testAccount *account.Account) (*txbuilder.Template, *types.TxData, error) {
	txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo)
	if err != nil {
		return nil, nil, err
	}

	b := txbuilder.NewBuilder(time.Now())
	b.AddInput(txInput, sigInst)
	out := types.NewTxOutput(*consensus.BTMAssetID, 100, []byte{byte(vm.OP_FAIL)})
	b.AddOutput(out)
	return b.Build()
}

// MockSign sign a tx
func MockSign(tpl *txbuilder.Template, hsm *pseudohsm.HSM, password string) (bool, error) {
	err := txbuilder.Sign(nil, tpl, nil, password, func(_ context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
		return hsm.XSign(xpub, path, data[:], password)
	})
	if err != nil {
		return false, err
	}
	return txbuilder.SignProgress(tpl), nil
}

// MockBlock mock a block
func MockBlock() *bc.Block {
	return &bc.Block{
		BlockHeader: &bc.BlockHeader{Height: 1},
	}
}

// GenerateGenesisBlock will return genesis block
func GenerateGenesisBlock() (*types.Block, error) {
	genesisCoinbaseTx := cfg.GenerateGenesisTx()
	merkleRoot, err := bc.TxMerkleRoot([]*bc.Tx{genesisCoinbaseTx.Tx})
	if err != nil {
		return nil, err
	}

	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, err := bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		return nil, err
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     4216085,
			Timestamp: 1516788453,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
			Bits: 2305843009222082559,
		},
		Transactions: []*types.Tx{genesisCoinbaseTx},
	}
	return block, nil
}
