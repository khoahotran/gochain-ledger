package domain

import (
	"bytes"
	"errors"
	"log"

	"github.com/dgraph-io/badger/v3"
)

const (
	dbPath              = "./tmp/blocks" // Thư mục lưu DB
	lastHashKey         = "lh"           // Key để lưu hash của block cuối cùng
	utxoPrefix          = "utxo-"        // Prefix mới để lưu UTXO
	contractStatePrefix = "contract-state-"
)

// Blockchain giữ con trỏ đến DB và hash cuối cùng
type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

func InitBlockchain(address string) *Blockchain {
	// 1. Mở CSDL (giữ nguyên)
	opts := badger.DefaultOptions(dbPath)
	opts.WithValueLogFileSize(1024 * 1024)
	opts.WithLogger(nil)
	db, err := badger.Open(opts)
	Handle(err)

	var lastHash []byte

	// 2. Dùng Read-Write Transaction (Update)
	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte(lastHashKey)); err == badger.ErrKeyNotFound {
			log.Println("Không tìm thấy blockchain. Đang tạo mới...")

			// Cập nhật: Dùng address string
			coinbaseTx := NewCoinbaseTransaction(address, 100)
			genesis := NewGenesisBlock(coinbaseTx)
			log.Println("Block Genesis đã được tạo.")

			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handle(err)
			err = txn.Set([]byte(lastHashKey), genesis.Hash)
			Handle(err)
			lastHash = genesis.Hash
		} else {
			// (giữ nguyên logic)
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

	// 3. (MỚI) Re-index UTXO Set
	// Khi khởi tạo/mở blockchain, chúng ta cần đảm bảo bộ UTXO là mới nhất
	utxoSet := UTXOSet{Blockchain: blockchain}
	utxoSet.Reindex()

	return blockchain
}

func ContinueBlockchain() *Blockchain {
	// ... (giữ nguyên logic mở DB và lấy lastHash)
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

	// (MỚI) Re-index UTXO
	// utxoSet := UTXOSet{Blockchain: blockchain}
	// utxoSet.Reindex() // Tạm thời re-index mỗi khi mở
	// Tối ưu hóa: Chúng ta sẽ dùng hàm Update() khi AddBlock

	return blockchain
}

func (bc *Blockchain) AddBlock(transactions []*Transaction) {
	// ... (giữ nguyên logic tạo NewBlock)
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

	// (MỚI) Cập nhật UTXO Set sau khi thêm block
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

// Next trả về block tiếp theo (đi ngược từ block cuối về Genesis)
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

// === HÀM MỚI: FindTransaction (Cần cho Sign/Verify) ===
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	it := bc.Iterator()
	for {
		block := it.Next()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}
		// Nếu PrevBlockHash rỗng, tức là đã đến block Genesis
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("Transaction không tồn tại")
}

// === HÀM MỚI: FindReferencedTxs (Lấy map các tx cũ) ===
func (bc *Blockchain) FindReferencedTxs(tx *Transaction) map[string]Transaction {
	prevTxs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.TxID)
		Handle(err) // Nếu không tìm thấy, crash (logic nghiệp vụ)
		prevTxs[string(prevTx.ID)] = prevTx
	}
	return prevTxs
}

// Close đóng CSDL
func (bc *Blockchain) Close() {
	bc.Database.Close()
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func ContinueBlockchainReadOnly() *Blockchain {
	opts := badger.DefaultOptions(dbPath)
	opts.WithValueLogFileSize(1024 * 1024)
	opts.WithLogger(nil)

	// DÒNG MỚI: Mở ở chế độ chỉ đọc
	opts.ReadOnly = true
	db, err := badger.Open(opts)
	Handle(err)
	var lastHash []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(lastHashKey))
		if err == badger.ErrKeyNotFound {
			return errors.New("Blockchain chưa được khởi tạo. Hãy chạy 'gochain init' trước.")
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

	// (MỚI) Re-index UTXO
	// utxoSet := UTXOSet{Blockchain: blockchain}
	// utxoSet.Reindex() // Tạm thời re-index mỗi khi mở
	// Tối ưu hóa: Chúng ta sẽ dùng hàm Update() khi AddBlock

	return blockchain
}

// SetContractState lưu trữ một cặp key-value cho một contract
func (bc *Blockchain) SetContractState(contractAddress []byte, key []byte, value []byte) error {
	return bc.Database.Update(func(txn *badger.Txn) error {
		// Key = "contract-state-" + <địa chỉ contract> + <key>
		dbKey := append([]byte(contractStatePrefix), contractAddress...)
		dbKey = append(dbKey, key...)
		return txn.Set(dbKey, value)
	})
}

// GetContractState đọc một giá trị từ state của contract
func (bc *Blockchain) GetContractState(contractAddress []byte, key []byte) ([]byte, error) {
	var value []byte
	err := bc.Database.View(func(txn *badger.Txn) error {
		dbKey := append([]byte(contractStatePrefix), contractAddress...)
		dbKey = append(dbKey, key...)

		item, err := txn.Get(dbKey)
		if err == badger.ErrKeyNotFound {
			return nil // Không tìm thấy, trả về nil (không phải lỗi)
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
