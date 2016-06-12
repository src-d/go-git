// +build ignore
package main

import (
	"C"
	"io/ioutil"
	"time"

	"gopkg.in/src-d/go-git.v3/core"
	. "gopkg.in/src-d/go-git.v3"
)

//export c_Signature_Name
func c_Signature_Name(s uint64) *C.char {
	obj, ok := GetObject(Handle(s))
	if !ok {
		return C.CString("")
	}
	sign := obj.(*Signature)
	return C.CString(sign.Name)
}

//export c_Signature_Email
func c_Signature_Email(s uint64) *C.char {
	obj, ok := GetObject(Handle(s))
	if !ok {
		return C.CString("")
	}
	sign := obj.(*Signature)
	return C.CString(sign.Email)
}

//export c_Signature_When
func c_Signature_When(s uint64) *C.char {
	obj, ok := GetObject(Handle(s))
	if !ok {
		return C.CString("")
	}
	sign := obj.(*Signature)
	return C.CString(sign.When.Format(time.RFC3339))
}

//export c_Signature_Decode
func c_Signature_Decode(b []byte) uint64 {
	sign := Signature{}
	sign.Decode(b)
	return uint64(RegisterObject(&sign))
}

//export c_Blob_get_Hash
func c_Blob_get_Hash(b uint64) *C.char {
	obj, ok := GetObject(Handle(b))
	if !ok {
		return nil
	}
	blob := obj.(*Blob)
	return C.CString(string(blob.Hash[:]))
}

//export c_Blob_Size
func c_Blob_Size(b uint64) int64 {
	obj, ok := GetObject(Handle(b))
	if !ok {
		return -1
	}
	blob := obj.(*Blob)
	return blob.Size
}

//export c_Blob_Decode
func c_Blob_Decode(o uint64) uint64 {
	obj, ok := GetObject(Handle(o))
	if !ok {
		return IH
	}
	cobj := obj.(*core.Object)
	blob := Blob{}
	blob.Decode(*cobj)
	return uint64(RegisterObject(&blob))
}

//export c_Blob_Read
func c_Blob_Read(b uint64) (int, *C.char) {
	obj, ok := GetObject(Handle(b))
	if !ok {
		return ErrorCodeNotFound, C.CString(MessageNotFound)
	}
	blob := obj.(*Blob)
	reader, err := blob.Reader()
	if err != nil {
		return ErrorCodeInternal, C.CString(err.Error())
	}
  data, err := ioutil.ReadAll(reader)
	reader.Close()
	if err != nil {
		return ErrorCodeInternal, C.CString(err.Error())
	}
	return len(data), C.CString(string(data))
}