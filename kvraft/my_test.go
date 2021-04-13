package kvraft

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)
var step = 0
// get/put/putappend that keep counts
func MyGet(cfg *Myconfig, ck *Clerk, key string) string {
	v := ck.Get(key)
	cfg.op()
	return v
}

func MyPut(cfg *Myconfig, ck *Clerk, key string, value string) {
	ck.Put(key, value)
	cfg.op()
}

func MyAppend(cfg *Myconfig, ck *Clerk, key string, value string) {
	ck.Append(key, value)
	cfg.op()
}

func Mycheck(cfg *Myconfig, t *testing.T, ck *Clerk, key string, value string) {
	v := MyGet(cfg, ck, key)
	if v != value {
		t.Fatalf("Get(%v): expected:\n%v\nreceived:\n%v", key, value, v)
	}
}

func normal(t *testing.T, nclients int, unreliable bool,partitions bool) {

	title := "Test: "
	if unreliable {
		// the network drops RPC requests and replies.
		title = title + "unreliable net, "
	}
	if partitions {
		// the network may partition
		title = title + "partitions, "
	}
	fmt.Println("kv:"+title)


	cfg := make_myconfig(t, 3, -1, unreliable,partitions)
	ck := cfg.makeClient(cfg.All()) //ck
	// done_partitioner := int32(0)      //是否结束partition
	done_clients := int32(0) //是否结束整个client
	clnts := make([]chan int, nclients)
	for i := 0; i < nclients; i++ {
		clnts[i] = make(chan int)
	}
	for i := 0; i < 3; i++ {
		// log.Printf("Iteration %v\n", i)
		atomic.StoreInt32(&done_clients, 0)
		// atomic.StoreInt32(&done_partitioner, 0)
		go my_spawn_clients_and_wait(t, cfg, nclients, func(cli int, myck *Clerk, t *testing.T) {
			j := 0
			defer func() {
				clnts[cli] <- j
			}()
			last := ""
			key := strconv.Itoa(cli)
			MyPut(cfg, myck, key, last)
			for atomic.LoadInt32(&done_clients) == 0 {
				if (rand.Int() % 1000) < 500 {
					nv := "x " + strconv.Itoa(cli) + " " + strconv.Itoa(j) + " y"
					// log.Printf("%d: client new append %v\n", cli, nv)
					MyAppend(cfg, myck, key, nv)
					last = NextValue(last, nv)
					j++
				} else {
					// log.Printf("%d: client new get %v\n", cli, key)
					v := MyGet(cfg, myck, key)
					if v != last {
						log.Fatalf("get wrong value, key %v, wanted:\n%v\n, got\n%v\n", key, last, v)
					}
				}
			}
		})
		step ++
		fmt.Printf("before sleep  step := %d\n",step)
		time.Sleep(5 * time.Second)

		atomic.StoreInt32(&done_clients, 1) // tell clients to quit
		// atomic.StoreInt32(&done_partitioner, 1) // tell partitioner to quit
		step ++
		fmt.Printf("after sleep  step := %d\n",step)
		for i := 0; i < nclients; i++ {
			j := <-clnts[i]
			key := strconv.Itoa(i)
			v := MyGet(cfg, ck, key)
			checkClntAppends(t, i, v, j)
		}

	}
	cfg.end()

}
func TestCheckLeader(t *testing.T) {
	cfg := make_myconfig(t,3,-1,false,false)
	a := cfg.checkLeader()
	fmt.Println(a)
}
func TestNo1(t *testing.T) {
	normal(t, 3,false,false)
}
func TestMyUnreliable(t *testing.T) {
	normal(t, 3,true,false)
}
func TestMyPartition(t *testing.T) {
	normal(t, 3,false,true)
}

// // repartition the servers periodically
// func mypartitioner(t *testing.T, cfg *Myconfig, ch chan bool, done *int32) {
// 	defer func() { ch <- true }()
// 	for atomic.LoadInt32(done) == 0 {
// 		a := make([]int, cfg.n)
// 		for i := 0; i < cfg.n; i++ {
// 			a[i] = (rand.Int() % 2)
// 		}
// 		pa := make([][]int, 2)
// 		for i := 0; i < 2; i++ {
// 			pa[i] = make([]int, 0)
// 			for j := 0; j < cfg.n; j++ {
// 				if a[j] == i {
// 					pa[i] = append(pa[i], j)
// 				}
// 			}
// 		}
// 		cfg.partition(pa[0], pa[1])
// 		time.Sleep(electionTimeout + time.Duration(rand.Int63()%200)*time.Millisecond)
// 	}
// }

func my_spawn_clients_and_wait(t *testing.T, cfg *Myconfig, ncli int, fn func(me int, ck *Clerk, t *testing.T)) {
	ca := make([]chan bool, ncli)
	for cli := 0; cli < ncli; cli++ {
		ca[cli] = make(chan bool)
		go my_run_client(t, cfg, cli, ca[cli], fn)
	}
	// log.Printf("spawn_clients_and_wait: waiting for clients")
	for cli := 0; cli < ncli; cli++ {
		ok := <-ca[cli]
		// log.Printf("spawn_clients_and_wait: client %d is done\n", cli)
		if ok == false {
			t.Fatalf("failure")
		}
	}
}

// a client runs the function f and then signals it is done
func my_run_client(t *testing.T, cfg *Myconfig, me int, ca chan bool, fn func(me int, ck *Clerk, t *testing.T)) {
	ok := false
	defer func() { ca <- ok }()
	ck := cfg.makeClient(cfg.All())
	fn(me, ck, t)
	ok = true
	cfg.deleteClient(ck)
}
