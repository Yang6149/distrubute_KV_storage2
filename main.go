package main

import (
	"bufio"
	"distrubute_KV_storage/labrpc"
	"distrubute_KV_storage/raft"
	"distrubute_KV_storage/shardkv"
	"distrubute_KV_storage/shardmaster"
	"distrubute_KV_storage/tool"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	serve      bool   //服务端
	cli        bool   //客户端
	host       string //host
	serverName string
	tiaoshi    bool
	raftPort   int
	Port       int
	me         int
	group      int
)

func main() {

	// &user 就是接收命令行中输入 -u 后面的参数值，其他同理
	flag.BoolVar(&serve, "s", false, "服务端使用")
	flag.BoolVar(&cli, "c", false, "客户端使用")
	flag.StringVar(&host, "h", "localhost", "主机名，默认为localhost")
	flag.StringVar(&serverName, "serverName", "ShardKV", "默认rpc服务名为：ShardKV")
	flag.BoolVar(&tiaoshi, "t", false, "是否调试")
	flag.IntVar(&raftPort, "rP", 10001, "端口号，默认为10001")
	flag.IntVar(&Port, "P", 10001, "")
	flag.IntVar(&me, "m", 0, "")
	flag.IntVar(&group, "g", 0, "")
	// 解析命令行参数写入注册的flag里
	flag.Parse()

	var c tool.Conf
	c.GetConf()
	if tiaoshi {
		fmt.Println("進入調試")
		c.RaftPort = raftPort
		c.Port = Port
		c.Id = me
		c.IsMasterClient = false
		c.Group = group
	}
	fmt.Println(c.Port)
	fmt.Println(c.IsMaster)
	fmt.Println(c)
	clients := labrpc.MakeAllTrueClient(c)
	per := raft.MakePersister()
	fmt.Println("isMasterClient", c.IsMasterClient)
	fmt.Println("isClient", c.IsClient)
	if c.IsMasterClient {

		client := shardkv.MakeClerk(clients)

		clerk := shardmaster.MakeClerk(clients)

		reader := bufio.NewReader(os.Stdin)

		// 用户端逻辑
		if true {

			fmt.Println("kvshard client Shell")
			fmt.Println("---------------------")
			for {
				fmt.Print("-> ")
				text, _ := reader.ReadString('\n')
				// convert CRLF to LF
				text = strings.Replace(text, "\n", "", -1)
				text = strings.Replace(text, "\r", "", -1)
				commends := strings.Split(text, " ")
				commend := commends[0]
				switch commend {
				case "Get":
					if len(commends) != 2 {
						fmt.Println("Get need 1 argument Key")
						continue
					}
					client.Get(commends[1])
				case "get":
					if len(commends) != 2 {
						fmt.Println("Get need 1 argument Key")
						continue
					}
					client.Get(commends[1])
				case "Put":
					if len(commends) != 3 {
						fmt.Println("Put need 2 argument Key")
						continue
					}
					client.Put(commends[1], commends[2])
				case "put":
					if len(commends) != 3 {
						fmt.Println("Put need 2 argument Key")
						continue
					}
					client.Put(commends[1], commends[2])
				case "Append":
					if len(commends) != 3 {
						fmt.Println("Append need 2 argument Key")
						continue
					}
					client.Append(commends[1], commends[2])
				case "append":
					if len(commends) != 3 {
						fmt.Println("Append need 2 argument Key")
						continue
					}
					client.Append(commends[1], commends[2])
				case "info":
					if len(commends) != 1 {
						fmt.Println("info need 1 argument Key")
						continue
					}
					fmt.Println(clerk.EasyQuery())
				case "join":
					if len(commends) != 2 {
						fmt.Println("info need 2 argument Key")
						continue
					}
					int, err := strconv.Atoi(commends[1])
					if err != nil {
						fmt.Println("输出请为0-2的数字")
						continue
					}
					clerk.EasyJoin(int)
				case "leave":
					if len(commends) != 2 {
						fmt.Println("info need 2 argument Key")
						continue
					}
					int, err := strconv.Atoi(commends[1])
					if err != nil {
						fmt.Println("输出请为0-2的数字")
						continue
					}
					clerk.EasyLeave(int)

				}

			}
		}

		clerk.EasyJoin(1)
		clerk.EasyJoin(2)
		clerk.EasyJoin(3)
		fmt.Print(clerk.Query(-1))

		for i := 0; i < 10; i++ {
			client.Put(strconv.Itoa(i), strconv.Itoa(i))
			if client.Get(strconv.Itoa(i)) != strconv.Itoa(i) {
				fmt.Println("fail")
				break
			}
			fmt.Println("finish")
		}
	} else {
		if c.IsMaster {
			//启动服务
			shardmaster.StartServer(clients, c, c.Id, c.Group, per)
		} else {
			shardkv.StartServer(clients, c, c.Id, per, -1, c.Group)
		}
	}

	// labrpc.MakeGroupRaftClient()

	for {

	}
}
