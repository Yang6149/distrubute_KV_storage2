package shardkv

//
// client code to talk to a sharded key/value service.
//
// the client first talks to the shardmaster to find out
// the assignment of shards (keys) to groups, and then
// talks to the group that holds the key's shard.
//

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/shardmaster"
)

//
// which shard is a key in?
// please use this function,
// and please do not change it.
//
func key2shard(key string) int {
	shard := 0
	if len(key) > 0 {
		shard = int(key[0])
	}
	shard %= shardmaster.NShards
	return shard
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

type Clerk struct {
	sm      *shardmaster.Clerk
	config  shardmaster.Config
	clients *labrpc.Clients
	id      int
	me      int64
	serial  int
	leader  int

	// You will have to modify this struct.
}

//
// the tester calls MakeClerk.
//
// masters[] is needed to call shardmaster.MakeClerk().
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs.
//
func MakeClerk(clients *labrpc.Clients) *Clerk {
	ck := new(Clerk)
	ck.sm = shardmaster.MakeClerk(clients)

	ck.me = nrand()
	ck.serial = 0
	ck.leader = 0
	// You'll have to add code here.
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
// You will have to modify this function.
//
func (ck *Clerk) Get(key string) string {
	args := GetArgs{}
	args.Key = key
	args.ClientId = ck.me
	ck.serial++
	args.SerialId = ck.serial
	for {
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			// try each server for the shard.
			for si := 0; si < len(servers); si++ {
				srv := ck.clients.GroupsServ[gid][si]
				var reply GetReply
				ok := srv.Call("Get", args, &reply)
				if ok && (reply.Err == OK || reply.Err == ErrNoKey) {
					//DPrintf("%d get res %s = %s", gid, args.Key, reply.Value)
					return reply.Value
				}
				if ok && (reply.Err == ErrWrongGroup) {
					break
				}
				// ... not ok, or ErrWrongLeader
			}
		}
		time.Sleep(100 * time.Millisecond)
		// ask master for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}

}

//
// shared by Put and Append.
// You will have to modify this function.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	args := PutAppendArgs{}
	args.Key = key
	args.Value = value
	args.Op = op
	args.ClientId = ck.me
	ck.serial++
	args.SerialId = ck.serial
	for {
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			for si := 0; si < len(servers); si++ {
				srv := ck.clients.GroupsServ[gid][si]
				var reply PutAppendReply
				ok := srv.Call("PutAppend", args, &reply)
				if ok && reply.Err == OK {
					return
				}
				if ok && reply.Err == ErrWrongGroup {
					break
				}
				// ... not ok, or ErrWrongLeader
			}
		}
		time.Sleep(100 * time.Millisecond)
		// ask master for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}

func StrToInt(str string) int {
	//"server-" + strconv.Itoa(gid) + "-" + strconv.Itoa(i)
	strs := strings.Split(str, "-")
	fmt.Println("str := ", strs)
	g := strs[1]
	i := strs[2]
	Gval, err := strconv.ParseInt(g, 10, 64)
	if err != nil {
		fmt.Printf("string convert to int error ,%v\n", err)
	}
	Ival, err := strconv.ParseInt(i, 10, 64)
	if err != nil {

		fmt.Printf("string convert to int error ,%v\n", err)
	}
	return int(Gval*100 + Ival)
}
