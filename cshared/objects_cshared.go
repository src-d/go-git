// +build ignore
package main

import (
	"C"
	"time"

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