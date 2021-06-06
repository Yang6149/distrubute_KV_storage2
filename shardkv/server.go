package shardkv

// import "../shardmaster"
import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"distrubute_KV_storage/labgob"
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"distrubute_KV_storage/shardmaster"
	"distrubute_KV_storage/tool"
)

const Debug = 0

var begin int64

func (kv *ShardKV) DPrintf(format string, a ...interface{}) (n int, err error) {

	if Debug > 0 {
		_, isLeader := kv.rf.GetState()
		if !isLeader {
			return
		}
		now := time.Now().UnixNano()
		res := (now - begin) / int64(time.Millisecond)
		a = append(a, res)
		log.Printf(format, a...)
	}
	return
}
func EPrintf(format string, a ...interface{}) (n int, err error) {

	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}
func init() {
	begin = time.Now().UnixNano()
	fmt.Println("123")
	f, err := os.OpenFile("logfile.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("?????????????????")
		log.Fatalf("file open error : %v", err)
	}
	//defer f.Close()
	log.SetOutput(f)
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Type     string
	Key      string
	Value    string
	ClientId int64
	SerialId int
	Err      Err
}

type Shard struct {
	Id      int
	Data    map[string]string
	Dup     map[int64]int
	Version int
}
type GC struct {
	Shard   int
	Version int
}

type MigrateArgs struct {
	Shard     Shard
	ConfigNum int
	Gid       int
}
type MigrateReply struct {
	Err Err
}

type ShardKV struct {
	mu           sync.Mutex
	me           int
	rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	gid          int
	maxraftstate int // snapshot if log grows this big
	sm           *shardmaster.Clerk
	config       shardmaster.Config
	conf         tool.Conf
	clients      *labrpc.Clients // true RPC client

	// Your definitions here.
	dead          int32 // set by Kill()
	apps          map[int]chan Op
	appsforConfig map[int]chan shardmaster.Config
	appsforShard  map[int]chan Shard
	appsforGC     map[int]chan GC
	shards        map[int]Shard

	impleConfig       int
	GCch              chan GC
	lastIncludedIndex int
	listener          *net.Listener
}

func (kv *ShardKV) Get(args GetArgs, reply *GetReply) error {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if isLeader {
		op := Op{"Get", args.Key, "", args.ClientId, args.SerialId, ""}
		reply.Value, reply.Err = kv.start(op)
	}
	return nil
}

func (kv *ShardKV) PutAppend(args PutAppendArgs, reply *PutAppendReply) error {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if isLeader {
		op := Op{args.Op, args.Key, args.Value, args.ClientId, args.SerialId, ""}
		_, Err := kv.start(op)
		reply.Err = Err
		return nil
	} else {
		reply.Err = ErrWrongLeader
		return nil
	}
}

func (kv *ShardKV) start(op Op) (string, Err) {
	kv.mu.Lock()
	//判断是否改变了配置而拒绝请求
	shard := key2shard(op.Key)
	gid := kv.config.Shards[shard]

	if s, ok := kv.shards[shard]; gid != kv.gid || !ok || s.Version < kv.config.Num {
		//gid 对不上、不存在该 shard、存在但是不可用状态
		defer kv.mu.Unlock()
		fmt.Println(kv.gid, kv.me, gid != kv.gid, !ok, s.Version < kv.config.Num, s.Version, kv.config.Num)
		fmt.Println("%d %d %d %d", kv.gid, kv.me, gid, ok)
		return "", ErrWrongGroup
	}
	//检查put重复或是否直接返回get
	shardId := key2shard(op.Key)
	if res, ok := kv.shards[shardId].Dup[op.ClientId]; ok && res >= op.SerialId {
		res := ""
		if op.Type == "Get" {
			res = kv.shards[shardId].Data[op.Key]
		}
		defer kv.mu.Unlock()
		return res, OK
	}
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		defer kv.mu.Unlock()
		return "", ErrWrongLeader
	}
	ch := make(chan Op, 1)
	kv.apps[index] = ch
	kv.mu.Unlock()
	defer func() {
		kv.mu.Lock()
		delete(kv.apps, index)
		kv.mu.Unlock()
	}()
	select {
	case oop := <-ch:
		//返回成功
		if op.ClientId == oop.ClientId && op.SerialId == oop.SerialId {
			res := oop.Value
			return res, oop.Err
		} else {
			return "", ErrWrongLeader
		}

	case <-time.After(500 * time.Millisecond):
		return "", ErrTimeOut
	}
}

//
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	EPrintf("%d %d :杀死一个 server", kv.gid, kv.me)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardmaster.
//
// pass masters[] to shardmaster.MakeClerk() so you can send
// RPCs to the shardmaster.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use masters[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartServer(clients *labrpc.Clients, conf tool.Conf, me int, persister *raft.Persister, maxraftstate int, g int) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(ShardKV)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.gid = g
	kv.conf = conf
	kv.clients = clients

	// Your initialization code here.

	// Use something like this to talk to the shardmaster:
	// kv.mck = shardmaster.MakeClerk(kv.masters)
	//构建一个自己的用户
	kv.sm = shardmaster.MakeClerk(clients)
	go func() {
		fmt.Println("*****************", conf.Ip, conf.Port)
		rpc.RegisterName("Serv", kv)
		listener, err := net.Listen("tcp", conf.Ip+":"+strconv.Itoa(conf.Port))
		kv.listener = &listener
		fmt.Printf("shardkv serverName := %s \t listener := %s \n", "Serv", conf.Ip+":"+strconv.Itoa(conf.Port))

		if err != nil {
			log.Fatal("ListenTCP error:", err)
		}
		for {
			conn, err := listener.Accept()
			if kv.killed() {
				return
			}
			if err != nil {
				log.Fatal("Accept error:", err)
			}

			go rpc.ServeConn(conn)
		}
	}()

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(clients, conf, g, persister, kv.applyCh)

	kv.apps = make(map[int]chan Op)
	kv.appsforConfig = make(map[int]chan shardmaster.Config)
	kv.appsforGC = make(map[int]chan GC)
	kv.appsforShard = make(map[int]chan Shard)
	kv.GCch = make(chan GC, 100)
	kv.shards = make(map[int]Shard)
	//kv.initShard()
	kv.maxraftstate = maxraftstate
	kv.LoadSnapshot(kv.rf.GetSnapshots())
	EPrintf("%d %d :init finished", kv.gid, kv.me)
	EPrintf("%d %d :config- %v", kv.gid, kv.me, kv.config)
	//labgob init
	labgob.Register(MigrateArgs{})
	labgob.Register(Shard{})
	labgob.Register(shardmaster.Config{})
	labgob.Register(GC{})
	// You may need initialization code here.
	go kv.apply()
	go kv.fetchLatestConfig()
	go kv.detectConfig()
	go kv.GCDeamon()
	return kv
}

func (kv *ShardKV) initShard() {
	for i := 0; i < shardmaster.NShards; i++ {
		res := Shard{}
		res.Version = 0
		res.Data = make(map[string]string)
		res.Dup = make(map[int64]int)
		kv.shards[i] = res
	}
}

func (kv *ShardKV) apply() {
	for {
		// if (time.Now().UnixNano()-begin)/int64(time.Millisecond) > 20000 {
		// 	kv.Kill()
		// }
		if kv.killed() {
			return
		}
		msg := <-kv.applyCh
		// now := time.Now().UnixNano()
		// res := (now - begin) / int64(time.Millisecond)
		//fmt.Println(kv.gid, kv.me, " 接收到msg=", msg, res)
		//msg.CommandValid is true,otherwise the command is snapshot
		kv.DPrintf("%d %d get a command %d", kv.gid, kv.me, msg)
		if msg.CommandValid {
			kv.mu.Lock()
			switch command := msg.Command.(type) {
			case Op:
				op := command
				shardId := key2shard(op.Key)
				if shard, ok := kv.shards[shardId]; ok {
					if shard.Version != kv.config.Num {
						//该 shard != config.Version,直接返回让 start 超时
						kv.DPrintf("FUCKFUCKFUCKFUCKFUCKFUCKFUCKFUCKFUCKFUCKFUCK")
						op.Err = ErrWrongGroup
					} else {
						if res, ok := kv.shards[shardId].Dup[op.ClientId]; !ok || (ok && op.SerialId > res) {
							switch op.Type {
							case "Put":
								kv.shards[shardId].Data[op.Key] = op.Value
								kv.DPrintf("%d %d put %s -> %s ", kv.gid, kv.me, op.Key, op.Value)
							case "Append":
								kv.DPrintf("%d %d append %s -> (%s) + (%s) ", kv.gid, kv.me, op.Key, kv.shards[shardId].Data[op.Key], op.Value)
								kv.shards[shardId].Data[op.Key] = kv.shards[shardId].Data[op.Key] + op.Value
								op.Value = kv.shards[shardId].Data[op.Key]
							}
							if kv.shards[shardId].Dup == nil {
								kv.DPrintf("出大大大问题")
							}
							kv.shards[shardId].Dup[op.ClientId] = op.SerialId
						} else {
						}
						if op.Type == "Get" {
							op.Value = kv.shards[shardId].Data[op.Key]
						}
						op.Err = OK
					}
				} else {
					//shard 已经被传送走并且被删除了，直接返回，让他超时。
					kv.DPrintf("shard 已经被传送走并且被删除了	")
				}

				ch, ok := kv.apps[msg.CommandIndex]
				if ok {
					ch <- op
				}

				// 判断是否达到max
				kv.checkMaxState(msg.CommandIndex)
			case shardmaster.Config:
				config := command
				if config.Num > kv.config.Num {
					kv.impleConfig = config.Num
					//为那些没有改变shard 的 version 进行同步
					kv.DPrintf("%d %d config=%d,kv.config=%d ", kv.gid, kv.me, config, kv.config)
					for i := 0; i < shardmaster.NShards; i++ {
						if config.Shards[i] == kv.gid && kv.config.Shards[i] == kv.gid {
							shard := kv.shards[i]
							shard.Version = config.Num
							kv.shards[i] = shard
							kv.DPrintf("%d %d 没改变 shard ，单纯升级 shard version shard[%d]->%d ", kv.gid, kv.me, i, config.Num)
						}
					}
					if kv.config.Num == 0 {
						temp := make([]int, 0)
						for i, _ := range config.Shards {
							if config.Shards[i] == kv.gid {
								shard := Shard{}
								shard.Id = i
								shard.Data = make(map[string]string)
								shard.Dup = make(map[int64]int)
								shard.Version = config.Num
								kv.shards[i] = shard
								temp = append(temp, i)
							}
						}
						kv.DPrintf("%d %d 初始化 %d ", kv.gid, kv.me, temp)
					}

					kv.config = config
					// now := time.Now().UnixNano()
					// res := (now - begin) / int64(time.Millisecond)
					//fmt.Println(kv.gid, kv.me, "更新", config.Num, res)
					kv.DPrintf("%d %d 更新 config%d ", kv.gid, kv.me, config)
				} else {
					kv.DPrintf("%d %d ignore 小于当前 config 的 config，config= %d,my=%d", kv.gid, kv.me, config.Num, kv.config.Num)
				}
				ch, ok := kv.appsforConfig[msg.CommandIndex]
				if ok {
					ch <- config
				}
				kv.checkMaxState(msg.CommandIndex)
			case Shard:
				shard := command
				if _, ok := kv.shards[shard.Id]; !ok || shard.Version > kv.shards[shard.Id].Version {
					if !ok {
						kv.DPrintf("%d %d 原来无 shard %d", kv.gid, kv.me, shard.Id)
					} else {
						kv.DPrintf("%d %d 原来的 shard%d ", kv.gid, kv.me, kv.shards[shard.Id])
					}
					kv.shards[shard.Id] = deepCopyShard(shard)
					if shard.Data == nil {
						kv.DPrintf("dadadadadadadadadada")
					}
					kv.DPrintf("%d %d 更新 shard%d ", kv.gid, kv.me, shard)
				}
				ch, ok := kv.appsforShard[msg.CommandIndex]
				if ok {
					ch <- shard
				}
				kv.checkMaxState(msg.CommandIndex)
			case GC:
				gc := command
				shard, ok := kv.shards[gc.Shard]
				if ok {
					//版本正好正确
					if shard.Version == gc.Version {
						kv.DPrintf("%d %d 更新gc shard %d ,version = %d ", kv.gid, kv.me, gc.Shard, gc.Version)
						delete(kv.shards, gc.Shard)
					} else {
						kv.DPrintf("%d %d 你想gc%d 版本号为 %d ，我的版本号为 %d", kv.gid, kv.me, gc.Shard, gc.Version, shard.Version)
					}
				}
				ch, ok := kv.appsforGC[msg.CommandIndex]
				if ok {
					ch <- gc
				}
				kv.checkMaxState(msg.CommandIndex)
			}
			kv.mu.Unlock()
			//判断是否重复指令

		} else {
			data := msg.Command.([]byte)
			index := msg.CommandIndex
			if index <= kv.lastIncludedIndex {
				kv.rf.SnapshotF <- -1
				continue
			}
			kv.LoadSnapshot(data)
			kv.mu.Lock()
			kv.SnapshotPersister(index)
			kv.mu.Unlock()
			kv.rf.SnapshotF <- 1
		}
	}
}

func (kv *ShardKV) checkMaxState(commitIndex int) {
	if kv.maxraftstate == -1 {
		return
	}
	if kv.maxraftstate*9/10 > kv.rf.GetStateSize() {
		return
	}
	kv.SnapshotPersister(commitIndex)
}
func (kv *ShardKV) encodeSnapshot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(kv.shards)
	e.Encode(kv.lastIncludedIndex)
	e.Encode(kv.impleConfig)
	e.Encode(kv.config)
	snapshot := w.Bytes()
	return snapshot
}

func (kv *ShardKV) SnapshotPersister(index int) {
	kv.rf.SaveSnapshot(kv.encodeSnapshot())
	kv.rf.Discard(index)
	kv.lastIncludedIndex = index
}

func (kv *ShardKV) LoadSnapshot(snapshot []byte) {
	if snapshot == nil || len(snapshot) < 1 { // bootstrap without any state?
		return
	}
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	var shards map[int]Shard
	var lastIncludedIndex int
	var impleConfig int
	var config shardmaster.Config
	if d.Decode(&shards) != nil ||
		d.Decode(&lastIncludedIndex) != nil ||
		d.Decode(&impleConfig) != nil ||
		d.Decode(&config) != nil {
	} else {
		kv.mu.Lock()
		defer kv.mu.Unlock()
		kv.shards = shards
		kv.lastIncludedIndex = lastIncludedIndex
		kv.impleConfig = impleConfig
		kv.config = config
	}
}

func (kv *ShardKV) fetchLatestConfig() {
	for {
		if kv.killed() {
			return
		}
		select {
		case <-time.After(50 * time.Millisecond):
			_, isLeader := kv.rf.GetState()
			if !isLeader {
				continue
			}
			kv.mu.Lock()

			isContinue := false
			for i := 0; i < shardmaster.NShards; i++ {
				shard, ok := kv.shards[i]
				if kv.config.Shards[i] == kv.gid && (!ok || shard.Version < kv.config.Num) {
					kv.DPrintf("%d %d 口口口口口口口口口口口口口口口口口口口口口口口", kv.gid, kv.me)
					if !ok {
						kv.DPrintf("%d %d 还没这个 shard %d ，所以不要传新的过来", kv.gid, kv.me, i)
					} else if shard.Version < kv.config.Num {
						kv.DPrintf("%d %d 还没更新 shard %d ，所以不要传新的过来shard.Version = %d ,kv.config.Num = %d", kv.gid, kv.me, i, shard.Version, kv.config.Num)
					}
					isContinue = true
					break
				}
				if ok && kv.config.Shards[i] != kv.gid && shard.Version < kv.config.Num {
					//我还没删除掉不属于我的东西
					isContinue = true
					kv.DPrintf("%d %d 还没删除 shard %d ，所以不要传新的过来", kv.gid, kv.me, i)
					break
				}
			}
			if isContinue {
				kv.mu.Unlock()
				continue
			}
			kv.DPrintf("%d %d 这是要poi呀config=%d", kv.gid, kv.me, kv.config)
			for k, _ := range kv.shards {
				kv.DPrintf("%d %d ,shards[%d].Version=%d", kv.gid, kv.me, k, kv.shards[k].Version)
			}
			config := kv.sm.Query(kv.config.Num + 1)
			kv.DPrintf("%d %d imple = %d newconfig is %d", kv.gid, kv.me, kv.config.Num, config)
			if !kv.check_same_config(config, kv.config) {
				_, _, isleader := kv.rf.Start(config)
				if !isleader {
					kv.DPrintf("%d %d Start - config not leader ", kv.gid, kv.me)
					kv.mu.Unlock()
					continue
				}
				kv.DPrintf("%d %d Start is %d", kv.gid, kv.me, config)
				// ch := make(chan shardmaster.Config, 1)
				// kv.appsforConfig[index] = ch
				// defer func() {
				// 	kv.mu.Lock()
				// 	delete(kv.appsforConfig, index)
				// 	kv.mu.Unlock()
				// }()
				// select {
				// case <-ch:
				// 	//其实我觉得这里没差，因为如果修改成功的话，就不会进入判断重复的这里了，所以等会把这个删掉
				// 	//成功
				// case <-time.After(1000 * time.Millisecond):
				// 	//没成功
				// }
			} else {
				kv.DPrintf("%d %d 没变哟", kv.gid, kv.me)
			}
			kv.mu.Unlock()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (kv *ShardKV) detectConfig() {
	for {
		if kv.killed() {
			return
		}
		select {
		case <-time.After(50 * time.Millisecond):
			_, isLeader := kv.rf.GetState()
			if isLeader {
				kv.mu.Lock()
				//检查当前 shard 是否发送了
				for i := 0; i < shardmaster.NShards; i++ {
					shard, ok := kv.shards[i]
					if kv.config.Shards[i] != kv.gid && ok && shard.Version < kv.config.Num {
						//拥有 shard 但config上指示我应该没有，所以我要把它发送给有用的人。再删除掉
						go kv.sendMigration(kv.config.Shards[i], i, shard.Version)
					}
				}
				kv.mu.Unlock()
			}

		}
		//time.Sleep(100 * time.Millisecond)
	}
}

func (kv *ShardKV) check_same_config(c1 shardmaster.Config, c2 shardmaster.Config) bool {
	if c1.Num != c2.Num {
		return false
	}
	return true
}

func (kv *ShardKV) sendMigrateArgs(cli *labrpc.TrueClient, args MigrateArgs, reply *MigrateReply) bool {
	ok := cli.Call("MigrateReply", args, reply)
	return ok
}

func (kv *ShardKV) MigrateReply(args MigrateArgs, reply *MigrateReply) error {
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		EPrintf("%d %d 返回reply wrong leader", kv.gid, kv.me)
		reply.Err = ErrWrongLeader
		return nil
	}
	EPrintf("%d %d 返回", kv.gid, kv.me)
	kv.mu.Lock()
	EPrintf("%d %d 锁内", kv.gid, kv.me)
	if args.ConfigNum < kv.config.Num {
		reply.Err = OK
		defer kv.mu.Unlock()
		return nil
	}
	if shard, ok := kv.shards[args.Shard.Id]; ok && shard.Version == args.Shard.Version {
		//已经拥有该shard ，判断重复直接返回确认。
		reply.Err = OK
		defer kv.mu.Unlock()
		return nil
	}
	if args.Shard.Data == nil {
		kv.DPrintf("poipoipoipoipoipoippoipoipoi")
	}
	copyShard := deepCopyShard(args.Shard)
	index, _, _ := kv.rf.Start(copyShard)
	ch := make(chan Shard, 1)
	kv.appsforShard[index] = ch
	kv.mu.Unlock()
	defer func() {
		kv.mu.Lock()
		delete(kv.appsforShard, index)
		kv.mu.Unlock()
	}()
	select {
	case shard := <-ch:
		kv.DPrintf("%d %d 接受到 来自 %d 的shard %d", kv.gid, kv.me, args.Gid, shard)
		reply.Err = OK
		return nil
	case <-time.After(200 * time.Millisecond):
		reply.Err = ErrTimeOut
		kv.DPrintf("%d %d rep-migrate %d 的shard %d,is timeout", kv.gid, kv.me, args.Gid, args.Shard)
		return nil
	}
}

// func (kv *ShardKV) updateDataAndDup(data map[string]string, dup map[int64]int) {

// }

func (kv *ShardKV) sendMigration(gid int, shard int, shardGCVersion int) {
	if _, isLeader := kv.rf.GetState(); !isLeader {
		return
	}

	//kv.DPrintf("%d %d 向 %d 发送shard[%d] %d", kv.gid, kv.me, gid, shard, kv.shards[shard])
	if servers, ok := kv.config.Groups[gid]; ok {
		for si := 0; si < len(servers); si++ {
			go kv.sendMigrationForOne(gid, shard, shardGCVersion, si, servers)
			// ... not ok, or ErrWrongLeader
		}
	}
}

func (kv *ShardKV) sendMigrationForOne(gid int, shard int, shardGCVersion int, si int, servers []string) {
	kv.mu.Lock()
	//kv.DPrintf("%d %d 获得锁", kv.gid, kv.me)
	srv := kv.clients.GroupsServ[gid][si]
	var args MigrateArgs
	args.Shard = kv.copyOfShard(shard)
	//判断是否取到了空值，代表已经发送过了，并且gc掉，所以这里判重
	if args.Shard.Id != shard {
		//fmt.Println("逮到了")
		kv.mu.Unlock()
		return
	}
	//判断是否在连个进程间，shard发生变化  shard[i]=100->101->100,导致想要发送第一个100，结果发送了第二个100的版本，所以这里判重
	if args.Shard.Version != shardGCVersion {
		kv.mu.Unlock()
		return
	}
	args.Shard.Version = kv.config.Num
	args.ConfigNum = kv.config.Num
	args.Gid = kv.gid
	var reply MigrateReply
	kv.mu.Unlock()
	done := make(chan bool, 1)
	kv.DPrintf("%d %d 发送srv= %d migrate 前-> %d %d", kv.gid, kv.me, srv, gid, shard)
	go func() {
		ok := kv.sendMigrateArgs(srv, args, &reply)
		kv.DPrintf("%d %d 发送srv= %d migrate 完-> %d %d，ok=%d", kv.gid, kv.me, srv, gid, shard, ok)
		done <- ok
	}()
	select {
	case ok := <-done:
		kv.DPrintf("%d %d 接收到done 完-> %d %d", kv.gid, kv.me, gid, shard)
		if ok && reply.Err == OK {
			//成功发送Migration ，开始删除
			//delete shard[i]
			gc := GC{}
			gc.Shard = shard
			gc.Version = shardGCVersion
			kv.DPrintf("%d %d 发送信号量删除->%d shard%d version %d nowlen=%d", kv.gid, kv.me, gid, gc.Shard, gc.Version, len(kv.GCch))

			kv.GCch <- gc
			break
		} else {
			if reply.Err == ErrWrongLeader {
				kv.DPrintf("%d %d 发送 migration->%d %d：%d，srv=%d", kv.gid, kv.me, gid, shard, reply.Err, srv)
				return
			}
			if ok {
				kv.DPrintf("%d %d 发送 migration->%d %d：%d", kv.gid, kv.me, gid, shard, reply.Err)
			} else {
				kv.DPrintf("%d %d 发送 migration->%d %d：rpc 失败", kv.gid, kv.me, gid, shard)
			}
			time.Sleep(100 * time.Millisecond)
			//go kv.sendMigration(gid, shard)
		}
	case <-time.After(500 * time.Millisecond):
		kv.DPrintf("%d %d 发送 migration->%d %d：超时", kv.gid, kv.me, gid, shard)
	}
}

func (kv *ShardKV) copyOfShard(i int) Shard {
	res := Shard{}
	res.Data = make(map[string]string)
	res.Dup = make(map[int64]int)
	for k, v := range kv.shards[i].Data {
		res.Data[k] = v
	}
	for k, v := range kv.shards[i].Dup {
		res.Dup[k] = v
	}
	if shard, ok := kv.shards[i]; ok {
		res.Id = shard.Id
	}
	res.Version = kv.shards[i].Version
	return res
}

func (kv *ShardKV) GC(shard int, version int) Err {
	gc := GC{}
	gc.Shard = shard
	gc.Version = version
	kv.DPrintf("%d %d 发送gcc试探 %d", kv.gid, kv.me, gc)
	b := time.Now().UnixNano()
	kv.mu.Lock()
	a := time.Now().UnixNano()
	res := (a - b) / int64(time.Millisecond)
	kv.DPrintf("%d %d 获得锁2,争锁时常 %d", kv.gid, kv.me, res)
	if s, ok := kv.shards[shard]; !ok {
		//已经不存在了，所以不发送gc 指令到 raft
		kv.mu.Unlock()
		return OK
	} else {
		//存在这个 shard，再判断 version 是否正确,不正确就代表被覆盖了
		if s.Version != version {
			kv.mu.Unlock()
			return OK
		}
	}
	kv.mu.Unlock()
	kv.DPrintf("%d %d 发送gcc %d", kv.gid, kv.me, gc)
	index, _, isLeader := kv.rf.Start(gc)
	if !isLeader {
		return ErrWrongLeader
	}
	kv.mu.Lock()

	ch := make(chan GC, 1)
	kv.appsforGC[index] = ch
	kv.mu.Unlock()
	defer func() {
		kv.mu.Lock()
		delete(kv.appsforGC, index)
		kv.mu.Unlock()
	}()
	select {
	case <-ch:
		//gc 成功
		kv.DPrintf("%d %d 成功返回gc()=OK shard=%d,version=%d", kv.gid, kv.me, shard, version)
		return OK
	case <-time.After(1000 * time.Millisecond):
		kv.DPrintf("%d %d 失败返回gc()=ErrTimeOut shard=%d,version=%d", kv.gid, kv.me, shard, version)
		return ErrTimeOut
	}

}
func (kv *ShardKV) GCDeamon() {
	for {
		if kv.killed() {
			return
		}
		//信号量
		gc := <-kv.GCch
		EPrintf("%d %d 想要GC %d", kv.gid, kv.me, gc)
		go func() {
			Err := kv.GC(gc.Shard, gc.Version)
			if Err != OK {
				EPrintf("%d %d 想要GC %d,失败了，再试一次,原因 %v", kv.gid, kv.me, gc, Err)
				//kv.GCch <- gc
			}
		}()

	}
}

func deepCopyMigrate(args MigrateArgs) MigrateArgs {
	res := MigrateArgs{}
	res.ConfigNum = args.ConfigNum
	res.Gid = args.Gid
	res.Shard = deepCopyShard(args.Shard)
	return res
}

func deepCopyShard(shard Shard) Shard {
	res := Shard{}
	res.Id = shard.Id
	res.Version = shard.Version
	res.Data = make(map[string]string)
	res.Dup = make(map[int64]int)
	for k, v := range shard.Data {
		res.Data[k] = v
	}
	for k, v := range shard.Dup {
		res.Dup[k] = v
	}
	return res
}
