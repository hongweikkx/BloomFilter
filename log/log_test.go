package log

import (
	"testing"
)

func TestLog(t *testing.T) {
	log := NewLog()
	for i := 0; i < 150; i++ {
		err := log.CommonLog("hwgao", "hello", "INFO")
		if err != nil {
			t.Error(err)
		}
		log.RecentLog("hwgao", "hello", "INFO")
	}
}
