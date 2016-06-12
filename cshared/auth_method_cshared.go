// +build ignore
package main

import (
	"C"
	"reflect"
	"strings"

	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v3/clients/http"
	. "gopkg.in/src-d/go-git.v3/clients/ssh"
)

//export c_NewBasicAuth
func c_NewBasicAuth(username, password string) uint64 {
	auth := http.NewBasicAuth(CopyString(username), CopyString(password))
	return uint64(RegisterObject(auth))
}

//export c_ParseRawPrivateKey
func c_ParseRawPrivateKey(pemBytes []byte) (uint64, int, *C.char) {
	pkey, err := ssh.ParseRawPrivateKey(pemBytes)
	if err != nil {
		return IH, ErrorCodeInternal, C.CString(err.Error())
	}
	return uint64(RegisterObject(pkey)), ErrorCodeSuccess, C.CString("")
}

//export c_ParsePrivateKey
func c_ParsePrivateKey(pemBytes []byte) (uint64, int, *C.char) {
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return IH, ErrorCodeInternal, C.CString(err.Error())
	}
	return uint64(RegisterObject(signer)), ErrorCodeSuccess, C.CString("")
}

//export c_NewPublicKey
func c_NewPublicKey(key uint64) (uint64, int, *C.char) {
	obj, ok := GetObject(Handle(key))
	if !ok {
		return IH, ErrorCodeNotFound, C.CString(MessageNotFound)
	}
	key_obj := obj.(ssh.PublicKey)
	pkey, err := ssh.NewPublicKey(key_obj)
	if err != nil {
		return IH, ErrorCodeInternal, C.CString(err.Error())
	}
	return uint64(RegisterObject(pkey)), ErrorCodeSuccess, C.CString("")
}

//export c_NewSignerFromKey
func c_NewSignerFromKey(key uint64) (uint64, int, *C.char) {
	obj, ok := GetObject(Handle(key))
	if !ok {
		return IH, ErrorCodeNotFound, C.CString(MessageNotFound)
	}
	signer, err := ssh.NewSignerFromKey(obj)
	if err != nil {
		return IH, ErrorCodeInternal, C.CString(err.Error())
	}
	return uint64(RegisterObject(signer)), ErrorCodeSuccess, C.CString("")
}

//export c_MarshalAuthorizedKey
func c_MarshalAuthorizedKey(key uint64) []byte {
	obj, ok := GetObject(Handle(key))
	if !ok {
		return []byte{}
	}
	obj_key := obj.(ssh.PublicKey)
	return ssh.MarshalAuthorizedKey(obj_key)
}

//export c_ParsePublicKey
func c_ParsePublicKey(in []byte) (uint64, int, *C.char) {
	pkey, err := ssh.ParsePublicKey(in)
	if err != nil {
		return IH, ErrorCodeInternal, C.CString(err.Error())
	}
	return uint64(RegisterObject(pkey)), ErrorCodeSuccess, C.CString("")
}

//export c_ParseAuthorizedKey
func c_ParseAuthorizedKey(in []byte) (uint64, *C.char, *C.char, []byte, int, *C.char) {
	pkey, comment, options, rest, err := ssh.ParseAuthorizedKey(in)
	if err != nil {
		return IH, C.CString(""), C.CString(""), []byte{}, ErrorCodeInternal,
		       C.CString(err.Error())
	}
	pkey_handle := RegisterObject(pkey)
	mopt := strings.Join(options, "\xff")
	return uint64(pkey_handle), C.CString(comment), C.CString(mopt), rest,
	       ErrorCodeSuccess, C.CString("")
}

//export c_ssh_Password_New
func c_ssh_Password_New(user, pass string) uint64 {
	obj := Password{User: CopyString(user), Pass: CopyString(pass)}
	return uint64(RegisterObject(obj))
}

//export c_ssh_Password_get_User
func c_ssh_Password_get_User(p uint64) *C.char {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return C.CString("")
	}
	return C.CString(obj.(Password).User)
}

//export c_ssh_Password_set_User
func c_ssh_Password_set_User(p uint64, v string) {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return
	}
	reflect.ValueOf(obj).Elem().FieldByName("User").SetString(CopyString(v))
}

//export c_ssh_Password_get_Pass
func c_ssh_Password_get_Pass(p uint64) *C.char {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return C.CString("")
	}
	return C.CString(obj.(Password).Pass)
}

//export c_ssh_Password_set_Pass
func c_ssh_Password_set_Pass(p uint64, v string) {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return
	}
	reflect.ValueOf(obj).Elem().FieldByName("Pass").SetString(CopyString(v))
}

//c_ssh_PublicKeys_New
func c_ssh_PublicKeys_New(user string, signer uint64) uint64 {
	obj, ok := GetObject(Handle(signer))
	if !ok {
		return IH
	}
	pk := PublicKeys{User: CopyString(user), Signer: obj.(ssh.Signer)}
	return uint64(RegisterObject(pk))
}

//export c_ssh_PublicKeys_get_User
func c_ssh_PublicKeys_get_User(p uint64) *C.char {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return C.CString("")
	}
	return C.CString(obj.(PublicKeys).User)
}

//export c_ssh_PublicKeys_set_User
func c_ssh_PublicKeys_set_User(p uint64, v string) {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return
	}
	reflect.ValueOf(obj).Elem().FieldByName("User").SetString(CopyString(v))
}

//export c_ssh_PublicKeys_get_Signer
func c_ssh_PublicKeys_get_Signer(p uint64) uint64 {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return IH
	}
	handle, ok := GetHandle(obj.(PublicKeys).Signer)
	if !ok {
		return IH
	}
	return uint64(handle)
}

//export c_ssh_PublicKeys_set_Signer
func c_ssh_PublicKeys_set_Signer(p uint64, v uint64) {
	obj, ok := GetObject(Handle(p))
	if !ok {
		return
	}
	signer, ok := GetObject(Handle(v))
	if !ok {
		return
	}
	reflect.ValueOf(obj).Elem().FieldByName("Signer").Set(reflect.ValueOf(signer))
}