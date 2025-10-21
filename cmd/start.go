package cmd

import (
	"context" // IMPORT MỚI
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8" // IMPORT MỚI
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"

	// IMPORT MỚI
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/network"
	"github.com/khoahotran/gochain-ledger/proto"
	"github.com/spf13/cobra"
	// IMPORT MỚI
	// IMPORT MỚI
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Khởi động node GoChain Ledger và bắt đầu lắng nghe",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		grpcPort, _ := cmd.Flags().GetString("grpcport") // LẤY FLAG MỚI
		minerAddress, _ := cmd.Flags().GetString("miner") // LẤY FLAG MỚI

		if port == "" {
			Handle(fmt.Errorf("cần cung cấp cổng (flag --port)"))
		}
		log.Printf("Khởi động node...\n - Cổng gRPC-Web (DApp): %s\n - Cổng gRPC (P2P/CLI): %s", port, grpcPort)

		// 1. Tải blockchain
		bc := domain.ContinueBlockchain()
		// defer bc.Close() // Đảm bảo DB được đóng khi thoát

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
		// --- LOGIC SERVER MỚI (Bật gRPC-Web) ---

		// 1. Tạo gRPC server (như cũ)
		grpcServer := grpc.NewServer()

        nodeService := &network.Server{Blockchain: bc, RedisClient: rdb}
        publicService := &network.PublicServer{Blockchain: bc, RedisClient: rdb}

		// 2. Đăng ký CẢ HAI service (NodeService P2P và PublicService)
		// Node P2P (máy chủ khác gọi)
		proto.RegisterNodeServiceServer(grpcServer, nodeService)
		// Public API (DApp gọi)
		proto.RegisterPublicServiceServer(grpcServer, publicService)

		go func() {
            lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
            if err != nil {
                log.Fatalf("Không thể lắng nghe gRPC trên cổng %s: %v", grpcPort, err)
            }
            log.Printf("gRPC Server thuần túy đang lắng nghe tại %v", lis.Addr())
            if err := grpcServer.Serve(lis); err != nil {
                // Không dùng Fatalf ở đây vì nó sẽ dừng cả server HTTP
                log.Printf("gRPC Server thuần túy thất bại: %v", err)
            }
        }()

		// 3. Bọc (wrap) gRPC server bằng gRPC-Web Proxy
		wrappedGrpc := grpcweb.WrapServer(grpcServer,
			// Cấu hình CORS (cho phép React (localhost:5173) gọi)
			grpcweb.WithOriginFunc(func(origin string) bool {
				return true // Tạm thời cho phép tất cả
			}),
		)

		// 4. Tạo một HTTP server để phục vụ proxy
		httpServer := &http.Server{
			Addr: fmt.Sprintf(":%s", port),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Nếu request là gRPC-Web, proxy sẽ xử lý
				if wrappedGrpc.IsAcceptableGrpcCorsRequest(r) || wrappedGrpc.IsGrpcWebRequest(r) {
					wrappedGrpc.ServeHTTP(w, r)
					return
				}
				// (Tương lai: Có thể phục vụ API REST ở đây)
				http.NotFound(w, r)
			}),
			ReadHeaderTimeout: 5 * time.Second,
		}

		// 5. Chạy server
		log.Printf("gRPC & gRPC-Web server đang lắng nghe tại [::]:%s", port)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Server thất bại: %v", err)
		}

		// Đóng CSDL khi server dừng hẳn
        log.Println("Đang đóng CSDL...")
        bc.Close()
        log.Println("CSDL đã đóng.")
	},
}

func init() {
	startCmd.Flags().String("port", "", "Cổng để node lắng nghe (ví dụ: 3000)")
	// FLAG MỚI
	startCmd.Flags().String("miner", "", "Bật chế độ Miner, gửi thưởng về địa chỉ ví này")
	startCmd.Flags().String("grpcport", "50051", "Cổng gRPC thuần túy (cho P2P/CLI)")
	rootCmd.AddCommand(startCmd)
}
