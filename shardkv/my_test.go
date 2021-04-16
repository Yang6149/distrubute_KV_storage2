package shardkv


import(
	"distrubute_KV_storage/labrpc"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestTime(t2 *testing.T){
	fmt.Println("Hello, playground")
	t := time.Now()
	fmt.Println(t.Month())
	fmt.Println(t.Day())
}


func TestMyStaticShards(t *testing.T) {
	fmt.Printf("Test: static shards ...\n")

	cfg := MakeMyConfig(5,3,3)
	defer cfg.cleanup()

	ck := cfg.makeClient(10)
	fmt.Println("123123123123123123123")
	cfg.join(0)
	cfg.join(1)
	//fmt.Println("join 100 and 101")
	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		//fmt.Println("1+", i)
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = strconv.Itoa(i)
		//fmt.Println("put", ka[i], "->", va[i])
		ck.Put(ka[i], va[i])
	}
	//fmt.Println("1111111111111111111111111111111111")
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i])
	}
	//fmt.Println("2222222222222222222222222222222222")
	for i := 0; i < n; i++ {
		v := strconv.Itoa(i)
		ck.Append(ka[i], v)
		va[i] += v
	}
	//fmt.Println("3333333333333333333333333333333333")
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i])
	}
	//fmt.Println("4444444444444444444444444444444444")
	// make sure that the data really is sharded by
	// shutting down one shard and checking that some
	// Get()s don't succeed.
	fmt.Println("before ShutdownGroup***********************************")
	//cfg.ShutdownGroup(1)
	cfg.checklogs() // forbid snapshots

	ch := make(chan bool)
	for xi := 0; xi < n; xi++ {
		ck1 := cfg.makeClient(xi) // only one call allowed per client
		go func(i int) {
			defer func() { ch <- true }()
			check(t, ck1, ka[i], va[i])
		}(xi)
	}

	// wait a bit, only about half the Gets should succeed.
	ndone := 0
	done := false
	for done == false {
		select {
		case <-ch:
			ndone += 1
		case <-time.After(time.Second * 2):
			done = true
			break
		}
	}

	if ndone != 5 {
		t.Fatalf("expected 5 completions with one shard dead; got %v\n", ndone)
	}

	// bring the crashed shard/group back to life.
	//cfg.StartGroup(1)
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i])
	}

	fmt.Printf("  ... Passed\n")
}

func (cfg *my_config) makeClient(i int) *Clerk {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	me := 1000+i
	cfg.svrclients[me] = labrpc.MakeGroupSerClientAll(10,i,cfg.ngroups,cfg.n,cfg.net)
	ck := MakeClerk(cfg.svrclients,me,cfg.nmasters,cfg.net)
	cfg.nextClientId++
	return ck
}

func (cfg *my_config) join(gi int){
	cfg.joinm([]int{gi})

}

func (cfg *my_config) joinm(gis []int) {
	m := make(map[int][]string, len(gis))
	for _, g := range gis {
		gid := cfg.groups[g].gid
		servernames := make([]string, cfg.n)
		for i := 0; i < cfg.n; i++ {
			servernames[i] = cfg.servername(gid, i)
		}
		m[gid] = servernames
	}
	cfg.mck.Join(m)
}
func (cfg *my_config) servername(gid int, i int) string {
	return "server-" + strconv.Itoa(gid) + "-" + strconv.Itoa(i)
}