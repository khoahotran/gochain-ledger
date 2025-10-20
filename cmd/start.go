package cmd

import (
	"context" // IMPORT MỚI
	"fmt"
	"log"

	"github.com/go-redis/redis/v8" // IMPORT MỚI
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/network"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Khởi động node GoChain Ledger và bắt đầu lắng nghe",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		minerAddress, _ := cmd.Flags().GetString("miner") // LẤY FLAG MỚI

		if port == "" {
			Handle(fmt.Errorf("cần cung cấp cổng (flag --port)"))
		}

		log.Printf("Khởi động node trên cổng: %s\n", port)

		// 1. Tải blockchain
		bc := domain.ContinueBlockchain()
		defer bc.Close() // Đảm bảo DB được đóng khi thoát

		// 2. (MỚI) Kết nối Redis (chuyển ra đây)
		rdb := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
		_, err := rdb.Ping(context.Background()).Result()
		if err != nil {
			log.Fatalf("Không thể kết nối đến Redis: %v", err)
		}
		log.Println("Đã kết nối đến Redis (Mempool).")

		// 3. (MỚI) Khởi động Miner (nếu được yêu cầu)
		if minerAddress != "" {
			if !domain.ValidateAddress(minerAddress) {
				log.Panic("LỖI: Địa chỉ ví miner không hợp lệ")
			}
			log.Printf("Node đang khởi động ở chế độ MINER. Phần thưởng sẽ về: %s", minerAddress)
			// Chạy Miner trong một tiến trình nền (goroutine)
			go network.StartMiningLoop(bc, rdb, minerAddress)
		}

		// 4. Khởi động gRPC Server (chạy ở tiến trình chính)
		// Hàm này sẽ chạy mãi mãi (blocking)
		network.StartServer(port, bc, rdb)
	},
}

func init() {
	startCmd.Flags().String("port", "", "Cổng để node lắng nghe (ví dụ: 3000)")
	// FLAG MỚI
	startCmd.Flags().String("miner", "", "Bật chế độ Miner, gửi thưởng về địa chỉ ví này")
	rootCmd.AddCommand(startCmd)
}
