package frame

import (
	"strings"
	"unicode"
	"unsafe"
)

func SubStringLast(str, substr string) string {
	//返回子串str在字符串s中第一次出现的位置。
	//如果找不到则返回-1；如果str为空，则返回0
	index := strings.Index(str, substr)
	if index < 0 {
		return ""
	}
	return str[index+len(substr):]
}

//判断是否是ASCII里的字符
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func StringtiBytes(s string) []byte {
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
