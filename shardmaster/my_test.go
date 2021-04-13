package shardmaster

import (
	"fmt"
	"testing"
)

func TestMyBasic(t *testing.T) {
	const nservers = 3
	cfg := make_myconfig(t, nservers)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	fmt.Printf("Test: Basic leave/join ...\n")

	cfa := make([]Config, 6)
	cfa[0] = ck.Query(-1)
	check(t, []int{}, ck)
	var gid1 int = 1
	ck.Join(map[int][]string{gid1: []string{"x", "y", "z"}})
	DPrintf("777777777777777")
	check(t, []int{gid1}, ck)
	DPrintf("000000000")
	cfa[1] = ck.Query(-1)
	DPrintf("11111111111111")
	var gid2 int = 2
	ck.Join(map[int][]string{gid2: []string{"a", "b", "c"}})
	check(t, []int{gid1, gid2}, ck)
	cfa[2] = ck.Query(-1)
	cfx := ck.Query(-1)
	sa1 := cfx.Groups[gid1]
	if len(sa1) != 3 || sa1[0] != "x" || sa1[1] != "y" || sa1[2] != "z" {
		t.Fatalf("wrong servers for gid %v: %v\n", gid1, sa1)
	}
	sa2 := cfx.Groups[gid2]
	if len(sa2) != 3 || sa2[0] != "a" || sa2[1] != "b" || sa2[2] != "c" {
		t.Fatalf("wrong servers for gid %v: %v\n", gid2, sa2)
	}

	ck.Leave([]int{gid1})
	check(t, []int{gid2}, ck)
	cfa[4] = ck.Query(-1)
	ck.Leave([]int{gid2})
	cfa[5] = ck.Query(-1)

	fmt.Printf("  ... Passed\n")

	fmt.Printf("Test: Historical queries ...\n")
	for s := 0; s < nservers; s++ {
		cfg.DisconnectServer(s)
		for i := 0; i < len(cfa); i++ {
			if s==2{
				fmt.Println("before ",i)
			}
			c := ck.Query(cfa[i].Num)
			if s==2{
				fmt.Println("c is ",c)
			}
			check_same_config(t, c, cfa[i])
		}
		fmt.Println("开启服务-test",s)
		cfg.ConnectServer(s)
	}
	fmt.Printf("  ... Passed\n")

	fmt.Printf("Test: Move ...\n")
	{
		var gid3 int = 503
		ck.Join(map[int][]string{gid3: []string{"3a", "3b", "3c"}})
		var gid4 int = 504
		ck.Join(map[int][]string{gid4: []string{"4a", "4b", "4c"}})
		for i := 0; i < NShards; i++ {
			cf := ck.Query(-1)
			if i < NShards/2 {
				ck.Move(i, gid3)
				if cf.Shards[i] != gid3 {
					cf1 := ck.Query(-1)
					if cf1.Num <= cf.Num {
						t.Fatalf("Move should increase Config.Num")
					}
				}
			} else {
				ck.Move(i, gid4)
				if cf.Shards[i] != gid4 {
					cf1 := ck.Query(-1)
					if cf1.Num <= cf.Num {
						t.Fatalf("Move should increase Config.Num")
					}
				}
			}
		}
		cf2 := ck.Query(-1)
		for i := 0; i < NShards; i++ {
			if i < NShards/2 {
				if cf2.Shards[i] != gid3 {
					t.Fatalf("expected shard %v on gid %v actually %v",
						i, gid3, cf2.Shards[i])
				}
			} else {
				if cf2.Shards[i] != gid4 {
					t.Fatalf("expected shard %v on gid %v actually %v",
						i, gid4, cf2.Shards[i])
				}
			}
		}
		ck.Leave([]int{gid3})
		ck.Leave([]int{gid4})
	}
	fmt.Printf("  ... Passed\n")

	fmt.Printf("Test: Concurrent leave/join ...\n")

	const npara = 10
	var cka [npara]*Clerk
	for i := 0; i < len(cka); i++ {
		cka[i] = cfg.makeClient()
	}
	gids := make([]int, npara)
	ch := make(chan bool)
	for xi := 0; xi < npara; xi++ {
		gids[xi] = int((xi * 10) + 100)
		go func(i int) {
			defer func() { ch <- true }()
			var gid int = gids[i]
			var sid1 = fmt.Sprintf("s%da", gid)
			var sid2 = fmt.Sprintf("s%db", gid)
			cka[i].Join(map[int][]string{gid + 1000: []string{sid1}})
			cka[i].Join(map[int][]string{gid: []string{sid2}})
			cka[i].Leave([]int{gid + 1000})
		}(xi)
	}
	for i := 0; i < npara; i++ {
		<-ch
	}
	check(t, gids, ck)

	fmt.Printf("  ... Passed\n")

	fmt.Printf("Test: Minimal transfers after joins ...\n")

	c1 := ck.Query(-1)
	for i := 0; i < 5; i++ {
		var gid = int(npara + 1 + i)
		ck.Join(map[int][]string{gid: []string{
			fmt.Sprintf("%da", gid),
			fmt.Sprintf("%db", gid),
			fmt.Sprintf("%db", gid)}})
	}
	c2 := ck.Query(-1)
	for i := int(1); i <= npara; i++ {
		for j := 0; j < len(c1.Shards); j++ {
			if c2.Shards[j] == i {
				if c1.Shards[j] != i {
					t.Fatalf("non-minimal transfer after Join()s")
				}
			}
		}
	}

	fmt.Printf("  ... Passed\n")

	fmt.Printf("Test: Minimal transfers after leaves ...\n")

	for i := 0; i < 5; i++ {
		ck.Leave([]int{int(npara + 1 + i)})
	}
	c3 := ck.Query(-1)
	for i := int(1); i <= npara; i++ {
		for j := 0; j < len(c1.Shards); j++ {
			if c2.Shards[j] == i {
				if c3.Shards[j] != i {
					t.Fatalf("non-minimal transfer after Leave()s")
				}
			}
		}
	}

	fmt.Printf("  ... Passed\n")
}
