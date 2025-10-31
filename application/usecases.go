package application

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/khoahotran/gochain-ledger/network"
	"github.com/khoahotran/gochain-ledger/proto"
	"github.com/khoahotran/gochain-ledger/vm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/wallet"
)

func CreateWalletUseCase(password string) string {

	w := domain.NewWallet()
	address := w.GetAddress()

	encryptedKey, salt, err := wallet.EncryptKey(&w.PrivateKey, password)
	if err != nil {
		log.Panicf("Không thể mã hóa key: %v", err)
	}

	wf := &wallet.WalletFile{
		Address:      address,
		PublicKey:    w.PublicKey,
		EncryptedKey: encryptedKey,
		Salt:         salt,
	}

	if err := wf.Save(); err != nil {
		log.Panicf("Không thể lưu file ví: %v", err)
	}

	fmt.Printf("Tạo ví mới thành công!\n")
	fmt.Printf("Đã lưu vào: wallets/%s.json\n", address)
	fmt.Printf("Address: %s\n", address)

	return address
}

func InitChainUseCase(address string) {
	if !domain.ValidateAddress(address) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}
	bc := domain.InitBlockchain(address)
	defer bc.Close()
	fmt.Println("Khởi tạo blockchain thành công!")
}

func SendUseCase(fromAddress, toAddress string, amount int64, wallet *domain.Wallet, targetNodeAddr string) {
	if !domain.ValidateAddress(fromAddress) || !domain.ValidateAddress(toAddress) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}

	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối node: %v", err)
	}
	defer conn.Close()
	client := proto.NewNodeServiceClient(conn)

	req := &proto.FindSpendableUTXOsRequest{
		Address: fromAddress,
		Amount:  amount,
	}
	res, err := client.FindSpendableUTXOs(context.Background(), req)
	if err != nil {

		log.Panicf("Lỗi khi tìm UTXO: %v", err)
	}

	var inputs []domain.TxInput
	var outputs []domain.TxOutput

	fakePrevTxs := make(map[string]domain.Transaction)

	pubKeyHash := domain.HashPubKey(wallet.PublicKey)

	for _, utxo := range res.Utxos {
		input := domain.TxInput{
			TxID:      utxo.TxId,
			VoutIndex: int(utxo.VoutIndex),
			Signature: nil,
			PublicKey: wallet.PublicKey,
		}
		inputs = append(inputs, input)

		txIDStr := string(utxo.TxId)
		if _, ok := fakePrevTxs[txIDStr]; !ok {

			voutSlice := make([]domain.TxOutput, utxo.VoutIndex+1)
			voutSlice[utxo.VoutIndex] = domain.TxOutput{
				Value:      utxo.Amount,
				PubKeyHash: utxo.PubKeyHash,
			}
			fakePrevTxs[txIDStr] = domain.Transaction{ID: utxo.TxId, Vout: voutSlice}
		}
	}

	outputs = append(outputs, domain.TxOutput{Value: amount, PubKeyHash: domain.DecodeAddress(toAddress)})
	if res.AccumulatedAmount > amount {

		outputs = append(outputs, domain.TxOutput{Value: res.AccumulatedAmount - amount, PubKeyHash: pubKeyHash})
	}

	tx := domain.Transaction{
		ID:   nil,
		Vin:  inputs,
		Vout: outputs,

		Type:    domain.TxTypeTransfer,
		Payload: nil,
	}
	tx.SetID()
	tx.Sign(wallet.PrivateKey, fakePrevTxs)

	log.Printf("Đã tạo và ký giao dịch: %x", tx.ID)

	network.SendTransactionToNode(targetNodeAddr, &tx)

	fmt.Println("Gửi giao dịch thành công (đã vào Mempool)!")
}

func NewUTXOTransaction(wallet *domain.Wallet, toAddress string, amount int64, u *domain.UTXOSet) (*domain.Transaction, error) {
	pubKeyHash := domain.HashPubKey(wallet.PublicKey)

	acc, spendableOutputs := u.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		return nil, errors.New("hông đủ tiền")
	}

	var inputs []domain.TxInput
	var outputs []domain.TxOutput

	for txIDStr, outIndexes := range spendableOutputs {
		txID, err := hex.DecodeString(txIDStr)
		if err != nil {

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

	outputs = append(outputs, domain.TxOutput{Value: amount, PubKeyHash: domain.DecodeAddress(toAddress)})
	if acc > amount {

		outputs = append(outputs, domain.TxOutput{Value: acc - amount, PubKeyHash: pubKeyHash})
	}

	tx := domain.Transaction{ID: nil, Vin: inputs, Vout: outputs}
	tx.SetID()

	prevTxs := u.Blockchain.FindReferencedTxs(&tx)
	tx.Sign(wallet.PrivateKey, prevTxs)

	return &tx, nil
}

func DeployContractUseCase(fromAddress string, code []byte, wallet *domain.Wallet, targetNodeAddr string) {
	if !domain.ValidateAddress(fromAddress) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}

	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối node: %v", err)
	}
	defer conn.Close()
	client := proto.NewNodeServiceClient(conn)

	req := &proto.FindSpendableUTXOsRequest{
		Address: fromAddress,
		Amount:  1,
	}
	res, err := client.FindSpendableUTXOs(context.Background(), req)
	if err != nil {
		log.Panicf("Lỗi khi tìm UTXO: %v", err)
	}

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

	var outputs []domain.TxOutput
	outputs = append(outputs, domain.TxOutput{Value: res.AccumulatedAmount, PubKeyHash: pubKeyHash})

	tx := domain.Transaction{
		ID:      nil,
		Vin:     inputs,
		Vout:    outputs,
		Type:    domain.TxTypeContractDeploy,
		Payload: code,
	}
	tx.SetID()
	tx.Sign(wallet.PrivateKey, fakePrevTxs)

	log.Printf("Đã tạo và ký TX Deploy: %x", tx.ID)
	log.Printf("Địa chỉ Contract sẽ là: %x", tx.ID)

	network.SendTransactionToNode(targetNodeAddr, &tx)
	fmt.Println("Gửi TX Deploy thành công (đã vào Mempool)!")
}

func CallContractUseCase(fromAddress string, contractAddress string, functionName string, args []interface{}, wallet *domain.Wallet, targetNodeAddr string) {
	if !domain.ValidateAddress(fromAddress) {
		log.Panic("LỖI: Địa chỉ ví không hợp lệ")
	}

	callPayload, err := vm.NewCallPayload(contractAddress, functionName, args)
	if err != nil {
		log.Panicf("Lỗi tạo payload: %v", err)
	}

	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối node: %v", err)
	}
	defer conn.Close()
	client := proto.NewNodeServiceClient(conn)

	req := &proto.FindSpendableUTXOsRequest{Address: fromAddress, Amount: 1}
	res, err := client.FindSpendableUTXOs(context.Background(), req)
	if err != nil {
		log.Panicf("Lỗi khi tìm UTXO: %v", err)
	}

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

	var outputs []domain.TxOutput
	outputs = append(outputs, domain.TxOutput{Value: res.AccumulatedAmount, PubKeyHash: pubKeyHash})

	tx := domain.Transaction{
		ID:      nil,
		Vin:     inputs,
		Vout:    outputs,
		Type:    domain.TxTypeContractCall,
		Payload: callPayload,
	}
	tx.SetID()
	tx.Sign(wallet.PrivateKey, fakePrevTxs)

	log.Printf("Đã tạo và ký TX Call: %x", tx.ID)

	network.SendTransactionToNode(targetNodeAddr, &tx)
	fmt.Println("Gửi TX Call thành công (đã vào Mempool)!")
}
