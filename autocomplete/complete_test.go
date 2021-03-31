package autocomplete

import (
	"testing"
)

func TestGetFix(t *testing.T) {
	if getPrefix("abc") != "abb{" {
		t.Error("err")
	}
	if getSuffix("abc") != "abc{" {
		t.Error("err")
	}
	if getPrefix("aba") != "ab`{" {
		t.Error("err")
	}
	if getSuffix("aba") != "aba{" {
		t.Error("err")
	}
}

func TestAutoComplete_Add(t *testing.T) {
	ac := NewAutoComplete()
	ac.Add("abca")
	ac.Add("abcd")
	ac.Add("dbcd")
	rets := ac.autoComplete("abc")
	if len(rets) != 2 || rets[0] != "abca" || rets[1] != "abcd" {
		t.Error("err")
	}
}
