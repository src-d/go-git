// +build ignore
package main

import (
	"C"
	"fmt"
	"math"
	"sync"
	"reflect"
)

type Handle uint64

const (
	ErrorCodeSuccess = iota
	ErrorCodeNotFound = iota
	ErrorCodeInternal = iota
)

const MessageNotFound string = "object not found"
const InvalidHandle Handle = 0
const IH uint64 = uint64(InvalidHandle)
var counter Handle = InvalidHandle
var opMutex sync.Mutex
var registryHandle2Obj map[Handle]interface{} = map[Handle]interface{}{}
var registryObj2Handle map[uintptr]Handle = map[uintptr]Handle{}

func getNewHandle() Handle {
	counter++
	if counter == math.MaxUint64 {
		panic("Handle cache is exhausted")
	}
	return counter
}

func RegisterObject(obj interface{}) Handle {
	data_ptr := reflect.ValueOf(&obj).Elem().InterfaceData()[1]
	opMutex.Lock()
	defer opMutex.Unlock()
	handle, ok := registryObj2Handle[data_ptr]
	if ok {
		return handle
	}
  handle = getNewHandle()
	registryHandle2Obj[handle] = obj
	registryObj2Handle[data_ptr] = handle
	return handle
}

func UnregisterObject(handle Handle) int {
	if handle == InvalidHandle {
		return ErrorCodeNotFound
	}
	opMutex.Lock()
	defer opMutex.Unlock()
	obj, ok := registryHandle2Obj[handle]
	if !ok {
		return ErrorCodeNotFound
	}
	delete(registryHandle2Obj, handle)
	data_ptr := reflect.ValueOf(&obj).Elem().InterfaceData()[1]
	other_handle, ok := registryObj2Handle[data_ptr]
	if !ok || other_handle != handle {
		panic("inconsistent internal object mapping state")
	}
	delete(registryObj2Handle, data_ptr)
	return ErrorCodeSuccess
}

func GetObject(handle Handle) (interface{}, bool) {
	if handle == InvalidHandle {
		return nil, false
	}
	opMutex.Lock()
	defer opMutex.Unlock()
	a, b := registryHandle2Obj[handle]
	return a, b
}

func GetHandle(obj interface{}) (Handle, bool) {
	data_ptr := reflect.ValueOf(&obj).Elem().InterfaceData()[1]
	opMutex.Lock()
	defer opMutex.Unlock()
	a, b := registryObj2Handle[data_ptr]
	return a, b
}

func CopyString(str string) string {
	buf := make([]byte, len(str))
	copy(buf, []byte(str))
	return string(buf)
}

func SafeIsNil(v reflect.Value) bool {
  defer func() { recover() }()
  return v.IsNil()
}

//export c_dispose
func c_dispose(handle uint64) {
	UnregisterObject(Handle(handle))
}

//export c_objects_size
func c_objects_size() int {
	return len(registryHandle2Obj)
}

//export c_dump_objects
func c_dump_objects() {
	fmt.Println("handles:")
	for h, obj := range(registryHandle2Obj) {
		fmt.Printf("0x%x\t0x%x  %v\n", h,
			reflect.ValueOf(&obj).Elem().InterfaceData()[1], obj)
	}
	fmt.Println()
	fmt.Println("pointers:")
	for ptr, h := range(registryObj2Handle) {
		fmt.Printf("0x%x\t0x%x\n", ptr, h)
	}
}

// dummy main() is needed by the linker
func main() {}
