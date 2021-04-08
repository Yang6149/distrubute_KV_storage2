package kvraft

import (
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type Myconfig struct {
	t            *testing.T
	n            int
	kvservers    []*KVServer
	saved        []*raft.Persister
	raftclients  []*labrpc.Client
	svrclients   []*labrpc.Client
	clerks       map[*Clerk][]string
	nextClientId int
	maxraftstate int
	start        time.Time // time at which make_config() was called
	// begin()/end() statistics
	t0    time.Time // time at which test_test.go called cfg.begin()
	rpcs0 int       // rpcTotal() at start of test
	ops   int32     // number of clerk get/put/append method calls
	mu    sync.Mutex
}

func make_myconfig(t *testing.T, n int, maxraftstate int) *Myconfig {

	cfg := &Myconfig{}
	cfg.t = t
	cfg.n = n
	cfg.kvservers = make([]*KVServer, cfg.n)
	cfg.saved = make([]*raft.Persister, cfg.n)
	cfg.raftclients = make([]*labrpc.Client, cfg.n)
	cfg.svrclients = make([]*labrpc.Client, cfg.n)
	cfg.clerks = make(map[*Clerk][]string)
	cfg.nextClientId = cfg.n + 1000 // client ids start 1000 above the highest serverid
	cfg.maxraftstate = maxraftstate

	// create a full set of KV servers.
	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(i)
	}

	//cfg.ConnectAll()
	//cfg.net.Reliable(!unreliable)

	return cfg
}

func (cfg *Myconfig) StartServer(i int) {

	cfg.raftclients = make([]*labrpc.Client, cfg.n)
	cfg.svrclients = make([]*labrpc.Client, cfg.n)
	for j := 0; j < cfg.n; j++ {
		cfg.raftclients[j] = labrpc.MakeMyClient("KV_Raft", i)
		cfg.svrclients[j] = labrpc.MakeMySerClient("KV_SVR", i)
	}
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = raft.MakePersister()
	}
	cfg.kvservers[i] = StartKVServer(cfg.raftclients, cfg.svrclients, i, cfg.saved[i], cfg.maxraftstate)
}

func (cfg *Myconfig) Leader() (bool, int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	for i := 0; i < cfg.n; i++ {
		_, is_leader := cfg.kvservers[i].rf.GetState()
		if is_leader {
			return true, i
		}
	}
	return false, 0
}

func (cfg *Myconfig) op() {
	atomic.AddInt32(&cfg.ops, 1)
}

func (cfg *Myconfig) makeClient(n []int) *Clerk {
	ck := MakeClerk(cfg.svrclients)

	return ck
}
func (cfg *Myconfig) All() []int {
	all := make([]int, cfg.n)
	for i := 0; i < cfg.n; i++ {
		all[i] = i
	}
	return all
}

func (cfg *Myconfig) deleteClient(ck *Clerk) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	v := cfg.clerks[ck]
	for i := 0; i < len(v); i++ {
		os.Remove(v[i])
	}
	delete(cfg.clerks, ck)
}
func (cfg *Myconfig) end() {
	cfg.checkTimeout()
	if cfg.t.Failed() == false {

		fmt.Printf("  ... Passed --")
	}
}
func (cfg *Myconfig) checkTimeout() {
	// enforce a two minute real-time limit on each test
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}
