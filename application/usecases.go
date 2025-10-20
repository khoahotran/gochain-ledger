package application

import (
	// MỚI
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	// IMPORT MỚI
	"github.com/khoahotran/gochain-ledger/network" // MỚI
	"github.com/khoahotran/gochain-ledger/proto"   // IMPORT MỚI
	"github.com/khoahotran/gochain-ledger/vm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// MỚI
	// MỚI
	// MỚI

	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/wallet"
)

// CreateWalletUseCase (Cập nhật)
// Yêu cầu mật khẩu để mã hóa
func CreateWalletUseCase(password string) string {
	// 1. Tạo ví (keypair)
	w := domain.NewWallet()
	address := w.GetAddress()

	// 2. Mã hóa private key
	encryptedKey, salt, err := wallet.EncryptKey(&w.PrivateKey, password)
	if err != nil {
		log.Panicf("Không thể mã hóa key: %v", err)
	}

	// 3. Tạo struct WalletFile để lưu
	wf := &wallet.WalletFile{
		Address:      address,
		PublicKey:    w.PublicKey,
		EncryptedKey: encryptedKey,
		Salt:         salt,
	}

	// 4. Lưu vào file
	if err := wf.Save(); err != nil {
		log.Panicf("Không thể lưu file ví: %v", err)
	}

	fmt.Printf("Tạo ví mới thành công!\n")
	fmt.Printf("Đã lưu vào: wallets/%s.json\n", address)
	fmt.Printf("Address: %s\n", address)
	// KHÔNG CÒN IN PRIVATE KEY

	return address
}

// InitChainUseCase
func InitChainUseCase(address string) {
	if !domain.ValidateAddress(address) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}
	bc := domain.InitBlockchain(address)
	defer bc.Close()
	fmt.Println("Khởi tạo blockchain thành công!")
}

// (Trong file application/usecases.go)

// SendUseCase (ĐÃ REFACTOR HOÀN TOÀN)
// Giờ sẽ hoạt động như một gRPC Client
func SendUseCase(fromAddress, toAddress string, amount int64, wallet *domain.Wallet, targetNodeAddr string) {
	if !domain.ValidateAddress(fromAddress) || !domain.ValidateAddress(toAddress) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}

	// 1. Kết nối gRPC
	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối node: %v", err)
	}
	defer conn.Close()
	client := proto.NewNodeServiceClient(conn)

	// 2. (MỚI) Hỏi server để tìm UTXO
	req := &proto.FindSpendableUTXOsRequest{
		Address: fromAddress,
		Amount:  amount,
	}
	res, err := client.FindSpendableUTXOs(context.Background(), req)
	if err != nil {
		// Lỗi này có thể là "Không đủ tiền"
		log.Panicf("Lỗi khi tìm UTXO: %v", err)
	}

	// 3. (MỚI) Tái tạo logic của NewUTXOTransaction (phía client)
	var inputs []domain.TxInput
	var outputs []domain.TxOutput

	// Dùng để tạo map "fake" cho hàm Sign
	fakePrevTxs := make(map[string]domain.Transaction)

	pubKeyHash := domain.HashPubKey(wallet.PublicKey)

	// 4. Tạo Inputs (và map "fake")
	for _, utxo := range res.Utxos {
		input := domain.TxInput{
			TxID:      utxo.TxId,
			VoutIndex: int(utxo.VoutIndex),
			Signature: nil,
			PublicKey: wallet.PublicKey,
		}
		inputs = append(inputs, input)

		// (Đây là phần "hacky" để tạo map prevTxs)
		txIDStr := string(utxo.TxId)
		if _, ok := fakePrevTxs[txIDStr]; !ok {
			// Tạo một Vout slice đủ lớn
			voutSlice := make([]domain.TxOutput, utxo.VoutIndex+1)
			voutSlice[utxo.VoutIndex] = domain.TxOutput{
				Value:      utxo.Amount,
				PubKeyHash: utxo.PubKeyHash,
			}
			fakePrevTxs[txIDStr] = domain.Transaction{ID: utxo.TxId, Vout: voutSlice}
		}
	}

	// 5. Tạo Outputs
	outputs = append(outputs, domain.TxOutput{Value: amount, PubKeyHash: domain.DecodeAddress(toAddress)})
	if res.AccumulatedAmount > amount {
		// Tiền thối
		outputs = append(outputs, domain.TxOutput{Value: res.AccumulatedAmount - amount, PubKeyHash: pubKeyHash})
	}

	// 6. Tạo và Ký Giao dịch

	tx := domain.Transaction{
		ID:   nil,
		Vin:  inputs,
		Vout: outputs,
		// SỬA LẠI 2 DÒNG NÀY
		Type:    domain.TxTypeTransfer,
		Payload: nil,
	}
	tx.SetID()
	tx.Sign(wallet.PrivateKey, fakePrevTxs)

	log.Printf("Đã tạo và ký giao dịch: %x", tx.ID)

	// 7. (Cũ) Gửi giao dịch đến node
	network.SendTransactionToNode(targetNodeAddr, &tx)

	fmt.Println("Gửi giao dịch thành công (đã vào Mempool)!")
}

// NewUTXOTransaction (Hàm nội bộ của usecase)
func NewUTXOTransaction(wallet *domain.Wallet, toAddress string, amount int64, u *domain.UTXOSet) (*domain.Transaction, error) {
	pubKeyHash := domain.HashPubKey(wallet.PublicKey)

	// 1. Tìm UTXO có thể tiêu
	acc, spendableOutputs := u.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		return nil, errors.New("hông đủ tiền")
	}

	var inputs []domain.TxInput
	var outputs []domain.TxOutput

	// 2. Tạo Inputs
	for txIDStr, outIndexes := range spendableOutputs {
		txID, err := hex.DecodeString(txIDStr) // Badger key là hex string
		if err != nil {
			// Nếu key không phải hex, nó là bytes
			txID = []byte(txIDStr)
		}

		for _, outIdx := range outIndexes {
			input := domain.TxInput{
				TxID:      txID,
				VoutIndex: outIdx,
				Signature: nil,
				PublicKey: wallet.PublicKey,
			}
			inputs = append(inputs, input)
		}
	}

	// 3. Tạo Outputs (1 cho người nhận, 1 cho tiền thối)
	outputs = append(outputs, domain.TxOutput{Value: amount, PubKeyHash: domain.DecodeAddress(toAddress)})
	if acc > amount {
		// Tiền thối
		outputs = append(outputs, domain.TxOutput{Value: acc - amount, PubKeyHash: pubKeyHash})
	}

	tx := domain.Transaction{ID: nil, Vin: inputs, Vout: outputs}
	tx.SetID()

	// 4. Ký Giao dịch
	// Chúng ta cần lấy các transaction cũ mà input tham chiếu đến
	prevTxs := u.Blockchain.FindReferencedTxs(&tx)
	tx.Sign(wallet.PrivateKey, prevTxs)

	return &tx, nil
}

// DeployContractUseCase xử lý logic deploy contract
func DeployContractUseCase(fromAddress string, code []byte, wallet *domain.Wallet, targetNodeAddr string) {
	if !domain.ValidateAddress(fromAddress) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}

	// 1. Kết nối gRPC
	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối node: %v", err)
	}
	defer conn.Close()
	client := proto.NewNodeServiceClient(conn)

	// 2. Hỏi server để tìm UTXO (chỉ cần 1 UTXO nhỏ nhất để "trả phí")
	// (Chúng ta tái sử dụng logic tìm UTXO của 'send' với amount = 1)
	req := &proto.FindSpendableUTXOsRequest{
		Address: fromAddress,
		Amount:  1, // Cần ít nhất 1 (đơn vị nhỏ nhất) để tạo TX
	}
	res, err := client.FindSpendableUTXOs(context.Background(), req)
	if err != nil {
		log.Panicf("Lỗi khi tìm UTXO: %v", err)
	}

	// 3. Tạo Inputs
	var inputs []domain.TxInput
	fakePrevTxs := make(map[string]domain.Transaction)
	pubKeyHash := domain.HashPubKey(wallet.PublicKey)
	for _, utxo := range res.Utxos {
		inputs = append(inputs, domain.TxInput{
			TxID: utxo.TxId, VoutIndex: int(utxo.VoutIndex), Signature: nil, PublicKey: wallet.PublicKey,
		})
		txIDStr := string(utxo.TxId)
		if _, ok := fakePrevTxs[txIDStr]; !ok {
			voutSlice := make([]domain.TxOutput, utxo.VoutIndex+1)
			voutSlice[utxo.VoutIndex] = domain.TxOutput{Value: utxo.Amount, PubKeyHash: utxo.PubKeyHash}
			fakePrevTxs[txIDStr] = domain.Transaction{ID: utxo.TxId, Vout: voutSlice}
		}
	}

	// 4. Tạo Outputs (Chỉ có tiền thối)
	var outputs []domain.TxOutput
	outputs = append(outputs, domain.TxOutput{Value: res.AccumulatedAmount, PubKeyHash: pubKeyHash}) // Trả lại toàn bộ

	// 5. Tạo TX
	tx := domain.Transaction{
		ID:      nil,
		Vin:     inputs,
		Vout:    outputs,
		Type:    domain.TxTypeContractDeploy,
		Payload: code, // Payload là code Lua
	}
	tx.SetID()
	tx.Sign(wallet.PrivateKey, fakePrevTxs)

	log.Printf("Đã tạo và ký TX Deploy: %x", tx.ID)
	log.Printf("Địa chỉ Contract sẽ là: %x", tx.ID)

	// 6. Gửi TX
	network.SendTransactionToNode(targetNodeAddr, &tx)
	fmt.Println("Gửi TX Deploy thành công (đã vào Mempool)!")
}

// CallContractUseCase xử lý logic gọi hàm contract
func CallContractUseCase(fromAddress string, contractAddress string, functionName string, args []interface{}, wallet *domain.Wallet, targetNodeAddr string) {
	if !domain.ValidateAddress(fromAddress) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}

	// 1. Tạo Payload (JSON)
	callPayload, err := vm.NewCallPayload(contractAddress, functionName, args)
	if err != nil {
		log.Panicf("Lỗi tạo payload: %v", err)
	}

	// 2. Kết nối gRPC
	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối node: %v", err)
	}
	defer conn.Close()
	client := proto.NewNodeServiceClient(conn)

	// 3. Tìm UTXO (tương tự Deploy, chỉ cần 1 UTXO nhỏ)
	req := &proto.FindSpendableUTXOsRequest{Address: fromAddress, Amount: 1}
	res, err := client.FindSpendableUTXOs(context.Background(), req)
	if err != nil {
		log.Panicf("Lỗi khi tìm UTXO: %v", err)
	}

	// 4. Tạo Inputs
	var inputs []domain.TxInput
	fakePrevTxs := make(map[string]domain.Transaction)
	pubKeyHash := domain.HashPubKey(wallet.PublicKey)
	for _, utxo := range res.Utxos {
		inputs = append(inputs, domain.TxInput{
			TxID: utxo.TxId, VoutIndex: int(utxo.VoutIndex), Signature: nil, PublicKey: wallet.PublicKey,
		})
		txIDStr := string(utxo.TxId)
		if _, ok := fakePrevTxs[txIDStr]; !ok {
			voutSlice := make([]domain.TxOutput, utxo.VoutIndex+1)
			voutSlice[utxo.VoutIndex] = domain.TxOutput{Value: utxo.Amount, PubKeyHash: utxo.PubKeyHash}
			fakePrevTxs[txIDStr] = domain.Transaction{ID: utxo.TxId, Vout: voutSlice}
		}
	}

	// 5. Tạo Outputs (Chỉ có tiền thối)
	var outputs []domain.TxOutput
	outputs = append(outputs, domain.TxOutput{Value: res.AccumulatedAmount, PubKeyHash: pubKeyHash})

	// 6. Tạo TX
	tx := domain.Transaction{
		ID:      nil,
		Vin:     inputs,
		Vout:    outputs,
		Type:    domain.TxTypeContractCall,
		Payload: callPayload, // Payload là JSON
	}
	tx.SetID()
	tx.Sign(wallet.PrivateKey, fakePrevTxs)

	log.Printf("Đã tạo và ký TX Call: %x", tx.ID)

	// 7. Gửi TX
	network.SendTransactionToNode(targetNodeAddr, &tx)
	fmt.Println("Gửi TX Call thành công (đã vào Mempool)!")
}
