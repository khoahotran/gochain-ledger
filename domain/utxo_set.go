package domain

import (
	"bytes"
	"encoding/gob"
	"log"

	"github.com/dgraph-io/badger/v3"
)

// UTXOSet quản lý bộ UTXO
type UTXOSet struct {
	Blockchain *Blockchain
}

// (Thêm struct này ở đầu file domain/utxo_set.go)
type SpendableUTXOData struct {
	TxID       []byte
	VoutIndex  int
	Amount     int64
	PubKeyHash []byte
}

// Reindex (Xóa và xây dựng lại toàn bộ UTXO Set)
func (u *UTXOSet) Reindex() {
	db := u.Blockchain.Database

	// 1. Xóa tất cả key có prefix "utxo-"
	err := db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Chỉ cần key
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(utxoPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := txn.Delete(it.Item().Key())
			if err != nil {
				return err
			}
		}
		return nil
	})
	Handle(err)

	// 2. Tìm tất cả Output chưa tiêu
	allUTXOs := u.Blockchain.FindAllUTXO()

	// 3. Ghi lại vào DB
	err = db.Update(func(txn *badger.Txn) error {
		for txID, outs := range allUTXOs {
			key := append([]byte(utxoPrefix), []byte(txID)...)
			// Serialize danh sách Output
			var outsData bytes.Buffer
			enc := gob.NewEncoder(&outsData)
			err := enc.Encode(outs)
			Handle(err)

			err = txn.Set(key, outsData.Bytes())
			if err != nil {
				return err
			}
		}
		return nil
	})
	Handle(err)
	log.Println("UTXO Set đã được re-index!")
}

// FindAllUTXO (Quét toàn bộ blockchain để tìm UTXO)
// Đây là hàm nội bộ cho Reindex
func (bc *Blockchain) FindAllUTXO() map[string][]TxOutput {
	utxos := make(map[string][]TxOutput)
	spentTXOs := make(map[string][]int) // map[txID][]index

	it := bc.Iterator()

	for {
		block := it.Next()

		// Duyệt qua các transaction trong block
		for _, tx := range block.Transactions {
			txIDStr := string(tx.ID)

			// 1. Duyệt Outputs
			for outIdx, out := range tx.Vout {
				// Kiểm tra xem output này đã bị tiêu (spent) chưa
				if spentTXOs[txIDStr] != nil {
					isSpent := false
					for _, spentIdx := range spentTXOs[txIDStr] {
						if spentIdx == outIdx {
							isSpent = true
							break
						}
					}
					if isSpent {
						continue // Bỏ qua output đã bị tiêu
					}
				}
				// Thêm output chưa tiêu vào map
				utxos[txIDStr] = append(utxos[txIDStr], out)
			}

			// 2. Duyệt Inputs (Chỉ áp dụng cho tx không phải coinbase)
			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					inTxIDStr := string(in.TxID)
					spentTXOs[inTxIDStr] = append(spentTXOs[inTxIDStr], in.VoutIndex)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break // Dừng ở Genesis block
		}
	}
	return utxos
}

// === Logic tìm kiếm UTXO ===

// FindUTXO (Tìm tất cả UTXO của một người - Dùng cho `balance`)
func (u *UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var utxos []TxOutput
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(utxoPrefix)

		// Duyệt qua tất cả các key utxo-
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var outs []TxOutput

			err := item.Value(func(val []byte) error {
				dec := gob.NewDecoder(bytes.NewReader(val))
				return dec.Decode(&outs)
			})
			Handle(err)

			// Duyệt qua các output trong entry
			for _, out := range outs {
				if out.IsLockedWithKey(pubKeyHash) {
					utxos = append(utxos, out)
				}
			}
		}
		return nil
	})
	Handle(err)
	return utxos
}

// FindSpendableOutputs (Tìm UTXO đủ cho 1 giao dịch - Dùng cho `send`)
func (u *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int64) (int64, map[string][]int) {
	spendableUTXOs := make(map[string][]int) // map[txID][]index
	var accumulated int64 = 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(utxoPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			txID := string(bytes.TrimPrefix(item.Key(), prefix))

			var outs []TxOutput
			err := item.Value(func(val []byte) error {
				dec := gob.NewDecoder(bytes.NewReader(val))
				return dec.Decode(&outs)
			})
			Handle(err)

			for outIdx, out := range outs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					spendableUTXOs[txID] = append(spendableUTXOs[txID], outIdx)
				}
			}
			// Nếu đã đủ tiền thì dừng
			if accumulated >= amount {
				return nil // Thoát sớm
			}
		}
		return nil
	})
	Handle(err)
	return accumulated, spendableUTXOs
}

// Update (Tối ưu hóa: Cập nhật UTXO Set khi có block mới)
func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			// 1. Xóa Outputs đã bị tiêu (Inputs)
			if !tx.IsCoinbase() {
				for _, vin := range tx.Vin {
					key := append([]byte(utxoPrefix), vin.TxID...)

					item, err := txn.Get(key)
					if err != nil {
						// Có thể đã bị xử lý bởi tx khác trong cùng block
						continue
					}

					var outs []TxOutput
					err = item.Value(func(val []byte) error {
						dec := gob.NewDecoder(bytes.NewReader(val))
						return dec.Decode(&outs)
					})
					Handle(err)

					updatedOuts := []TxOutput{}
					for outIdx, out := range outs {
						if outIdx != vin.VoutIndex {
							updatedOuts = append(updatedOuts, out)
						}
					}

					if len(updatedOuts) == 0 {
						err = txn.Delete(key) // Xóa nếu không còn output nào
					} else {
						// Ghi đè
						var outsData bytes.Buffer
						enc := gob.NewEncoder(&outsData)
						enc.Encode(updatedOuts)
						err = txn.Set(key, outsData.Bytes())
					}
					Handle(err)
				}
			}

			// 2. Thêm Outputs mới (Outputs)
			key := append([]byte(utxoPrefix), tx.ID...)
			var outsData bytes.Buffer
			enc := gob.NewEncoder(&outsData)
			enc.Encode(tx.Vout)
			err := txn.Set(key, outsData.Bytes())
			Handle(err)
		}
		return nil
	})
	Handle(err)
}

// (Dán hàm này vào cuối file domain/utxo_set.go)

// FindSpendableUTXOData tìm và trả về dữ liệu UTXO đầy đủ
func (u *UTXOSet) FindSpendableUTXOData(pubKeyHash []byte, amount int64) (int64, []SpendableUTXOData) {
	var utxos []SpendableUTXOData
	var accumulated int64 = 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(utxoPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			txID := item.KeyCopy(nil)             // Lấy key
			txID = bytes.TrimPrefix(txID, prefix) // Bỏ prefix

			var outs []TxOutput
			err := item.Value(func(val []byte) error {
				dec := gob.NewDecoder(bytes.NewReader(val))
				return dec.Decode(&outs)
			})
			Handle(err)

			for outIdx, out := range outs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					utxos = append(utxos, SpendableUTXOData{
						TxID:       txID,
						VoutIndex:  outIdx,
						Amount:     out.Value,
						PubKeyHash: out.PubKeyHash,
					})
				}
			}
			// Nếu đã đủ tiền thì dừng
			if accumulated >= amount {
				return nil // Thoát sớm
			}
		}
		return nil
	})
	Handle(err)
	return accumulated, utxos
}
