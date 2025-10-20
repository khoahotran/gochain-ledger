package network

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/khoahotran/gochain-ledger/domain"
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

			// Xác thực chữ ký
			if tx.Verify(prevTxs) {
				log.Printf("Miner: TX hợp lệ: %x", tx.ID)
				validTxs = append(validTxs, &tx)
			} else {
				log.Printf("Miner: Phát hiện TX không hợp lệ: %x. Đang xóa...", tx.ID)
				// Không cần làm gì thêm, nó sẽ bị xóa ở bước 5
			}
		}

		// 3. Tạo giao dịch thưởng (Coinbase)
		coinbaseTx := domain.NewCoinbaseTransaction(minerAddress, rewardAmount)
		allTxs := append([]*domain.Transaction{coinbaseTx}, validTxs...)

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
