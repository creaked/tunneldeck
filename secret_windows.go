//go:build windows

package main

import (
	"encoding/base64"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modCrypt32         = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtect   = modCrypt32.NewProc("CryptProtectData")
	procCryptUnprotect = modCrypt32.NewProc("CryptUnprotectData")
	modKernel32        = syscall.NewLazyDLL("kernel32.dll")
	procLocalFree      = modKernel32.NewProc("LocalFree")
)

// dataBlob mirrors the Win32 DATA_BLOB / CRYPTOAPI_BLOB structure.
type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newDataBlob(data []byte) *dataBlob {
	if len(data) == 0 {
		return &dataBlob{}
	}
	return &dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}

func (b *dataBlob) bytes() []byte {
	if b.pbData == nil || b.cbData == 0 {
		return nil
	}
	return unsafe.Slice(b.pbData, b.cbData)
}

// encryptSecret encrypts plaintext using Windows DPAPI (user-scoped).
// The returned string is a base64-encoded DPAPI blob safe to store in JSON.
func encryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	input := newDataBlob([]byte(plaintext))
	var output dataBlob
	r, _, err := procCryptProtect.Call(
		uintptr(unsafe.Pointer(input)),
		0, // szDataDescr
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		0, // dwFlags
		uintptr(unsafe.Pointer(&output)),
	)
	if r == 0 {
		return "", fmt.Errorf("CryptProtectData: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(output.pbData)))
	return base64.StdEncoding.EncodeToString(output.bytes()), nil
}

// decryptSecret decrypts a base64-encoded DPAPI blob produced by encryptSecret.
func decryptSecret(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	input := newDataBlob(raw)
	var output dataBlob
	r, _, err := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(input)),
		0, // ppszDataDescr
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		0, // dwFlags
		uintptr(unsafe.Pointer(&output)),
	)
	if r == 0 {
		return "", fmt.Errorf("CryptUnprotectData: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(output.pbData)))
	return string(output.bytes()), nil
}
