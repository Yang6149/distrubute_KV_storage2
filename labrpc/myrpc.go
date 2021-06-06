package labrpc

import (
	"distrubute_KV_storage/tool"
	"fmt"
	"net/rpc"
	"strconv"
	"strings"
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

type TrueClient struct {
	myIp        string
	IP          string
	Port        int
	ClusterName string
	client      *rpc.Client //client
}
type NetWork struct {
	Unreliable  bool
	Partitions  bool
	Connect     map[int]map[int]bool
	ServerState map[int]bool
}

func (t TrueClient) GetIP() string {
	return t.IP + ":" + strconv.Itoa(t.Port)
}

func MakeTrueClient(ip string, port int, name string, myself string) *TrueClient {
	client := &TrueClient{}
	client.myIp = myself
	client.Port = port
	client.IP = ip
	client.ClusterName = name
	return client
}

func (c *TrueClient) Call(svcMeth string, args interface{}, reply interface{}) bool {
	for c.client == nil {
		fmt.Println("开始建立连接")
		client, err := rpc.Dial("tcp", c.IP+":"+strconv.Itoa(c.Port))
		fmt.Println("建立完成")
		if err != nil {
			fmt.Println(c, svcMeth)
			fmt.Println("dialing:", err)
			time.Sleep(time.Millisecond * 100)
		}
		c.client = client
	}

	err := c.client.Call(c.ClusterName+"."+svcMeth, args, reply)
	if err != nil {
		fmt.Printf("错误%v ,me is %s,target is %d\n", err.Error(), c.myIp, c.IP+":"+strconv.Itoa(c.Port))
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

type Clients struct {
	MasterN    int
	GroupN     int
	Num        int
	GroupsServ [][]*TrueClient
	GroupsRaft [][]*TrueClient
}

func MakeAllTrueClient(conf tool.Conf) *Clients {
	clients := &Clients{}
	clients.MasterN = conf.MasterN
	clients.GroupN = conf.GroupN
	clients.Num = conf.Num
	clients.GroupsServ = make([][]*TrueClient, conf.GroupN+1)
	clients.GroupsRaft = make([][]*TrueClient, conf.GroupN+1)
	clients.GroupsServ[0] = make([]*TrueClient, conf.Num)
	clients.GroupsRaft[0] = make([]*TrueClient, conf.Num)
	fmt.Println(conf.Group, conf.Num)
	for i := 0; i < clients.MasterN; i++ {
		masterIP := conf.MasterIP[i]
		strs := strings.Split(masterIP, " ")
		ip := strs[0]
		raftIp, err := strconv.Atoi(strs[1])
		servIp, err := strconv.Atoi(strs[2])
		if err != nil {

		}
		clients.GroupsServ[0][i] = MakeTrueClient(ip, servIp, "Serv", conf.Ip+strconv.Itoa(conf.Port))
		clients.GroupsRaft[0][i] = MakeTrueClient(ip, raftIp, "Raft", conf.Ip+strconv.Itoa(conf.RaftPort))
	}
	for i := 0; i < clients.GroupN; i++ {
		clients.GroupsRaft[i+1] = make([]*TrueClient, conf.Num)
		clients.GroupsServ[i+1] = make([]*TrueClient, conf.Num)
		for j := 0; j < clients.Num; j++ {
			masterIP := conf.Spec.Conditions[i].IP[j]
			strs := strings.Split(masterIP, " ")
			ip := strs[0]
			raftIp, err := strconv.Atoi(strs[1])
			servIp, err := strconv.Atoi(strs[2])
			if err != nil {

			}
			clients.GroupsRaft[i+1][j] = MakeTrueClient(ip, raftIp, "Raft", conf.Ip+strconv.Itoa(conf.RaftPort))
			clients.GroupsServ[i+1][j] = MakeTrueClient(ip, servIp, "Serv", conf.Ip+strconv.Itoa(conf.Port))
		}
	}
	return clients
}
