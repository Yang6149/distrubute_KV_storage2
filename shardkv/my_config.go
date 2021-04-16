package shardkv

import (
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"distrubute_KV_storage/shardmaster"
	"fmt"
	"sync"
	"time"
)

type my_config struct {
	mu    sync.Mutex
	net   *labrpc.NetWork
	start time.Time // time at which make_config() was called

	nmasters      int // n 个机器足证masterserver
	masterservers []*shardmaster.ShardMaster
	mck           *shardmaster.Clerk

	ngroups int // 有多少个集群
	n       int // servers per k/v group  每个集群有多少server
	groups  []*my_group

	raftclients map[int]map[int]*labrpc.Client //[group+me][target(group+me)]Client
	svrclients  map[int]map[int]*labrpc.Client //[clientNum][target(group+me)]Client

	clerks       map[*Clerk][]string
	nextClientId int
	maxraftstate int
}

type my_group struct {
	gid       int
	servers   []*ShardKV
	saved     []*raft.Persister
	endnames  [][]string
	mendnames [][]string
}

func MakeMyConfig(masterN, groupN, serverN int) *my_config {
	cfg := &my_config{}
	//最多9个集群，每个99个机器
	//0 为 master ,9 为clerk,1-8为普通集群
	cfg.net = labrpc.MakeAllNet(false, false, masterN, groupN, serverN)
	cfg.start = time.Now()
	//初始化metadata
	cfg.nmasters = masterN
	cfg.ngroups = groupN
	cfg.n = serverN
	cfg.raftclients = make(map[int]map[int]*labrpc.Client, cfg.n)
	cfg.svrclients = make(map[int]map[int]*labrpc.Client, cfg.n)

	// 初始化网络结构 -- 创建Net时已经完成

	//masterservice
	cfg.nmasters = masterN
	cfg.masterservers = make([]*shardmaster.ShardMaster, cfg.nmasters)
	for i := 0; i < cfg.nmasters; i++ {
		cfg.StartMasterServer(i)
	}
	cfg.mck = cfg.masterClerk()

	//启动所有集群
	cfg.ngroups = groupN
	cfg.groups = make([]*my_group, cfg.ngroups)
	cfg.n = serverN
	for gi := 1; gi <= cfg.ngroups; gi++ {
		gg := &my_group{}
		cfg.groups[gi-1] = gg
		gg.gid = gi
		gg.servers = make([]*ShardKV, cfg.n)
		gg.saved = make([]*raft.Persister, cfg.n)
		gg.endnames = make([][]string, cfg.n)
		gg.mendnames = make([][]string, cfg.nmasters)
		for i := 0; i < cfg.n; i++ {
			cfg.StartServer(gi, i)
		}
	}
	fmt.Println("********************************",cfg.groups)

	return cfg
}

func (cfg *my_config) StartMasterServer(i int) {
	// 传入每个Raft的client，传入Server信息
	cfg.raftclients[i] = labrpc.MakeGroupRaftClient(0,i, 0, cfg.nmasters, cfg.net)
	//用于初始化Server信息
	cfg.svrclients[i] = labrpc.MakeGroupSerClient(0,i, 0, cfg.nmasters, cfg.net)

	p := raft.MakePersister()

	cfg.masterservers[i] = shardmaster.StartServer(cfg.raftclients, cfg.svrclients, i, 0, p)

}

func (cfg *my_config) StartServer(g, i int) {

	gg := cfg.groups[g-1]
	// 传入每个Raft的client，传入Server信息
	cfg.raftclients[g*100+i] = labrpc.MakeGroupRaftClient(g,i, g, cfg.n, cfg.net)

	cfg.svrclients[g*100+i] = labrpc.MakeGroupSerClientAll(g,i, cfg.ngroups, cfg.n, cfg.net)
	// 第一份参数为自己id，
	master := labrpc.MakeGroupSerClientAsList(g,i, 0, cfg.n, cfg.net)
	if gg.saved[i] != nil {
		gg.saved[i] = gg.saved[i].Copy()
	} else {
		gg.saved[i] = raft.MakePersister()
	}

	gg.servers[i] = StartServer(cfg.raftclients, cfg.svrclients,master,i,gg.saved[i],-1,g)


}


func (cfg *my_config) cleanup() {
	for gi := 0; gi < cfg.ngroups; gi++ {
		cfg.ShutdownGroup(gi)
	}
}

func (cfg *my_config) ShutdownGroup(g int){
	for i := 0; i < cfg.n; i++ {
		cfg.ShutdownServer(g, i)
	}
}
func (cfg *my_config) ShutdownServer(g,i int){
	cfg.groups[g].servers[i].Kill()
}

func (cfg *my_config) checklogs() {
	for gi := 0; gi < cfg.ngroups; gi++ {
		for i := 0; i < cfg.n; i++ {
			raft := cfg.groups[gi].saved[i].RaftStateSize()
			snap := len(cfg.groups[gi].saved[i].ReadSnapshot())
			if cfg.maxraftstate >= 0 && raft > 2*cfg.maxraftstate {
				fmt.Printf("checklogs 1\n")
			}
			if cfg.maxraftstate < 0 && snap > 0 {
				fmt.Printf("checklogs 2\n")
			}
		}
	}
}
func (cfg *my_config) StartGroup(gi int) {
	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(gi, i)
	}
}

func (cfg *my_config) masterClerk() *shardmaster.Clerk{
	return shardmaster.MakeClerk(labrpc.MakeGroupSerClientAsList(10,0,0,cfg.ngroups,cfg.net))
}
