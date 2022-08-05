package util

import "testing"

func TestNewUID(t *testing.T) {
	uidMap := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		uid := NewUID()
		if uidMap[uid] {
			t.Errorf("duplicated uid: %s", uid)
		}
		uidMap[uid] = true
	}
	if len(uidMap) != 10000 {
		t.Errorf("unexpected uid map size: %d", len(uidMap))
	}
}
