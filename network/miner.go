package network

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/vm"
	lua "github.com/yuin/gopher-lua"
)

const (
	miningInterval = 10 * time.Second
	rewardAmount   = 100
)

func StartMiningLoop(bc *domain.Blockchain, rdb *redis.Client, minerAddress string) {
	ctx := context.Background()

	ticker := time.NewTicker(miningInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Miner: Đang kiểm tra Mempool...")

		txsData, err := rdb.SMembers(ctx, mempoolKey).Result()
		if err != nil {
			log.Printf("Miner: Lỗi khi đọc mempool: %v", err)
			continue
		}

		if len(txsData) == 0 {
			log.Println("Miner: Mempool trống. Đang chờ...")
			continue
		}

		log.Printf("Miner: Tìm thấy %d giao dịch! Bắt đầu đào...", len(txsData))

		var validTxs []*domain.Transaction
		var processedTxsData [][]byte

		for _, data := range txsData {
			var tx domain.Transaction
			dec := gob.NewDecoder(bytes.NewReader([]byte(data)))
			if err := dec.Decode(&tx); err != nil {
				log.Printf("Miner: Lỗi giải mã TX: %v. Bỏ qua.", err)
				continue
			}

			processedTxsData = append(processedTxsData, []byte(data))

			prevTxs := bc.FindReferencedTxs(&tx)

			switch tx.Type {
			case domain.TxTypeTransfer:

				if tx.Verify(prevTxs) {
					log.Printf("Miner: TX Transfer hợp lệ: %x", tx.ID)
					validTxs = append(validTxs, &tx)
				} else {
					log.Printf("Miner: Phát hiện TX Transfer không hợp lệ: %x", tx.ID)
				}

			case domain.TxTypeContractDeploy:

				if tx.Verify(prevTxs) {
					log.Printf("Miner: TX Deploy hợp lệ: %x", tx.ID)

					senderPubKeyHash := domain.HashPubKey(tx.Vin[0].PublicKey)
					contractAddress := tx.ID

					err := executeVM(bc, tx.Payload, contractAddress, senderPubKeyHash, "", nil)

					if err != nil {

						log.Printf("Miner: LỖI VM (Deploy %x): %v. Giao dịch bị TỪ CHỐI.", tx.ID, err)

					} else {

						log.Println("Miner: VM Deploy thành công. Đang lưu code...")

						if err := bc.SetContractCode(tx.ID, tx.Payload); err != nil {
							log.Printf("Miner: LỖI không lưu được code: %v. Bỏ qua TX.", err)
						} else {

							validTxs = append(validTxs, &tx)
						}
					}
				} else {
					log.Printf("Miner: Phát hiện TX Deploy không hợp lệ: %x", tx.ID)
				}

			case domain.TxTypeContractCall:

				if tx.Verify(prevTxs) {
					log.Printf("Miner: TX Call hợp lệ: %x", tx.ID)

					payload, err := vm.ParseCallPayload(tx.Payload)
					if err != nil {
						log.Printf("Miner: LỖI Payload Call: %v. Từ chối TX.", err)
						continue
					}

					contractAddressBytes, err := hex.DecodeString(payload.ContractAddress)
					if err != nil {
						log.Printf("Miner: LỖI Địa chỉ Contract: %v. Từ chối TX.", err)
						continue
					}

					code, err := bc.GetContractCode(contractAddressBytes)
					if err != nil {
						log.Printf("Miner: LỖI không tìm thấy code contract: %v. Từ chối TX.", err)
						continue
					}

					senderPubKeyHash := domain.HashPubKey(tx.Vin[0].PublicKey)

					luaArgs := vm.ConvertArgsToLValues(payload.Args)

					err = executeVM(bc, code, contractAddressBytes, senderPubKeyHash, payload.FunctionName, luaArgs)

					if err != nil {

						log.Printf("Miner: LỖI VM (Call %x): %v. Giao dịch bị TỪ CHỐI.", tx.ID, err)
					} else {

						log.Printf("Miner: VM Call (%s) thành công.", payload.FunctionName)
						validTxs = append(validTxs, &tx)
					}
				} else {
					log.Printf("Miner: Phát hiện TX Call không hợp lệ: %x", tx.ID)
				}
			}

		}

		coinbaseTx := domain.NewCoinbaseTransaction(minerAddress, rewardAmount)

		allTxs := validTxs

		if len(allTxs) == 0 {
			log.Println("Miner: Không có TX hợp lệ để đào.")

			continue
		}

		allTxs = append([]*domain.Transaction{coinbaseTx}, allTxs...)

		for _, tx := range allTxs {

			senderPubKeyHash := domain.HashPubKey(tx.Vin[0].PublicKey)

			if tx.Type == domain.TxTypeContractDeploy {

				contractAddress := tx.ID
				err := executeVM(bc, tx.Payload, contractAddress, senderPubKeyHash, "", nil)
				if err != nil {
					log.Printf("Miner: LỖI VM (Deploy %x): %v", tx.ID, err)

				}
			}

			if tx.Type == domain.TxTypeContractCall {

				log.Println("Miner: (TODO) Xử lý Contract Call")
			}
		}

		bc.AddBlock(allTxs)

		log.Printf("Miner: === 🚀 ĐÀO THÀNH CÔNG BLOCK MỚI! ===")

		if len(processedTxsData) > 0 {
			pipe := rdb.Pipeline()
			for _, data := range processedTxsData {
				pipe.SRem(ctx, mempoolKey, data)
			}
			_, err = pipe.Exec(ctx)
			if err != nil {
				log.Printf("Miner: Lỗi dọn dẹp mempool: %v", err)
			}
			log.Printf("Miner: Đã dọn dẹp %d TX khỏi Mempool.", len(processedTxsData))
		}
	}
}

func executeVM(bc *domain.Blockchain, code []byte, contractAddress []byte, senderAddress []byte, functionName string, args []lua.LValue) error {

	v := vm.NewVM()
	defer v.Close()

	v.RegisterBridgeFunctions()

	v.SetContext(bc, contractAddress, senderAddress)

	if functionName == "" {

		return v.RunContractDeploy(code)
	} else {

		return v.RunContractCall(code, functionName, args)
	}
}
