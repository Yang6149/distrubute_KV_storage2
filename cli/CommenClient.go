package cli

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/rpc"
	"strconv"
)

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut     = "ErrTimeOut"
	ErrStaleData   = "ErrStaleData"
)

type Err string

// Put or Append
type PutArgs struct {
	// You'll have to add definitions here.
	Key      string
	Value    string
	ClientId int64
}
type AppendArgs struct {
	// You'll have to add definitions here.
	Key      string
	Value    string
	ClientId int64
}

type PutAppendReply struct {
}

type GetArgs struct {
	Key      string
	ClientId int64
	// You'll have to add definitions here.
}

type GetReply struct {
	Value string
}

type JLArgs struct{
	Commend string
	G int
}
type JLReply struct{
	Options string
}

type RPCClient struct{
	serverName string
	Me int64
	client *rpc.Client
}

type ConArgs struct{
	G int
	I int
}
type ConReply struct{
	Options string
}

func MakeClient(serverName string,port int) (*RPCClient,error) {
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("dialing:", err)
	}
	return &RPCClient{Me: nrand(),serverName: serverName,client:client},err
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func (cli *RPCClient) Get(Key string){
	args := GetArgs{}
	args.ClientId = cli.Me
	args.Key = Key
	reply := &GetReply{}
	err := cli.Call("Get",args,reply)
	if err!=nil{
		return
	}
	fmt.Printf("val:%s\n",reply.Value)

}
func (cli *RPCClient) Put(Key , Val string){
	args := PutArgs{}
	args.ClientId = cli.Me
	args.Key = Key
	args.Value = Val
	reply := &PutAppendReply{}
	err := cli.Call("Put",args,reply)
	if err!=nil{
		return
	}
	fmt.Println("OK")
}
func (cli *RPCClient) Append(Key , Val string){
	args := AppendArgs{}
	args.ClientId = cli.Me
	args.Key = Key
	args.Value = Val
	reply := &PutAppendReply{}
	err := cli.Call("Append",args,reply)
	if err!=nil{
		return
	}
	fmt.Println("OK")
}

func (cli *RPCClient) Join(g int){
	args := JLArgs{}
	args.Commend = "Join"
	args.G = g
	reply := &JLReply{}
	err := cli.Call("Join",args,reply)
	if err!=nil{
		return
	}
	fmt.Println("OK")
}
func (cli *RPCClient) Leave(g int){
	args := JLArgs{}
	args.Commend = "Leave"
	args.G = g
	reply := &JLReply{}
	err := cli.Call("Leave",args,reply)
	if err!=nil{
		return
	}
	fmt.Println("OK")
}

func (cli *RPCClient) GetInfo(){
	args := JLArgs{}
	reply := &JLReply{}
	err := cli.Call("GetInfo",args,reply)
	if err!=nil{
		fmt.Println("GetInfo Error")
		return
	}
	fmt.Print(reply.Options)
}

func (cli *RPCClient) Connect(g,i int){
	args := ConArgs{
		G:g,
		I:i,
	}
	reply := &ConReply{}
	err := cli.Call("Connect",args,reply)
	if err!=nil{
		fmt.Println("Connect Error")
		return
	}
	fmt.Print(reply.Options)
}
func (cli *RPCClient) DisConnect(g,i int){
	args := ConArgs{
		G:g,
		I:i,
	}
	reply := &ConReply{}
	err := cli.Call("DisConnect",args,reply)
	if err!=nil{
		fmt.Println("DisConnect Error")
		return
	}
	fmt.Print(reply.Options)
}
func (cli *RPCClient) Call(method string, args interface{}, reply interface{})error{
	err := cli.client.Call(cli.serverName+"."+method, args, reply)
	if err != nil {
		fmt.Printf("发生错误调用,err := %v",err.Error())
		return err
	}
	return nil
}
