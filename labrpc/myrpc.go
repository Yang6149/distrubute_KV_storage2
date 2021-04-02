package labrpc

import "net/rpc"

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
