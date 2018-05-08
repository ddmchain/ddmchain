
// +build evmjit

package vm

/*

void* evmjit_create();
int   evmjit_run(void* _jit, void* _data, void* _env);
void  evmjit_destroy(void* _jit);

#cgo LDFLAGS: -levmjit
*/
import "C"

//export env_sha3
func env_sha3(dataPtr *byte, length uint64, resultPtr unsafe.Pointer) {
	data := llvm2bytesRef(dataPtr, length)
	hash := crypto.Keccak256(data)
	result := (*i256)(resultPtr)
	*result = hash2llvm(hash)
}

//export env_sstore
func env_sstore(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, valuePtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := llvm2hash(bswap((*i256)(valuePtr)))
	value = trim(value)
	if len(value) == 0 {
		prevValue := vm.env.State().GetState(vm.me.Address(), index)
		if len(prevValue) != 0 {
			vm.Env().State().Refund(vm.callerAddr, GasSStoreRefund)
		}
	}

	vm.env.State().SetState(vm.me.Address(), index, value)
}

//export env_sload
func env_sload(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, resultPtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := vm.env.State().GetState(vm.me.Address(), index)
	result := (*i256)(resultPtr)
	*result = hash2llvm(value)
	bswap(result)
}

//export env_balance
func env_balance(_vm unsafe.Pointer, _addr unsafe.Pointer, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)
	addr := llvm2hash((*i256)(_addr))
	balance := vm.Env().State().GetBalance(addr)
	result := (*i256)(_result)
	*result = big2llvm(balance)
}

//export env_blockhash
func env_blockhash(_vm unsafe.Pointer, _number unsafe.Pointer, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)
	number := llvm2big((*i256)(_number))
	result := (*i256)(_result)

	currNumber := vm.Env().BlockNumber()
	limit := big.NewInt(0).Sub(currNumber, big.NewInt(256))
	if number.Cmp(limit) >= 0 && number.Cmp(currNumber) < 0 {
		hash := vm.Env().GetHash(uint64(number.Int64()))
		*result = hash2llvm(hash)
	} else {
		*result = i256{}
	}
}

//export env_call
func env_call(_vm unsafe.Pointer, _gas *int64, _receiveAddr unsafe.Pointer, _value unsafe.Pointer, inDataPtr unsafe.Pointer, inDataLen uint64, outDataPtr *byte, outDataLen uint64, _codeAddr unsafe.Pointer) bool {
	vm := (*JitVm)(_vm)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in env_call (depth %d, out %p %d): %s\n", vm.Env().Depth(), outDataPtr, outDataLen, r)
		}
	}()

	balance := vm.Env().State().GetBalance(vm.me.Address())
	value := llvm2big((*i256)(_value))

	if balance.Cmp(value) >= 0 {
		receiveAddr := llvm2hash((*i256)(_receiveAddr))
		inData := C.GoBytes(inDataPtr, C.int(inDataLen))
		outData := llvm2bytesRef(outDataPtr, outDataLen)
		codeAddr := llvm2hash((*i256)(_codeAddr))
		gas := big.NewInt(*_gas)
		var out []byte
		var err error
		if bytes.Equal(codeAddr, receiveAddr) {
			out, err = vm.env.Call(vm.me, codeAddr, inData, gas, vm.price, value)
		} else {
			out, err = vm.env.CallCode(vm.me, codeAddr, inData, gas, vm.price, value)
		}
		*_gas = gas.Int64()
		if err == nil {
			copy(outData, out)
			return true
		}
	}

	return false
}

//export env_create
func env_create(_vm unsafe.Pointer, _gas *int64, _value unsafe.Pointer, initDataPtr unsafe.Pointer, initDataLen uint64, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	value := llvm2big((*i256)(_value))
	initData := C.GoBytes(initDataPtr, C.int(initDataLen)) 
	result := (*i256)(_result)
	*result = i256{}

	gas := big.NewInt(*_gas)
	ret, suberr, ref := vm.env.Create(vm.me, nil, initData, gas, vm.price, value)
	if suberr == nil {
		dataGas := big.NewInt(int64(len(ret))) 
		dataGas.Mul(dataGas, params.CreateDataGas)
		gas.Sub(gas, dataGas)
		*result = hash2llvm(ref.Address())
	}
	*_gas = gas.Int64()
}

//export env_log
func env_log(_vm unsafe.Pointer, dataPtr unsafe.Pointer, dataLen uint64, _topic1 unsafe.Pointer, _topic2 unsafe.Pointer, _topic3 unsafe.Pointer, _topic4 unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	data := C.GoBytes(dataPtr, C.int(dataLen))

	topics := make([][]byte, 0, 4)
	if _topic1 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic1)))
	}
	if _topic2 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic2)))
	}
	if _topic3 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic3)))
	}
	if _topic4 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic4)))
	}

	vm.Env().AddLog(state.NewLog(vm.me.Address(), topics, data, vm.env.BlockNumber().Uint64()))
}

//export env_extcode
func env_extcode(_vm unsafe.Pointer, _addr unsafe.Pointer, o_size *uint64) *byte {
	vm := (*JitVm)(_vm)
	addr := llvm2hash((*i256)(_addr))
	code := vm.Env().State().GetCode(addr)
	*o_size = uint64(len(code))
	return getDataPtr(code)
}*/
