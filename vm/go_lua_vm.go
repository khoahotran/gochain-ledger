package vm

import (
	"context"
	"fmt"
	"log"

	"github.com/khoahotran/gochain-ledger/domain"
	lua "github.com/yuin/gopher-lua"
)

// VM là lớp bao bọc (wrapper) cho trạng thái Lua (LState)
type VM struct {
	L *lua.LState
}

// ContextKey là kiểu dữ liệu an toàn để lưu trữ giá trị trong LState
type ContextKey string

const (
	// Khóa để lưu trữ các giá trị "toàn cục" của blockchain trong VM
	ctxBlockchainKey      ContextKey = "blockchain"
	ctxContractAddressKey ContextKey = "contract_address"
	ctxSenderAddressKey   ContextKey = "sender_address"
)

// NewVM tạo một máy ảo Lua mới và an toàn
func NewVM() *VM {
	// 1. Tạo một LState mới
	L := lua.NewState()

	// 2. (QUAN TRỌNG) Mở TỪNG thư viện an toàn một
	// Đây là các hàm của package 'lua', KHÔNG phải method của L
	lua.OpenBase(L)   // Các hàm cơ bản (print, pcall, error, ...)
	lua.OpenTable(L)  // Thao tác với bảng (table.*)
	lua.OpenString(L) // Thao tác chuỗi (string.*)
	lua.OpenMath(L)   // Toán học (math.*)

	// Chúng ta cố ý KHÔNG mở:
	// lua.OpenIo(L)
	// lua.OpenOs(L)
	// lua.OpenModule(L)
	// lua.OpenGo(L) (Rất nguy hiểm)

	return &VM{L: L}
}

// Close đóng LState để giải phóng bộ nhớ
func (v *VM) Close() {
	v.L.Close()
}

// SetContext tiêm (inject) các đối tượng Go vào LState
// Code Lua sẽ cần truy cập các đối tượng này
// (Trong file vm/go_lua_vm.go)

func (v *VM) SetContext(bc *domain.Blockchain, contractAddress []byte, senderAddress []byte) {
	// 1. Tạo một context
	ctx := context.Background()

	// 2. Thêm TẤT CẢ các giá trị vào context
	ctx = context.WithValue(ctx, ctxBlockchainKey, bc)
	ctx = context.WithValue(ctx, ctxContractAddressKey, contractAddress)
	ctx = context.WithValue(ctx, ctxSenderAddressKey, senderAddress)

	// 3. Đặt context MỘT LẦN
	v.L.SetContext(ctx)
}

// RegisterBridgeFunctions đăng ký các hàm Go (syscalls) vào môi trường Lua
func (v *VM) RegisterBridgeFunctions() {
	// Tiêm hàm "db_put"
	v.L.SetGlobal("db_put", v.L.NewFunction(luaDbPut))
	// Tiêm hàm "db_get"
	v.L.SetGlobal("db_get", v.L.NewFunction(luaDbGet))
	// Tiêm hàm "get_sender"
	v.L.SetGlobal("get_sender", v.L.NewFunction(luaGetSender))
}

// === Các hàm "Cầu nối" (Bridge Functions) ===
// Đây là các hàm Go sẽ được gọi TỪ bên trong code Lua

// luaDbPut là hàm Go cho 'db_put(key, value)'
func luaDbPut(L *lua.LState) int {
	// 1. Lấy tham số từ Lua stack
	key := L.ToString(1)
	value := L.ToString(2)

	// 2. Lấy context đã lưu (Blockchain và Địa chỉ Contract)
	ctx := L.Context()
	bc := ctx.Value(ctxBlockchainKey).(*domain.Blockchain)
	contractAddress := ctx.Value(ctxContractAddressKey).([]byte)

	// 3. Gọi hàm Go (lưu vào CSDL)
	err := bc.SetContractState(contractAddress, []byte(key), []byte(value))
	if err != nil {
		log.Printf("VM (db_put): Lỗi: %v", err)
		L.Push(lua.LBool(false)) // Trả về false (thất bại)
		return 1                 // Trả về 1 giá trị (boolean)
	}

	L.Push(lua.LBool(true)) // Trả về true (thành công)
	return 1
}

// luaDbGet là hàm Go cho 'db_get(key)'
func luaDbGet(L *lua.LState) int {
	// 1. Lấy tham số
	key := L.ToString(1)

	// 2. Lấy context
	ctx := L.Context()
	bc := ctx.Value(ctxBlockchainKey).(*domain.Blockchain)
	contractAddress := ctx.Value(ctxContractAddressKey).([]byte)

	// 3. Gọi hàm Go (đọc từ CSDL)
	value, err := bc.GetContractState(contractAddress, []byte(key))
	if err != nil {
		log.Printf("VM (db_get): Lỗi: %v", err)
		L.Push(lua.LNil) // Trả về nil (lỗi)
		return 1
	}

	if value == nil {
		L.Push(lua.LNil) // Trả về nil (không tìm thấy)
		return 1
	}

	L.Push(lua.LString(string(value))) // Trả về giá trị (dạng string)
	return 1
}

// luaGetSender là hàm Go cho 'get_sender()'
func luaGetSender(L *lua.LState) int {
	// 1. Lấy context
	ctx := L.Context()
	senderAddress := ctx.Value(ctxSenderAddressKey).([]byte)

	// 2. Trả về địa chỉ người gửi (dạng string)
	// (Chúng ta cần Base58 encode, tạm thời dùng hex)
	L.Push(lua.LString(fmt.Sprintf("%x", senderAddress)))
	return 1
}

// === Logic Thực thi ===

// RunContractDeploy thực thi code Lua (khi triển khai)
func (v *VM) RunContractDeploy(code []byte) error {
	// Đơn giản là chạy code
	// Nếu code có lỗi cú pháp, nó sẽ báo lỗi ở đây
	return v.L.DoString(string(code))
}

// RunContractCall gọi một hàm cụ thể bên trong code Lua
func (v *VM) RunContractCall(code []byte, functionName string, args []lua.LValue) error {
	// 1. Load lại code contract (để đảm bảo các hàm đã được định nghĩa)
	if err := v.L.DoString(string(code)); err != nil {
		return fmt.Errorf("lỗi khi load code: %v", err)
	}

	// 2. Lấy hàm từ môi trường Lua
	fn := v.L.GetGlobal(functionName)
	if fn.Type() == lua.LTNil {
		return fmt.Errorf("hàm '%s' không tồn tại trong contract", functionName)
	}

	// 3. Gọi hàm
	err := v.L.CallByParam(lua.P{
		Fn:      fn,   // Hàm để gọi
		NRet:    0,    // Không mong đợi giá trị trả về
		Protect: true, // Bắt lỗi (panic) nếu có
	}, args...) // Truyền các tham số (nếu có)

	if err != nil {
		return fmt.Errorf("lỗi khi thực thi hàm '%s': %v", functionName, err)
	}
	return nil
}
