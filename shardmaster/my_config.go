package shardmaster

import (
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"sync"
	"testing"
	"time"
)

type my_config struct {
	mu           sync.Mutex
	t            *testing.T
	n            int
	servers      []*ShardMaster
	raftclients  map[int]map[int]*labrpc.Client //[group+me][target(group+me)]Client
	svrclients   map[int]map[int]*labrpc.Client //[clientNum][target(group+me)]Client
	saved        []*raft.Persister
	endnames     [][]string // names of each server's sending ClientEnds
	clerks       map[*Clerk][]string
	nextClientId int
	start        time.Time // time at which make_config() was called
	net          *labrpc.NetWork
	g            int //集群为0
}

func make_myconfig(t *testing.T, n int, unreliable bool, partitions bool) *my_config {

	cfg := &my_config{}
	cfg.t = t
	cfg.n = n
	cfg.servers = make([]*ShardMaster, cfg.n)
	cfg.saved = make([]*raft.Persister, cfg.n)
	cfg.raftclients = make(map[int]map[int]*labrpc.Client, cfg.n)
	cfg.svrclients = make(map[int]map[int]*labrpc.Client, cfg.n)
	cfg.clerks = make(map[*Clerk][]string)
	cfg.nextClientId = cfg.n + 1000 // client ids start 1000 above the highest serverid
	cfg.start = time.Now()
	cfg.net = labrpc.MakeNet(false, false, n)

	// create a full set of KV servers.
	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(i, 0)
	}

	return cfg
}
func (cfg *my_config) StartServer(i, g int) {

	cfg.raftclients[g*100+i] = labrpc.MakeGroupRaftClient(i, g, cfg.n, cfg.net)
	cfg.raftclients[g*100+i] = labrpc.MakeGroupSerClient(i, g, cfg.n, cfg.net)
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = raft.MakePersister()
	}
	cfg.servers[i] = StartServer(cfg.raftclients, cfg.svrclients, i, g, cfg.saved[i])
}

func (cfg *my_config) cleanup() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for i := 0; i < len(cfg.servers); i++ {
		if cfg.servers[i] != nil {
			cfg.servers[i].Kill()
		}
	}
	cfg.checkTimeout()
}

func (cfg *my_config) checkTimeout() {
	// enforce a two minute real-time limit on each test
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}

func (cfg *my_config) makeClient() *Clerk {
	cks := make([]*labrpc.Client,0)
	for _,v:= range cfg.svrclients[cfg.g]{
		cks = append(cks,v)
	}
	ck := MakeClerk(cks)

	return ck
}
func (cfg *my_config) DisconnectServer(s int) {
	cfg.net.ServerState[s] = false
	for i := 0; i < cfg.n; i++ {
		cfg.net.Connect[s][i] = false
		cfg.net.Connect[i][s] = false
	}
}
func (cfg *my_config) ConnectServer(s int) {
	cfg.net.ServerState[s] = true
	for i := 0; i < cfg.n; i++ {
		cfg.net.Connect[s][i] = true
		cfg.net.Connect[i][s] = true
	}
}
