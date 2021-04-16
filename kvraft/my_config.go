package kvraft

import (
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"fmt"
	"math/rand"
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
	raftclients  map[int]map[int]*labrpc.Client  //[group+me][target(group+me)]Client
	svrclients   map[int]map[int]*labrpc.Client	 //[clientNum][target(group+me)]Client
	clerks       map[*Clerk][]string
	nextClientId int
	maxraftstate int
	net          *labrpc.NetWork
	start        time.Time // time at which make_config() was called
	// begin()/end() statistics
	t0    time.Time // time at which test_test.go called cfg.begin()
	rpcs0 int       // rpcTotal() at start of test
	ops   int32     // number of clerk get/put/append method calls
	mu    sync.Mutex
}

func make_myconfig(t *testing.T, n int, maxraftstate int, unreliable bool, partitions bool) *Myconfig {

	cfg := &Myconfig{}
	cfg.t = t
	cfg.n = n
	cfg.kvservers = make([]*KVServer, cfg.n)
	cfg.saved = make([]*raft.Persister, cfg.n)
	cfg.raftclients = make(map[int]map[int]*labrpc.Client, cfg.n)
	cfg.svrclients = make(map[int]map[int]*labrpc.Client, cfg.n)
	cfg.clerks = make(map[*Clerk][]string)
	cfg.nextClientId = cfg.n + 1000 // client ids start 1000 above the highest serverid
	cfg.maxraftstate = maxraftstate
	cfg.start = time.Now()
	cfg.net = labrpc.MakeNet(unreliable, partitions, n)

	// create a full set of KV servers.
	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(i,0)
	}

	//cfg.ConnectAll()
	//cfg.net.Reliable(!unreliable)

	return cfg
}

func (cfg *Myconfig) StartServer(i, g int) {

	cfg.raftclients[g*100+i] = labrpc.MakeGroupRaftClient(g,i, g, cfg.n, cfg.net)
	cfg.raftclients[g*100+i] = labrpc.MakeGroupSerClient(g,i, g, cfg.n, cfg.net)
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = raft.MakePersister()
	}
	cfg.kvservers[i] = StartKVServer(cfg.raftclients, cfg.svrclients, i,g, cfg.saved[i], cfg.maxraftstate)
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
	cks := make([]*labrpc.Client,0)
	for _,v:= range cfg.svrclients[0]{
		cks = append(cks,v)
	}
	ck := MakeClerk(cks)

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
		cfg.t.Fatal("test took longer than 120 seconds" + time.Since(cfg.start).String())
	}
}

func (cfg *Myconfig) checkLeader() int {
	for iters := 0; iters < 100; iters++ {
		ms := 450 + (rand.Int63() % 100)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		leaders := make(map[int][]int)
		for i := 0; i < cfg.n; i++ {
			if term, leader := cfg.kvservers[i].rf.GetState(); leader {
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

			return leaders[lastTermWithLeader][0]

		}
	}
	cfg.t.Fatalf("expected one leader, got none")
	return -1
}


