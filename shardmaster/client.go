package shardmaster

//
// Shardmaster clerk.
//

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"distrubute_KV_storage/labrpc"
)

type Clerk struct {
	servers *labrpc.Clients
	// Your data here.
	id       int64
	serialId int
	leader   int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers *labrpc.Clients) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.id = nrand()
	ck.serialId = 0

	// Your code here.
	return ck
}

func (ck *Clerk) Query(num int) Config {
	//如果多个client 同时发起命令还想保持线性化结果，就不能注释掉下一行
	ck.serialId++
	args := QueryArgs{Num: num}
	// Your code here.
	args.Num = num
	args.ClientId = ck.id
	args.SerialId = ck.serialId
	for {
		// try each known server.
		srv := ck.servers.GroupsServ[0][ck.leader]
		var reply QueryReply
		ok := srv.Call("Query", args, &reply)
		// fmt.Println(reply)
		if ok && reply.WrongLeader == false && reply.Err == OK {
			return reply.Config
		}
		time.Sleep(100 * time.Millisecond)
		ck.leader = (ck.leader + 1) % len(ck.servers.GroupsServ[0])
	}
}

func (ck *Clerk) Join(servers map[int][]string) {
	ck.serialId++
	args := JoinArgs{}
	// Your code here.
	args.Servers = servers
	args.ClientId = ck.id
	args.SerialId = ck.serialId
	for {
		// try each known server.
		srv := ck.servers.GroupsServ[0][ck.leader]
		var reply JoinReply
		fmt.Println("try2")
		ok := srv.Call("Join", args, &reply)
		fmt.Println("try3")
		fmt.Println(ok, reply)
		fmt.Println(srv)
		if ok && reply.WrongLeader == false && reply.Err == OK {
			return
		}
		fmt.Println("try4")
		time.Sleep(100 * time.Millisecond)
		ck.leader = (ck.leader + 1) % len(ck.servers.GroupsServ[0])
	}
}

func (ck *Clerk) Leave(gids []int) {
	ck.serialId++
	args := LeaveArgs{}
	// Your code here.
	args.GIDs = gids
	args.ClientId = ck.id
	args.SerialId = ck.serialId
	for {
		// try each known server.
		srv := ck.servers.GroupsServ[0][ck.leader]
		var reply LeaveReply
		ok := srv.Call("Leave", args, &reply)
		if ok && reply.WrongLeader == false && reply.Err == OK {
			return
		}
		time.Sleep(100 * time.Millisecond)
		ck.leader = (ck.leader + 1) % len(ck.servers.GroupsServ[0])
	}
}

func (ck *Clerk) Move(shard int, gid int) {
	ck.serialId++
	args := MoveArgs{}
	// Your code here.
	args.Shard = shard
	args.GID = gid
	args.ClientId = ck.id
	args.SerialId = ck.serialId
	for {
		// try each known server.
		srv := ck.servers.GroupsServ[0][ck.leader]
		var reply MoveReply
		ok := srv.Call("Move", args, &reply)
		if ok && reply.WrongLeader == false && reply.Err == OK {
			return
		}
		time.Sleep(100 * time.Millisecond)
		ck.leader = (ck.leader + 1) % len(ck.servers.GroupsServ[0])
	}
}

func (ck *Clerk) Joinm(gis []int) {
	ck.joinm(gis)
}
func (ck *Clerk) EasyJoin(gi int) {
	ck.joinm([]int{gi})
}

func (ck *Clerk) joinm(gis []int) {
	m := make(map[int][]string, len(gis))
	for _, g := range gis {
		servernames := make([]string, ck.servers.Num)
		for i := 0; i < ck.servers.Num; i++ {
			servernames[i] = ck.servers.GroupsServ[g][i].GetIP()
		}
		m[g] = servernames
	}
	ck.Join(m)
}

func (ck *Clerk) EasyLeave(gi int) {
	ck.Leave([]int{gi})
}
func (ck *Clerk) EasyQuery() Config {
	return ck.Query(-1)
}
