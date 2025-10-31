package domain

import (
	"bytes"
	"encoding/gob"
	"log"

	"github.com/dgraph-io/badger/v3"
)

type UTXOSet struct {
	Blockchain *Blockchain
}

type SpendableUTXOData struct {
	TxID       []byte
	VoutIndex  int
	Amount     int64
	PubKeyHash []byte
}

func (u *UTXOSet) Reindex() {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
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

	allUTXOs := u.Blockchain.FindAllUTXO()

	err = db.Update(func(txn *badger.Txn) error {
		for txID, outs := range allUTXOs {
			key := append([]byte(utxoPrefix), []byte(txID)...)

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

func (bc *Blockchain) FindAllUTXO() map[string][]TxOutput {
	utxos := make(map[string][]TxOutput)
	spentTXOs := make(map[string][]int)

	it := bc.Iterator()

	for {
		block := it.Next()

		for _, tx := range block.Transactions {
			txIDStr := string(tx.ID)

			for outIdx, out := range tx.Vout {

				if spentTXOs[txIDStr] != nil {
					isSpent := false
					for _, spentIdx := range spentTXOs[txIDStr] {
						if spentIdx == outIdx {
							isSpent = true
							break
						}
					}
					if isSpent {
						continue
					}
				}

				utxos[txIDStr] = append(utxos[txIDStr], out)
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					inTxIDStr := string(in.TxID)
					spentTXOs[inTxIDStr] = append(spentTXOs[inTxIDStr], in.VoutIndex)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return utxos
}

func (u *UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var utxos []TxOutput
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(utxoPrefix)

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var outs []TxOutput

			err := item.Value(func(val []byte) error {
				dec := gob.NewDecoder(bytes.NewReader(val))
				return dec.Decode(&outs)
			})
			Handle(err)

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

func (u *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int64) (int64, map[string][]int) {
	spendableUTXOs := make(map[string][]int)
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

			if accumulated >= amount {
				return nil
			}
		}
		return nil
	})
	Handle(err)
	return accumulated, spendableUTXOs
}

func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {

			if !tx.IsCoinbase() {
				for _, vin := range tx.Vin {
					key := append([]byte(utxoPrefix), vin.TxID...)

					item, err := txn.Get(key)
					if err != nil {

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
						err = txn.Delete(key)
					} else {

						var outsData bytes.Buffer
						enc := gob.NewEncoder(&outsData)
						enc.Encode(updatedOuts)
						err = txn.Set(key, outsData.Bytes())
					}
					Handle(err)
				}
			}

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
			txID := item.KeyCopy(nil)
			txID = bytes.TrimPrefix(txID, prefix)

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

			if accumulated >= amount {
				return nil
			}
		}
		return nil
	})
	Handle(err)
	return accumulated, utxos
}
