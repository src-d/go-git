package cshared

import (
	"errors"
	"math"
	"sync"
)

type Handle uint64

var NotFoundError error = errors.New("object not found")
const InvalidHandle Handle = 0
const IH uint64 = uint64(InvalidHandle)
var counter Handle = InvalidHandle
var opMutex sync.Mutex
var registryHandle2Obj map[Handle]interface{}
var registryObj2Handle map[interface{}]Handle

func getNewHandle() Handle {
	counter++
	if counter == math.MaxUint64 {
		panic("Handle cache is exhausted")
	}
	return counter
}

func RegisterObject(obj interface{}) Handle {
	opMutex.Lock()
	defer opMutex.Unlock()
	handle, ok := registryObj2Handle[obj]
	if ok {
		return handle
	}
  handle = getNewHandle()
	registryHandle2Obj[handle] = obj
	registryObj2Handle[obj] = handle
	return handle
}

func UnregisterObject(handle Handle) error {
	if handle == InvalidHandle {
		return NotFoundError
	}
	opMutex.Lock()
	defer opMutex.Unlock()
	obj, ok := registryHandle2Obj[handle]
	if !ok {
		return errors.New("handle not found")
	}
	delete(registryHandle2Obj, handle)
	other_handle, ok := registryObj2Handle[obj]
	if !ok || other_handle != handle {
		panic("inconsistent internal object mapping state")
	}
	delete(registryObj2Handle, obj)
	return nil
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
	opMutex.Lock()
	defer opMutex.Unlock()
	a, b := registryObj2Handle[obj]
	return a, b
}

//export c_dispose
func c_dispose(handle uint64) {
	UnregisterObject(Handle(handle))
}

//export c_exists
func c_objects_size() int {
	return len(registryHandle2Obj)
}
