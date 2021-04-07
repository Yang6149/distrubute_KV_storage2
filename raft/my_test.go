package raft

import (
	"testing"
	"time"
)

func Test3Raft(t *testing.T) {
	cfg := make_my_config(t, 3)
	time.Sleep(2000)
	cfg.checkOneLeader()
	cfg.one(99, 3, true)
}
