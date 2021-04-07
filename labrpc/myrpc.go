package labrpc

import (
	"fmt"
	"log"
	"net/rpc"
	"strconv"
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
	ClusterName string      //集群名字
	Port        string      //端口
	Num         int         //第几号机器
	client      *rpc.Client //client
}

func MakeMyClient(name string, i int) *Client {
	client := &Client{}
	client.ClusterName = name
	client.Port = "3000" + strconv.Itoa(i)
	client.Num = i
	return client
}

func (c *Client) Call(svcMeth string, args interface{}, reply interface{}) bool {
	if c.client == nil {
		client, err := rpc.Dial("tcp", "localhost:"+c.Port)
		if err != nil {
			log.Fatal("dialing:", err)
		}
		c.client = client
	}
	err := c.client.Call(c.ClusterName+"."+svcMeth, args, reply)
	if err != nil {
		fmt.Print("错误" + err.Error())
	}
	return true
}
