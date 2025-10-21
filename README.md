# GoChain Ledger ⛓️

[![Go Version](https://img.shields.io/badge/go-1.18%2B-blue.svg)](https://golang.org/)

Một nền tảng blockchain private được xây dựng từ đầu bằng **Golang**, lấy cảm hứng từ kiến trúc của Bitcoin (UTXO, PoW) và Ethereum (Smart Contract). Dự án này phục vụ mục đích học tập và trình diễn các khái niệm cốt lõi của công nghệ blockchain.

**➡️ Frontend DApp tương ứng:** [**gochain-frontend**](https://github.com/khoahotran/gochain-frontend)

---

## ✨ Tính năng chính

* **Lõi Blockchain:**
    * Mô hình **UTXO** (Unspent Transaction Output) giống Bitcoin.
    * Cơ chế đồng thuận **Proof-of-Work (PoW)** đơn giản.
    * Lưu trữ dữ liệu bền bỉ bằng **BadgerDB** (Key-Value Store).
* **Mạng P2P:**
    * Giao tiếp giữa các node sử dụng **gRPC**.
    * **Mempool** (Transaction Pool) sử dụng **Redis** để chia sẻ giao dịch chờ.
    * Miner tự động lấy giao dịch từ Mempool và đào block mới.
* **Smart Contract (Hợp đồng thông minh):**
    * Tích hợp Máy ảo **Lua (Gopher-Lua)** để thực thi logic tùy chỉnh.
    * Hỗ trợ triển khai (Deploy) và gọi (Call) các hàm trong contract.
    * Lưu trữ trạng thái (State) của contract trong CSDL.
* **Quản lý Ví:**
    * Tạo và quản lý cặp khóa ECDSA (đường cong P256).
    * Địa chỉ ví mã hóa **Base58Check**.
    * Lưu trữ ví an toàn bằng cách **mã hóa Private Key** với mật khẩu (AES + Scrypt) và lưu vào file JSON.
* **Giao diện Dòng lệnh (CLI):**
    * Xây dựng bằng **Cobra**.
    * Các lệnh: `init`, `createwallet`, `start` (chế độ server/miner), `balance`, `send`, `deploy`, `call`, `read`.
* **Hỗ trợ Frontend:**
    * Tích hợp **gRPC-Web Proxy** để cho phép DApp (React) tương tác trực tiếp với node.

---

## 🛠️ Công nghệ sử dụng

* **Ngôn ngữ:** Go
* **CSDL:** BadgerDB (Blockchain & State), Redis (Mempool)
* **Mạng:** gRPC, Protocol Buffers
* **CLI:** Cobra
* **VM:** Gopher-Lua
* **Crypto:** `crypto/ecdsa`, `crypto/sha256`, `golang.org/x/crypto/scrypt`, `crypto/aes`
* **Encoding:** `encoding/json`, `encoding/gob`, `github.com/mr-tron/base58`
* **Proxy:** `github.com/improbable-eng/grpc-web/go/grpcweb`

---

## 🚀 Bắt đầu

### Chuẩn bị môi trường

1.  **Cài đặt Go:** Phiên bản 1.18 trở lên.
2.  **Cài đặt `protoc`:** Trình biên dịch Protocol Buffers (xem [hướng dẫn](https://grpc.io/docs/protoc-installation/)).
3.  **Cài đặt Go plugins cho `protoc`:**
    ```bash
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    ```
4.  **Cài đặt và chạy Redis Server:** (Xem [hướng dẫn](https://redis.io/docs/getting-started/installation/)). Đảm bảo Redis chạy trên `localhost:6379`.

### Chạy dự án

1.  **Clone repository:**
    ```bash
    git clone https://github.com/khoahotran/gochain-ledger.git
    cd gochain-ledger
    ```
2.  **Tải dependencies:**
    ```bash
    go mod tidy
    ```
3.  **Biên dịch code Proto:**
    ```bash
    # Biên dịch cho Go
    protoc --go_out=. --go-grpc_out=. proto/blockchain.proto proto/public.proto
    # Biên dịch cho Frontend
    protoc \
     --plugin=protoc-gen-ts=../gochain-frontend/node_modules/.bin/protoc-gen-ts \
     --ts_out=client=grpc-web,mode=grpc-web-text:../gochain-frontend/src/proto \
     proto/blockchain.proto proto/public.proto
    ```
4.  **Biên dịch ứng dụng CLI:**
    ```bash
    go build -o gochain-cli
    ```
5.  **Sử dụng CLI:**

    * **Tạo ví đầu tiên (Quan trọng: Lưu lại địa chỉ và đặt mật khẩu):**
        ```bash
        ./gochain-cli createwallet
        ```
        *(File ví `.json` sẽ được lưu trong thư mục `wallets/`)*

    * **Khởi tạo Blockchain (CHẠY MỘT LẦN DUY NHẤT):**
        ```bash
        ./gochain-cli init --address <ĐỊA_CHỈ_VÍ_BẠN_VỪA_TẠO>
        ```

    * **Khởi động Node (Server + Miner):**
        ```bash
        # Chạy ở chế độ Miner, phần thưởng sẽ về ví của bạn
        ./gochain-cli start --miner <ĐỊA_CHỈ_VÍ_CỦA_BẠN>
        # Node sẽ lắng nghe gRPC-Web trên cổng 3000 và gRPC thuần túy trên 50051
        ```
        *(Để node này chạy trong một cửa sổ terminal riêng)*

    * **Kiểm tra số dư (Terminal khác):**
        ```bash
        ./gochain-cli balance --address <ĐỊA_CHỈ_VÍ>
        # (Mặc định kết nối tới localhost:50051)
        ```

    * **Gửi tiền (Terminal khác):**
        ```bash
        ./gochain-cli send --from <VÍ_GỬI> --to <VÍ_NHẬN> --amount <SỐ_TIỀN>
        # Sẽ yêu cầu nhập mật khẩu của ví gửi
        ```

    * **Triển khai Smart Contract (Terminal khác):**
        ```bash
        # Ví dụ với file counter.lua
        ./gochain-cli deploy --from <VÍ_CỦA_BẠN> --file ./counter.lua
        # Ghi lại địa chỉ Contract (là ID của giao dịch)
        ```

    * **Gọi hàm Smart Contract (Terminal khác):**
        ```bash
        # Ví dụ gọi hàm increment() trên contract counter
        ./gochain-cli call --from <VÍ_CỦA_BẠN> --contract <ĐỊA_CHỈ_CONTRACT> --function "increment" --args "[]"
        ```

    * **Đọc trạng thái Smart Contract (Terminal khác):**
        ```bash
        # Ví dụ đọc key "counter"
        ./gochain-cli read --contract <ĐỊA_CHỈ_CONTRACT> --key "counter"
        ```

---

## 🏗️ Cấu trúc Dự án

* **`cmd/`**: Mã nguồn cho các lệnh CLI (Cobra).
* **`domain/`**: Các cấu trúc dữ liệu cốt lõi (Block, Transaction, Wallet...) và logic nghiệp vụ cơ bản (PoW, UTXO).
* **`network/`**: Logic xử lý mạng P2P (gRPC Server, Client, Miner) và Public API (gRPC-Web).
* **`application/`**: Các Use Cases điều phối hoạt động giữa các lớp.
* **`wallet/`**: Logic mã hóa, lưu trữ và tải file ví.
* **`vm/`**: Máy ảo Lua (Gopher-Lua) và "cầu nối" (bridge) với Go.
* **`proto/`**: Các file định nghĩa Protocol Buffers (`.proto`) và code Go được tạo ra.
* **`main.go`**: Điểm vào của ứng dụng CLI.
* **`tmp/blocks/`**: Thư mục chứa CSDL BadgerDB.
* **`wallets/`**: Thư mục chứa các file ví `.json` đã mã hóa.

---

## 🏛️ Kiến trúc

Dự án tuân theo kiến trúc Client-Server rõ ràng:
* Chỉ có tiến trình **`start`** mới được phép ghi vào CSDL BadgerDB.
* Tất cả các lệnh CLI khác (`send`, `balance`, `deploy`...) hoạt động như các **client**, gửi yêu cầu đến node đang chạy qua **gRPC thuần túy** (mặc định cổng 50051).
* **Frontend DApp** cũng là client, gửi yêu cầu qua **gRPC-Web** (mặc định cổng 3000). Server `start` chạy một proxy tích hợp để xử lý các request này.

Việc đồng bộ hóa hashing giữa Go (backend) và JavaScript (frontend) cho việc ký/xác thực giao dịch được thực hiện bằng cách sử dụng **JSON serialization** (với `[]byte` được encode Base64 và các key được sắp xếp) ở cả hai phía khi tính toán hash để ký.

---

## 📄 License

[MIT](LICENSE)

---

`File được tạo ra bởi AI  nếu có bất kỳ thắc mắc gì, vui lòng không hỏi chủ sở hữu :))`
