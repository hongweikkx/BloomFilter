package BloomFilter

import "testing"

func TestBloomFilter(t *testing.T) {
	bloom, _ := NewBloom("redis")
	e := bloom.Add("123456780")
	if e != nil {
		t.Error(e.Error())
		return
	}
	exist := bloom.IsExist("123456780")
	if !exist {
		t.Error("test1 error")
	}
	exist = bloom.IsExist("1234567801")
	if exist {
		t.Error("test2 error")

	}
}
