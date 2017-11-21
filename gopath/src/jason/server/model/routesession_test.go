package model

import (
	"regexp"
	"testing"
)

var (
	ptnRFC4122 = regexp.MustCompile("([\\w\\d]{8,8}-[\\w\\d]{4,4}-[\\w\\d]{4,4}-[\\w\\d]{4,4}-[\\w\\d]{12,12})")
)

func TestTokenIsRFC4122(t *testing.T) {
	if !ptnRFC4122.MatchString("9d3503e0-7236-4e47-a62f-8b01b5646c16") {
		t.Error("Test case incorrect")
	} else {
		t.Log("RFC4122 pattern check")
	}

	tk := newToken()
	// t.Log("new token", tk)
	if !ptnRFC4122.MatchString(tk) {
		t.Error("Token format incorrect")
	} else {
		t.Log("RFC4122 token format OK")
	}

	tk2 := newToken()
	if tk == tk2 {
		t.Error("Token not unique")
	}
}
