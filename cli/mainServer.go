package cli

import (
	"distrubute_KV_storage/shardkv"
	"fmt"
)


type MainServer struct{
	cliMap map[int64]*shardkv.Clerk
	config *shardkv.Config
}
func MakeServer(config *shardkv.Config) *MainServer {
	return &MainServer{config: config,cliMap: map[int64]*shardkv.Clerk{}}
}

func (ms *MainServer)Get(args GetArgs,reply *GetReply)error{
	client := ms.GetClient(args.ClientId)
	reply.Value = client.Get(args.Key)
	fmt.Println(len(args.Key))
	return nil
}
func (ms *MainServer)Put(args PutArgs,reply *PutAppendReply)error{
	client := ms.GetClient(args.ClientId)
	client.Put(args.Key,args.Value)
	return nil
}
func (ms *MainServer)Append(args AppendArgs,reply *PutAppendReply)error{
	client := ms.GetClient(args.ClientId)
	client.Append(args.Key,args.Value)
	return nil
}

func (ms *MainServer)GetClient(id int64)*shardkv.Clerk{
	if _,ok:= ms.cliMap[id];!ok{
		ms.cliMap[id] = ms.config.MakeClient()
		ms.cliMap[id].SetMe(id)
	}
	return ms.cliMap[id]
}