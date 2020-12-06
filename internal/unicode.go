package internal

import "unicode/utf8"

func rawToRune(x ...byte) (r rune) {
	r, _ = utf8.DecodeRune(x)
	return
}

func toSubscript(n int) (r string) {
	if n == 0 {
		return string(rawToRune(0xE2, 0x82, 0x80))
	}
	ms := false
	if n < 0 {
		ms = true
		n = -n
	}
	rs := make([]rune, 0, 20)
	for n != 0 {
		rs = append(rs, rawToRune(0xE2, 0x82, byte(0x80+n%10)))
		n /= 10
	}
	if ms {
		rs = append(rs, rawToRune(0xE2, 0x82, 0x8B))
	}

	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}

	return string(rs)
}
