package vm

import (
	"context"
	"fmt"
	"log"

	"github.com/khoahotran/gochain-ledger/domain"
	lua "github.com/yuin/gopher-lua"
)

type VM struct {
	L *lua.LState
}

type ContextKey string

const (
	ctxBlockchainKey      ContextKey = "blockchain"
	ctxContractAddressKey ContextKey = "contract_address"
	ctxSenderAddressKey   ContextKey = "sender_address"
)

func NewVM() *VM {

	L := lua.NewState()

	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)

	return &VM{L: L}
}

func (v *VM) Close() {
	v.L.Close()
}

func (v *VM) SetContext(bc *domain.Blockchain, contractAddress []byte, senderAddress []byte) {

	ctx := context.Background()

	ctx = context.WithValue(ctx, ctxBlockchainKey, bc)
	ctx = context.WithValue(ctx, ctxContractAddressKey, contractAddress)
	ctx = context.WithValue(ctx, ctxSenderAddressKey, senderAddress)

	v.L.SetContext(ctx)
}

func (v *VM) RegisterBridgeFunctions() {

	v.L.SetGlobal("db_put", v.L.NewFunction(luaDbPut))

	v.L.SetGlobal("db_get", v.L.NewFunction(luaDbGet))

	v.L.SetGlobal("get_sender", v.L.NewFunction(luaGetSender))
}

func luaDbPut(L *lua.LState) int {

	key := L.ToString(1)
	value := L.ToString(2)

	ctx := L.Context()
	bc := ctx.Value(ctxBlockchainKey).(*domain.Blockchain)
	contractAddress := ctx.Value(ctxContractAddressKey).([]byte)

	err := bc.SetContractState(contractAddress, []byte(key), []byte(value))
	if err != nil {
		log.Printf("VM (db_put): Lỗi: %v", err)
		L.Push(lua.LBool(false))
		return 1
	}

	L.Push(lua.LBool(true))
	return 1
}

func luaDbGet(L *lua.LState) int {

	key := L.ToString(1)

	ctx := L.Context()
	bc := ctx.Value(ctxBlockchainKey).(*domain.Blockchain)
	contractAddress := ctx.Value(ctxContractAddressKey).([]byte)

	value, err := bc.GetContractState(contractAddress, []byte(key))
	if err != nil {
		log.Printf("VM (db_get): Lỗi: %v", err)
		L.Push(lua.LNil)
		return 1
	}

	if value == nil {
		L.Push(lua.LNil)
		return 1
	}

	L.Push(lua.LString(string(value)))
	return 1
}

func luaGetSender(L *lua.LState) int {

	ctx := L.Context()
	senderAddress := ctx.Value(ctxSenderAddressKey).([]byte)

	L.Push(lua.LString(fmt.Sprintf("%x", senderAddress)))
	return 1
}

func (v *VM) RunContractDeploy(code []byte) error {

	return v.L.DoString(string(code))
}

func (v *VM) RunContractCall(code []byte, functionName string, args []lua.LValue) error {

	if err := v.L.DoString(string(code)); err != nil {
		return fmt.Errorf("lỗi khi load code: %v", err)
	}

	fn := v.L.GetGlobal(functionName)
	if fn.Type() == lua.LTNil {
		return fmt.Errorf("hàm '%s' không tồn tại trong contract", functionName)
	}

	err := v.L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, args...)

	if err != nil {
		return fmt.Errorf("lỗi khi thực thi hàm '%s': %v", functionName, err)
	}
	return nil
}
