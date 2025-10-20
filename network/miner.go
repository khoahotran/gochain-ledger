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
	miningInterval = 10 * time.Second // Đào 10 giây một lần
	rewardAmount   = 100              // Thưởng đào block
)

// StartMiningLoop là vòng lặp chạy nền của thợ đào
func StartMiningLoop(bc *domain.Blockchain, rdb *redis.Client, minerAddress string) {
	ctx := context.Background()

	// Tạo một "đồng hồ" đếm 10 giây
	ticker := time.NewTicker(miningInterval)
	defer ticker.Stop()

	// Vòng lặp vô tận, chạy mỗi khi đồng hồ kêu
	for range ticker.C {
		log.Println("Miner: Đang kiểm tra Mempool...")

		// 1. Lấy tất cả giao dịch từ Mempool (Redis)
		txsData, err := rdb.SMembers(ctx, mempoolKey).Result()
		if err != nil {
			log.Printf("Miner: Lỗi khi đọc mempool: %v", err)
			continue
		}

		if len(txsData) == 0 {
			log.Println("Miner: Mempool trống. Đang chờ...")
			continue // Không có gì để đào, chờ 10 giây tiếp
		}

		log.Printf("Miner: Tìm thấy %d giao dịch! Bắt đầu đào...", len(txsData))

		var validTxs []*domain.Transaction
		var processedTxsData [][]byte // Dùng để xóa khỏi Redis sau

		// 2. Xác thực các giao dịch
		// (Rất quan trọng! Không tin tưởng giao dịch từ mạng)
		for _, data := range txsData {
			var tx domain.Transaction
			dec := gob.NewDecoder(bytes.NewReader([]byte(data)))
			if err := dec.Decode(&tx); err != nil {
				log.Printf("Miner: Lỗi giải mã TX: %v. Bỏ qua.", err)
				continue
			}

			processedTxsData = append(processedTxsData, []byte(data))

			// Lấy các TX cũ mà TX này tham chiếu
			prevTxs := bc.FindReferencedTxs(&tx)

			// --- LOGIC MỚI: XỬ LÝ THEO LOẠI TX ---
			switch tx.Type {
			case domain.TxTypeTransfer:
				// Giao dịch chuyển tiền (như cũ)
				if tx.Verify(prevTxs) {
					log.Printf("Miner: TX Transfer hợp lệ: %x", tx.ID)
					validTxs = append(validTxs, &tx)
				} else {
					log.Printf("Miner: Phát hiện TX Transfer không hợp lệ: %x", tx.ID)
				}

			case domain.TxTypeContractDeploy:
				// Giao dịch Triển khai Contract
				if tx.Verify(prevTxs) {
					log.Printf("Miner: TX Deploy hợp lệ: %x", tx.ID)

					// Lấy sender
					senderPubKeyHash := domain.HashPubKey(tx.Vin[0].PublicKey)
					contractAddress := tx.ID // ID của tx chính là địa chỉ contract

					// THỰC THI VM NGAY BÂY GIỜ
					err := executeVM(bc, tx.Payload, contractAddress, senderPubKeyHash, "", nil)

					if err != nil {
						// VM Thất bại!
						log.Printf("Miner: LỖI VM (Deploy %x): %v. Giao dịch bị TỪ CHỐI.", tx.ID, err)
						// Không thêm vào validTxs
					} else {
						// VM Thành công!
						log.Println("Miner: VM Deploy thành công. Đang lưu code...")
						// Lưu code vào CSDL
						if err := bc.SetContractCode(tx.ID, tx.Payload); err != nil {
							log.Printf("Miner: LỖI không lưu được code: %v. Bỏ qua TX.", err)
						} else {
							// Chỉ thêm vào block nếu VM chạy VÀ code lưu thành công
							validTxs = append(validTxs, &tx)
						}
					}
				} else {
					log.Printf("Miner: Phát hiện TX Deploy không hợp lệ: %x", tx.ID)
				}

			case domain.TxTypeContractCall:
				// Giao dịch Gọi Contract
				if tx.Verify(prevTxs) {
					log.Printf("Miner: TX Call hợp lệ: %x", tx.ID)

					// 1. Giải mã Payload (JSON)
					payload, err := vm.ParseCallPayload(tx.Payload)
					if err != nil {
						log.Printf("Miner: LỖI Payload Call: %v. Từ chối TX.", err)
						continue // Bỏ qua, TX này sẽ bị dọn dẹp
					}

					// 2. Lấy địa chỉ contract (dạng hex) và chuyển về bytes
					contractAddressBytes, err := hex.DecodeString(payload.ContractAddress)
					if err != nil {
						log.Printf("Miner: LỖI Địa chỉ Contract: %v. Từ chối TX.", err)
						continue
					}

					// 3. Lấy code của contract đó từ CSDL
					code, err := bc.GetContractCode(contractAddressBytes)
					if err != nil {
						log.Printf("Miner: LỖI không tìm thấy code contract: %v. Từ chối TX.", err)
						continue
					}

					// 4. Lấy địa chỉ người gửi
					senderPubKeyHash := domain.HashPubKey(tx.Vin[0].PublicKey)

					// 5. Chuyển đổi tham số
					luaArgs := vm.ConvertArgsToLValues(payload.Args)

					// 6. THỰC THI VM
					err = executeVM(bc, code, contractAddressBytes, senderPubKeyHash, payload.FunctionName, luaArgs)

					if err != nil {
						// VM Thất bại!
						log.Printf("Miner: LỖI VM (Call %x): %v. Giao dịch bị TỪ CHỐI.", tx.ID, err)
					} else {
						// VM Thành công!
						log.Printf("Miner: VM Call (%s) thành công.", payload.FunctionName)
						validTxs = append(validTxs, &tx)
					}
				} else {
					log.Printf("Miner: Phát hiện TX Call không hợp lệ: %x", tx.ID)
				}
			}
			// --- HẾT LOGIC MỚI ---
		}

		// 3. Tạo giao dịch thưởng (Coinbase)
		coinbaseTx := domain.NewCoinbaseTransaction(minerAddress, rewardAmount)
		// (Tạm thời bỏ coinbase để test VM)
		// allTxs := append([]*domain.Transaction{coinbaseTx}, validTxs...)
		allTxs := validTxs // CHỈ CÓ CÁC TX ĐÃ XÁC THỰC

		// NẾU KHÔNG CÓ GIAO DỊCH HỢP LỆ (NGOẠI TRỪ COINBASE)
		if len(allTxs) == 0 {
			log.Println("Miner: Không có TX hợp lệ để đào.")
			// Vẫn phải dọn dẹp mempool (nếu có TX rác)
			// (Logic dọn dẹp ở cuối đã lo việc này)
			continue // Bỏ qua vòng đào này
		}

		allTxs = append([]*domain.Transaction{coinbaseTx}, allTxs...)

		// 4. (MỚI) Thực thi VM cho các giao dịch Contract
		// (Phải làm trước khi AddBlock)
		for _, tx := range allTxs {
			// Địa chỉ "contract" chính là ID của giao dịch Deploy
			// Địa chỉ "sender" chính là PubKeyHash của người gọi

			// Lấy địa chỉ người gửi (Input đầu tiên)
			senderPubKeyHash := domain.HashPubKey(tx.Vin[0].PublicKey)

			if tx.Type == domain.TxTypeContractDeploy {
				// Thực thi code Deploy
				// Địa chỉ contract chính là ID của chính TX này
				contractAddress := tx.ID
				err := executeVM(bc, tx.Payload, contractAddress, senderPubKeyHash, "", nil)
				if err != nil {
					log.Printf("Miner: LỖI VM (Deploy %x): %v", tx.ID, err)
					// TODO: Xử lý giao dịch thất bại
				}
			}

			if tx.Type == domain.TxTypeContractCall {
				// TODO: Xử lý Call
				// 1. Lấy địa chỉ contract từ tx.Payload (chúng ta sẽ thiết kế sau)
				// 2. Lấy code của contract đó từ CSDL
				// 3. Gọi executeVM(...)
				log.Println("Miner: (TODO) Xử lý Contract Call")
			}
		}

		// 4. Đào block (Hàm này đã bao gồm PoW và cập nhật UTXO Set)
		bc.AddBlock(allTxs)

		log.Printf("Miner: === 🚀 ĐÀO THÀNH CÔNG BLOCK MỚI! ===")

		// 5. Dọn dẹp Mempool (Xóa các TX đã xử lý)
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

// (Dán hàm này vào cuối file network/miner.go)

// executeVM là hàm helper để khởi tạo và chạy VM
func executeVM(bc *domain.Blockchain, code []byte, contractAddress []byte, senderAddress []byte, functionName string, args []lua.LValue) error {

	v := vm.NewVM() // Tạo VM mới
	defer v.Close() // Đảm bảo đóng VM

	// Tiêm "syscalls" (db_put, db_get,...)
	v.RegisterBridgeFunctions()

	// Tiêm context (blockchain, địa chỉ, người gửi)
	v.SetContext(bc, contractAddress, senderAddress)

	if functionName == "" {
		// Đây là giao dịch DEPLOY
		return v.RunContractDeploy(code)
	} else {
		// Đây là giao dịch CALL
		return v.RunContractCall(code, functionName, args)
	}
}
