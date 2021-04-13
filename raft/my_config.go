package raft

import (
	"distrubute_KV_storage/labrpc"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type MyConfig struct {
	clients   []*labrpc.Client
	n         int
	rafts     []*Raft
	saved     []*Persister
	t         *testing.T
	mu        sync.Mutex
	logs      []map[int]interface{}
	maxIndex  int
	maxIndex0 int
	applyErr  []string // from apply channel readers
}

func make_my_config(t *testing.T, n int) *MyConfig {
	cfg := &MyConfig{}
	cfg.n = n
	cfg.rafts = make([]*Raft, cfg.n)
	cfg.clients = make([]*labrpc.Client, cfg.n)
	cfg.applyErr = make([]string, cfg.n)
	cfg.saved = make([]*Persister, cfg.n)
	cfg.logs = make([]map[int]interface{}, cfg.n)

	for i := 0; i < n; i++ {
		cfg.clients[i] = labrpc.MakeMyClient("Raft", 0, i, &labrpc.NetWork{false, false})
	}
	for i := 0; i < n; i++ {
		cfg.logs[i] = map[int]interface{}{}
		cfg.start(i)
	}

	return cfg
}
func (cfg *MyConfig) start(i int) {
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = MakePersister()
	}

	applyCh := make(chan ApplyMsg)
	go func() {
		for m := range applyCh {
			err_msg := ""
			if m.CommandValid == false {
				// ignore other types of ApplyMsg
			} else {
				v := m.Command
				cfg.mu.Lock()
				for j := 0; j < len(cfg.logs); j++ {
					if old, oldok := cfg.logs[j][m.CommandIndex]; oldok && old != v {
						// some server has already committed a different value for this entry!
						err_msg = fmt.Sprintf("commit index=%v server=%v %v != server=%v %v",
							m.CommandIndex, i, m.Command, j, old)
					}
				}
				_, prevok := cfg.logs[i][m.CommandIndex-1]
				cfg.logs[i][m.CommandIndex] = v
				if m.CommandIndex > cfg.maxIndex {
					cfg.maxIndex = m.CommandIndex
				}
				cfg.mu.Unlock()

				if m.CommandIndex > 1 && prevok == false {
					err_msg = fmt.Sprintf("server %v apply out of order %v", i, m.CommandIndex)
				}
			}

			if err_msg != "" {
				log.Fatalf("apply error: %v\n", err_msg)
				cfg.applyErr[i] = err_msg
				// keep reading after error so that Raft doesn't block
				// holding locks...
			}
		}
	}()
	rf := Make2(cfg.clients, i, cfg.saved[i], applyCh)
	cfg.rafts[i] = rf
}

func (cfg *MyConfig) checkOneLeader() int {
	for iters := 0; iters < 100; iters++ {
		ms := 450 + (rand.Int63() % 100)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		leaders := make(map[int][]int)
		for i := 0; i < cfg.n; i++ {
			if term, leader := cfg.rafts[i].GetState(); leader {
				leaders[term] = append(leaders[term], i)
			}
		}

		lastTermWithLeader := -1
		for term, leaders := range leaders {
			if len(leaders) > 1 {
				cfg.t.Fatalf("term %d has %d (>1) leaders", term, len(leaders))
			}
			if term > lastTermWithLeader {
				lastTermWithLeader = term
			}
		}

		if len(leaders) != 0 {
			for i := 0; i < len(cfg.rafts); i++ {
				fmt.Printf("num := %d , term := %d , state := %d ", cfg.rafts[i].me, cfg.rafts[i].currentTerm, cfg.rafts[i].state)

			}
			return leaders[lastTermWithLeader][0]

		}
	}
	cfg.t.Fatalf("expected one leader, got none")
	return -1
}

func (cfg *MyConfig) one(cmd interface{}, expectedServers int, retry bool) int {
	t0 := time.Now()
	starts := 0
	for time.Since(t0).Seconds() < 20 {
		// try all the servers, maybe one is the leader.
		index := -1
		for si := 0; si < cfg.n; si++ {
			starts = (starts + 1) % cfg.n
			var rf *Raft
			cfg.mu.Lock()
			rf = cfg.rafts[starts]
			cfg.mu.Unlock()
			if rf != nil {
				index1, _, ok := rf.Start(cmd)
				if ok {
					index = index1
					break
				}
			}
		}

		if index != -1 {
			// somebody claimed to be the leader and to have
			// submitted our command; wait a while for agreement.
			t1 := time.Now()
			for time.Since(t1).Seconds() < 2 {
				nd, cmd1 := cfg.nCommitted(index)
				if nd > 0 && nd >= expectedServers {
					// committed
					if cmd1 == cmd {
						// and it was the command we submitted.
						return index
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			if retry == false {
				cfg.t.Fatalf("one(%v) failed to reach agreement", cmd)
			}
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
	cfg.t.Fatalf("one(%v) failed to reach agreement", cmd)
	return -1
}

// how many servers think a log entry is committed?
func (cfg *MyConfig) nCommitted(index int) (int, interface{}) {
	count := 0
	var cmd interface{} = nil
	for i := 0; i < len(cfg.rafts); i++ {
		if cfg.applyErr[i] != "" {
			cfg.t.Fatal(cfg.applyErr[i])
		}

		cfg.mu.Lock()
		cmd1, ok := cfg.logs[i][index]
		cfg.mu.Unlock()

		if ok {
			if count > 0 && cmd != cmd1 {
				cfg.t.Fatalf("committed values do not match: index %v, %v, %v\n",
					index, cmd, cmd1)
			}
			count += 1
			cmd = cmd1
		}
	}
	return count, cmd
}
