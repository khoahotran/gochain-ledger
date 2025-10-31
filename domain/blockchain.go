package domain

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v3"
)

const (
	dbPath              = "./tmp/blocks"
	lastHashKey         = "lh"
	utxoPrefix          = "utxo-"
	contractStatePrefix = "contract-state-"
	contractCodePrefix  = "contract-code-"
)

type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

func InitBlockchain(address string) *Blockchain {

	opts := badger.DefaultOptions(dbPath)
	opts.WithValueLogFileSize(1024 * 1024)
	opts.WithLogger(nil)
	db, err := badger.Open(opts)
	Handle(err)

	var lastHash []byte

	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte(lastHashKey)); err == badger.ErrKeyNotFound {
			log.Println("Không tìm thấy blockchain. Đang tạo mới...")

			coinbaseTx := NewCoinbaseTransaction(address, 100)
			genesis := NewGenesisBlock(coinbaseTx)
			log.Println("Block Genesis đã được tạo.")

			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handle(err)
			err = txn.Set([]byte(lastHashKey), genesis.Hash)
			Handle(err)
			lastHash = genesis.Hash
		} else {

			item, err := txn.Get([]byte(lastHashKey))
			Handle(err)
			err = item.Value(func(val []byte) error {
				lastHash = val
				return nil
			})
			Handle(err)
		}
		return nil
	})
	Handle(err)

	blockchain := &Blockchain{LastHash: lastHash, Database: db}

	utxoSet := UTXOSet{Blockchain: blockchain}
	utxoSet.Reindex()

	return blockchain
}

func ContinueBlockchain() *Blockchain {

	opts := badger.DefaultOptions(dbPath)
	opts.WithValueLogFileSize(1024 * 1024)
	opts.WithLogger(nil)
	db, err := badger.Open(opts)
	Handle(err)
	var lastHash []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(lastHashKey))
		if err == badger.ErrKeyNotFound {
			log.Panic("Blockchain chưa được khởi tạo. Hãy chạy 'gochain init' trước.")
		}
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		return err
	})
	Handle(err)

	blockchain := &Blockchain{LastHash: lastHash, Database: db}

	return blockchain
}

func (bc *Blockchain) AddBlock(transactions []*Transaction) {

	prevBlockHash := bc.LastHash
	newBlock := NewBlock(prevBlockHash, transactions)

	err := bc.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte(lastHashKey), newBlock.Hash)
		Handle(err)
		bc.LastHash = newBlock.Hash
		return err
	})
	Handle(err)

	utxoSet := UTXOSet{Blockchain: bc}
	utxoSet.Update(newBlock)
}

type BlockchainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{
		CurrentHash: bc.LastHash,
		Database:    bc.Database,
	}
}

func (it *BlockchainIterator) Next() *Block {
	var block *Block

	err := it.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(it.CurrentHash)
		Handle(err)
		err = item.Value(func(val []byte) error {
			block = DeserializeBlock(val)
			return nil
		})
		return err
	})
	Handle(err)

	it.CurrentHash = block.PrevBlockHash
	return block
}

func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	it := bc.Iterator()
	for {
		block := it.Next()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("Transaction không tồn tại")
}

func (bc *Blockchain) FindReferencedTxs(tx *Transaction) map[string]Transaction {
	prevTxs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.TxID)
		Handle(err)
		prevTxs[string(prevTx.ID)] = prevTx
	}
	return prevTxs
}

func (bc *Blockchain) Close() {
	bc.Database.Close()
}

func ContinueBlockchainReadOnly() *Blockchain {
	opts := badger.DefaultOptions(dbPath)
	opts.WithValueLogFileSize(1024 * 1024)
	opts.WithLogger(nil)

	opts.ReadOnly = true
	db, err := badger.Open(opts)
	Handle(err)
	var lastHash []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(lastHashKey))
		if err == badger.ErrKeyNotFound {
			return errors.New("Blockchain chưa được khởi tạo. Hãy chạy 'gochain init' trước")
		}
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		return err
	})
	Handle(err)

	blockchain := &Blockchain{LastHash: lastHash, Database: db}

	return blockchain
}

func (bc *Blockchain) SetContractState(contractAddress []byte, key []byte, value []byte) error {
	return bc.Database.Update(func(txn *badger.Txn) error {

		dbKey := append([]byte(contractStatePrefix), contractAddress...)
		dbKey = append(dbKey, key...)
		return txn.Set(dbKey, value)
	})
}

func (bc *Blockchain) GetContractState(contractAddress []byte, key []byte) ([]byte, error) {
	var value []byte
	err := bc.Database.View(func(txn *badger.Txn) error {
		dbKey := append([]byte(contractStatePrefix), contractAddress...)
		dbKey = append(dbKey, key...)

		item, err := txn.Get(dbKey)
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}
	return value, nil
}

func (bc *Blockchain) SetContractCode(contractAddress []byte, code []byte) error {
	return bc.Database.Update(func(txn *badger.Txn) error {

		dbKey := append([]byte(contractCodePrefix), contractAddress...)
		return txn.Set(dbKey, code)
	})
}

func (bc *Blockchain) GetContractCode(contractAddress []byte) ([]byte, error) {
	var code []byte
	err := bc.Database.View(func(txn *badger.Txn) error {
		dbKey := append([]byte(contractCodePrefix), contractAddress...)

		item, err := txn.Get(dbKey)
		if err == badger.ErrKeyNotFound {
			return fmt.Errorf("không tìm thấy contract: %x", contractAddress)
		}
		if err != nil {
			return err
		}

		code, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}
	return code, nil
}
