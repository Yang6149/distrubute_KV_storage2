package shardkv

import (
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"distrubute_KV_storage/shardmaster"
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

	raftclients  map[int]map[int]*labrpc.Client //[group+me][target(group+me)]Client
	svrclients   map[int]map[int]*labrpc.Client //[clientNum][target(group+me)]Client

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

func MakeMyConfig(masterN ,groupN, serverN int)*my_config{
	cfg := &my_config{}
	//最多9个集群，每个99个机器
	//0 为 master ,9 为clerk,1-8为普通集群
	cfg.net = labrpc.MakeAllNet(false,false,masterN,groupN,serverN)
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


	return cfg
}

func (cfg *my_config)StartMasterServer(i int){
	// 传入每个Raft的client，传入Server信息
	cfg.raftclients[+i] = labrpc.MakeGroupRaftClient(i, 0, cfg.n, cfg.net)
	cfg.raftclients[i] = labrpc.MakeGroupSerClient(i, 0, cfg.n, cfg.net)

}


