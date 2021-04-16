package labrpc

import (
	"fmt"
	"math/rand"
	"net/rpc"
	"strconv"
	"time"
)

const (
	RaftName = "Raft"
	SerName  = "Ser"
)

type HelloService struct{}

func (p *HelloService) Hello(request string, reply *string) error {
	*reply = "hello:" + request
	return nil
}

type HelloService2 struct{}

func (p *HelloService2) Hello(request string, reply *string) error {
	*reply = "hello2:" + request
	return nil
}

const HelloServiceName1 = "path/to/pkg.HelloService1"
const HelloServiceName2 = "path/to/pkg.HelloService1"

type HelloServiceInterface = interface {
	Hello(request string, reply *string) error
}

func RegisterHelloService(svc HelloServiceInterface, serName string) error {
	return rpc.RegisterName(serName, svc)
}

//***************************************

type Client struct {
	Me          int    //代表自己单独Client
	ClusterName string //集群名字
	Port        string //端口
	Num         int    //第几号机器
	Group       int
	client      *rpc.Client //client
	net         *NetWork
}
type NetWork struct {
	Unreliable  bool
	Partitions  bool
	Connect     map[int]map[int]bool
	ServerState map[int]bool
}

func MakeMyClient(m int, name string, Targetgroup int, TargetI int, net *NetWork) *Client {
	client := &Client{}
	client.Me = m
	client.Port = strconv.Itoa(30000 + Targetgroup*100 + TargetI)
	client.Group = Targetgroup
	client.Num = TargetI
	client.ClusterName = name + strconv.Itoa(Targetgroup*100+TargetI)
	client.net = net
	return client
}
func MakeMySerClient(m int, name string, group int, i int, net *NetWork) *Client {
	client := &Client{}
	client.Me = m
	client.Port = strconv.Itoa(20000 + group*100 + i)
	client.Group = group
	client.Num = i
	client.ClusterName = name + strconv.Itoa(group*100+i)
	client.net = net
	return client
}

func (c *Client) Call(svcMeth string, args interface{}, reply interface{}) bool {
	for c.client == nil {
		client, err := rpc.Dial("tcp", "localhost:"+c.Port)
		if err != nil {
			//log.Println("dialing:", err)
			time.Sleep(time.Millisecond * 100)
		}
		c.client = client
	}
	if c.net.Unreliable == true {
		// short delay
		ms := (rand.Int() % 27)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}

	if c.net.Unreliable == true && (rand.Int()%1000) < 100 {
		// drop the request, return as if timeout
		return false
	}
	err := c.client.Call(c.ClusterName+"."+svcMeth, args, reply)
	if err != nil {
		fmt.Printf("错误%v ,me is %d,target is %d\n",err.Error(),c.Me,c.Num+c.Group*100)
	}
	return true
}
func MakeNet(Unreliable, Partitions bool, n int) *NetWork {
	net := &NetWork{}
	net.Unreliable = Unreliable
	net.Partitions = Partitions
	net.Connect = make(map[int]map[int]bool)
	for i := 0; i < n; i++ {
		net.Connect[i] = make(map[int]bool)
		net.ServerState[i] = true
		for j := 0; j < n; j++ {
			net.Connect[i][j] = true
		}
	}
	return net
}
func MakeAllNet(Unreliable, Partitions bool, nmaster, ngroup, n int) *NetWork {
	net := &NetWork{}
	net.Unreliable = Unreliable
	net.Partitions = Partitions
	net.Connect = make(map[int]map[int]bool)
	// 初始化 master集群间互相通信
	for i := 0; i < nmaster; i++ {
		net.Connect[i] = make(map[int]bool)
		for j := 0; j < nmaster; j++ {
			net.Connect[i][j] = true
		}
	}
	// 初始化所有group间通信
	for g := 1; g <= ngroup; g++ {
		for i := 0; i < n; i++ {
			net.Connect[g*100+i] = make(map[int]bool)
			for j := 0; j < n; j++ {
				net.Connect[g*100+i][g*100+j] = true
			}
		}
	}

	return net
}

//me+g 为自己，n为有几台机器
func MakeGroupRaftClient(muGroup, me, g int, n int, net *NetWork) map[int]*Client {
	//自己只能调用自己
	fmt.Println("[MakeGroupRaftClient]",n)
	m := make(map[int]*Client)
	for i := 0; i < n; i++ {
		m[i+g*100] = MakeMyClient(me+muGroup*100, RaftName, g, i, net)
	}
	return m
}

//me+g 为目标
func MakeGroupSerClient(muGroup, me, g int, n int, net *NetWork) map[int]*Client {
	m := make(map[int]*Client)
	for i := 0; i < n; i++ {
		m[i+g*100] = MakeMySerClient(me+muGroup*100, SerName, g, i, net)
	}
	return m
}
func MakeGroupSerClientAll(muGroup, me, gNum int, n int, net *NetWork) map[int]*Client {
	m := make(map[int]*Client)
	for j := 1;j<=gNum;j++{
		for i := 0; i < n; i++ {
			m[i+j*100] = MakeMySerClient(me+muGroup*100, SerName, j, i, net)
		}
	}

	return m
}

//me+g 为目标
func MakeGroupSerClientAsList(muGroup, me, g int, n int, net *NetWork) []*Client {
	m := make([]*Client, 0)
	for i := 0; i < n; i++ {
		m = append(m, MakeMySerClient(muGroup*100+me, SerName, g, i, net))
	}
	return m
}

func (net *NetWork) Disconnect(i, j int) {

}
