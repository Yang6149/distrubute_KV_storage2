package main

import (
	"bufio"
	cli2 "distrubute_KV_storage/cli"
	"distrubute_KV_storage/shardkv"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"testing"
)

var(
	serve bool //服务端
	cli bool //客户端
	host string //host
	port int //占用端口
	serverName string
)
func main(){


	// &user 就是接收命令行中输入 -u 后面的参数值，其他同理
	flag.BoolVar(&serve, "s", false, "服务端使用")
	flag.BoolVar(&cli, "c", false, "客户端使用")
	flag.StringVar(&host, "h", "localhost", "主机名，默认为localhost")
	flag.IntVar(&port, "P", 10001, "端口号，默认为10001")
	flag.StringVar(&serverName, "serverName", "ShardKV", "默认rpc服务名为：ShardKV")
	// 解析命令行参数写入注册的flag里
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)


	// 输出结果
	if serve && cli{
		fmt.Println("不能同时为服务端和客户端")
		return
	}
	//服务端逻辑
	if serve{
		config := shardkv.Make_config(&testing.T{},3,false,-1)
		kvShard := cli2.MakeServer(config)
		config.Joinm([]int{0,1,2})
		go func() {
			rpc.RegisterName("ShardKV", kvShard)
			listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(port))
			fmt.Printf("shardkv serverName := %s \t listenerPort := %d\n", serverName, port)

			if err != nil {
				log.Fatal("ListenTCP error:", err)
			}
			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Fatal("Accept error:", err)
				}

				go rpc.ServeConn(conn)
			}
		}()
		defer config.Cleanup()
		for {}
	}
	// 用户端逻辑
	if cli{
		client ,err:= cli2.MakeClient(serverName,port)
		if err!=nil{
			return
		}
		fmt.Println("kvshard client Shell")
		fmt.Println("---------------------")
		fmt.Printf("Connect Server : %s,Port %d\n",serverName,port)
		fmt.Printf("Client Id : %d\n",client.Me)
		for {
			fmt.Print("-> ")
			text, _ := reader.ReadString('\n')
			// convert CRLF to LF
			text = strings.Replace(text, "\n", "", -1)
			text = strings.Replace(text, "\r", "", -1)
			commends := strings.Split(text," ")
			commend := commends[0]
			fmt.Printf("len := %d\n",len(commends))
			switch commend{
			case "Get":
				if len(commends)!=2{
					fmt.Println("Get need 1 argument Key")
				}
				client.Get(commends[1])
			case "get":
				if len(commends)!=2{
					fmt.Println("Get need 1 argument Key")
				}
				client.Get(commends[1])
			case "Put":
				if len(commends)!=3{
					fmt.Println("Put need 2 argument Key")
				}
				client.Put(commends[1],commends[2])
			case "put":
				if len(commends)!=3{
					fmt.Println("Put need 2 argument Key")
				}
				client.Put(commends[1],commends[2])
			case "Append":
				if len(commends)!=3{
					fmt.Println("Append need 2 argument Key")
				}
				client.Append(commends[1],commends[2])
			case "append":
				if len(commends)!=3{
					fmt.Println("Append need 2 argument Key")
				}
				client.Append(commends[1],commends[2])
			}

		}
	}
}