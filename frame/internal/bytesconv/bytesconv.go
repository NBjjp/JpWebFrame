package bytesconv

import "unsafe"

func StringtoBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{
			s,
			len(s),
		},
	))
}
