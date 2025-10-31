package vm

import (
	"encoding/json"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type ContractCallPayload struct {
	ContractAddress string        `json:"contract_address"`
	FunctionName    string        `json:"function_name"`
	Args            []interface{} `json:"args"`
}

func NewCallPayload(contractAddr string, funcName string, args []interface{}) ([]byte, error) {
	payload := ContractCallPayload{
		ContractAddress: contractAddr,
		FunctionName:    funcName,
		Args:            args,
	}
	return json.Marshal(payload)
}

func ParseCallPayload(data []byte) (*ContractCallPayload, error) {
	var payload ContractCallPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return nil, fmt.Errorf("lỗi giải mã payload: %v", err)
	}
	return &payload, nil
}

func ConvertArgsToLValues(args []interface{}) []lua.LValue {
	lvalues := make([]lua.LValue, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			lvalues[i] = lua.LString(v)
		case float64:
			lvalues[i] = lua.LNumber(v)
		case bool:
			lvalues[i] = lua.LBool(v)
		default:
			lvalues[i] = lua.LNil
		}
	}
	return lvalues
}
