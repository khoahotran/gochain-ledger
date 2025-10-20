package vm

import (
	"encoding/json"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// ContractCallPayload là struct để đóng gói/giải mã payload của TX Call
type ContractCallPayload struct {
	ContractAddress string        `json:"contract_address"` // Địa chỉ (ID) của contract
	FunctionName    string        `json:"function_name"`    // Tên hàm Lua để gọi
	Args            []interface{} `json:"args"`             // Các tham số
}

// NewCallPayload tạo payload (JSON bytes)
func NewCallPayload(contractAddr string, funcName string, args []interface{}) ([]byte, error) {
	payload := ContractCallPayload{
		ContractAddress: contractAddr,
		FunctionName:    funcName,
		Args:            args,
	}
	return json.Marshal(payload)
}

// ParseCallPayload giải mã payload
func ParseCallPayload(data []byte) (*ContractCallPayload, error) {
	var payload ContractCallPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return nil, fmt.Errorf("lỗi giải mã payload: %v", err)
	}
	return &payload, nil
}

// ConvertArgsToLValues (Helper)
// Chuyển đổi tham số từ Go (interface{}) sang kiểu của Lua (LValue)
func ConvertArgsToLValues(args []interface{}) []lua.LValue {
	lvalues := make([]lua.LValue, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			lvalues[i] = lua.LString(v)
		case float64: // JSON mặc định là float64 cho số
			lvalues[i] = lua.LNumber(v)
		case bool:
			lvalues[i] = lua.LBool(v)
		default:
			lvalues[i] = lua.LNil // Không hỗ trợ kiểu khác
		}
	}
	return lvalues
}
