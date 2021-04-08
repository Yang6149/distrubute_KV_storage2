package raft

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"distrubute_KV_storage/labgob"
	"distrubute_KV_storage/labrpc"
)

// import "bytes"

//
// A Go object implementing a single Raft peer.
//
const leader = 2
const follower = 0
const candidate = 1
const heartbeatConstTime = 50 * time.Millisecond
const isDan = true

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}
type HBchs struct {
	c chan int
}

//raft comment
type Raft struct {
	mu         sync.Mutex // Lock to protect shared access to this peer's state
	LeaderCond sync.Cond
	peers      []*labrpc.ClientEnd // RPC end points of all peers
	client     []*labrpc.Client    // true RPC client
	persister  *Persister          // Object to hold this peer's persisted state
	me         int                 // this peer's index into peers[]
	dead       int32               // set by Kill()
	state      int

	//持久化
	currentTerm int     // 当前的 term
	voteFor     int     // 为某人投票
	menkan      int     //大多数的一个阈值
	log         []Entry //logEntries

	//log
	//所有可变属性
	commitIndex int //	当前 commit 的index
	lastApplied int //	本机apply的最大的log的index

	//可变 on leader
	nextIndex  []int //	下一个要发送给server的logEntry的index
	matchIndex []int //	直到server 中各个log的index到哪了

	voteGrantedChan chan int
	appendChan      chan int
	findBiggerChan  chan int
	applyCh         chan ApplyMsg
	sendApply       chan int
	heartBeatchs    []HBchs
	SnapshotF       chan int
	// Your data here (2A, 2B, 2C).-------------------------------------
	//3B
	lastIncludedIndex int
	lastIncludedTerm  int
}

// GetState get command .
// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	//log.Println("GetState")
	var term int
	var isleader bool
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.state == leader
	// Your code here (2A).------------------------------------------
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.voteFor)
	e.Encode(rf.log)
	e.Encode(rf.lastIncludedIndex)
	e.Encode(rf.commitIndex)
	e.Encode(rf.lastIncludedTerm)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var voteFor int
	var log []Entry
	var lastIncludedIndex int
	var lastIncludedTerm int
	var commitIndex int
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&voteFor) != nil ||
		d.Decode(&log) != nil ||
		d.Decode(&lastIncludedIndex) != nil ||
		d.Decode(&commitIndex) != nil ||
		d.Decode(&lastIncludedTerm) != nil {
	} else {
		rf.currentTerm = currentTerm
		rf.voteFor = voteFor
		rf.log = log
		rf.lastIncludedIndex = lastIncludedIndex
		rf.commitIndex = commitIndex
		rf.lastIncludedTerm = lastIncludedTerm
		if rf.lastIncludedTerm > 500 {
			DPrintf("%d 有问题1", rf.me)
			DPrintf("问题是 lastIncludedIndex = %d", lastIncludedIndex)
			DPrintf("问题是 commitIndex = %d", commitIndex)
			DPrintf("问题是 lastIncludedTerm = %d", lastIncludedTerm)
		}
	}
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state != leader {
		return -1, -1, false
	} else {
		entry := Entry{Term: rf.currentTerm, Command: command}
		rf.log = append(rf.log, entry) //向log 中加入client 最新的request
		rf.persist()
		DPrintf("%d add a command:%d at index %d", rf.me, command, rf.logLen()-1)
		if isDan {
			for i := range rf.client {
				if i == rf.me || len(rf.heartBeatchs[i].c) > 0 {
					continue
				}
				rf.heartBeatchs[i].c <- 1
			}
		}

		return rf.logLen() - 1, rf.currentTerm, true

	}
}

func (rf *Raft) Discard(index int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("%d 调用discard %d", rf.me, index)
	if index <= rf.lastIncludedIndex {
		DPrintf("%d 调用discard 失败%d %d", rf.me, index, rf.lastIncludedIndex)
		return
	}
	if index+1 >= rf.logLen() {
		DPrintf("%d 调用discard全删 %d len = %d", rf.me, index, rf.logLen())
		//fmt.Println(rf.me, "删掉多的", index)
		//fmt.Println(rf.me,rf.nextIndex)
		//如果index+1 > rf.logLen()，说明是传来的snapshot，在appendEntry中会进行统一的状态处理
		//及使是传来的snapshot，也可能是==的结果，先处理一遍，等下在后面reply中还会再处理一遍，所以这里主要是进行自我snapshot的处理
		if index+1 == rf.logLen() {
			rf.lastIncludedTerm = rf.logTerm(index)
			if rf.lastIncludedTerm > 500 {
				DPrintf("%d 有问题2", rf.me)
			}
			rf.lastIncludedIndex = index
			rf.commitIndex = max(rf.commitIndex, rf.lastIncludedIndex)
		}
		rf.lastIncludedIndex = index
		rf.commitIndex = max(rf.commitIndex, rf.lastIncludedIndex)
		DPrintf("%d Term->%d", rf.me, rf.lastIncludedTerm)
		rf.log = make([]Entry, 0)
		rf.persist()
		return
	}
	DPrintf("%d 调用discard正常 %d", rf.me, index)
	term := rf.logTerm(index)
	rf.logDiscard(index)
	rf.lastIncludedIndex = index
	rf.commitIndex = max(rf.commitIndex, rf.lastIncludedIndex)
	rf.lastIncludedTerm = term
	if rf.lastIncludedTerm > 500 {
		DPrintf("%d 有问题3", rf.me)
	}
	// if len(rf.nextIndex) != 0 {
	// 	for i := range rf.peers {
	// 		if i == rf.me {
	// 			continue
	// 		}
	// 		if i >= len(rf.nextIndex) {
	// 			fmt.Println(i, rf.nextIndex)
	// 			fmt.Println("!!!")
	// 		}
	// 		//rf.nextIndex[i]-1=preIndex
	// 		//pre-lastIncludex-1 = realIndex
	// 		if rf.nextIndex[i]-1-rf.lastIncludedIndex-1 >= len(rf.log) {
	// 			fmt.Println("--", rf.nextIndex[i], rf.lastIncludedIndex, len(rf.log))
	// 			fmt.Println(rf.log[rf.nextIndex[i]-1-rf.lastIncludedIndex-1])
	// 		}
	// 	}
	// }

	rf.persist()
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	//rpc.RegisterName("HelloService", new(Raft))
	rf.peers = peers
	go func() {
		rpc.RegisterName("Server"+strconv.Itoa(me)+"Raft", rf)
		listener, err := net.Listen("tcp", ":3000"+strconv.Itoa(me))
		fmt.Printf("serverName := %s \t listener := "+":3000"+strconv.Itoa(me)+"\n", "Server"+strconv.Itoa(me)+"Raft")
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
	rf.persister = persister
	rf.me = me
	rf.menkan = len(rf.client)/2 + 1
	rf.applyCh = applyCh
	rf.currentTerm = 1
	rf.log = make([]Entry, 1)
	rf.log[0] = Entry{Term: rf.currentTerm}
	rf.sendApply = make(chan int, 1000)
	rf.heartBeatchs = make([]HBchs, len(rf.client))
	//3B
	rf.lastIncludedIndex = -1
	rf.lastIncludedTerm = 0
	for i := range rf.heartBeatchs {
		rf.heartBeatchs[i].c = make(chan int, 1000)
	}
	rf.chanReset()
	// Your initialization code here (2A, 2B, 2C).-------------------------------

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	go func(rf *Raft) {
		for {
			if rf.killed() {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				rf.convert(follower)
				return
			}
			rf.mu.Lock()
			st := rf.state
			rf.mu.Unlock()
			switch st {
			case follower:
				//
				select {
				case <-rf.findBiggerChan:
				case <-rf.appendChan:
					//follower收到有效append，重置超时
				case <-time.After(electionConstTime()):
					//超时啦，进行选举
					rf.mu.Lock()
					rf.chanReset()
					rf.convert(candidate)
					rf.election()
					rf.mu.Unlock()
				}
			case candidate:
				select {
				case <-rf.findBiggerChan:

					//发现了更大地 term ，转为follower
				case <-rf.appendChan:
					//candidate 收到有效心跳，转回follower
				case <-rf.voteGrantedChan:
					//candidate 收到多数投票结果，升级为 leader
				case <-time.After(electionConstTime()):
					//没有投票结果，也没有收到有效append，重新giao
					rf.mu.Lock()
					rf.chanReset()
					rf.election()
					rf.mu.Unlock()
				}
				//
			case leader:
				select {
				case <-rf.findBiggerChan:

				case <-rf.appendChan:
					//收到有效地心跳，转为follower
				case <-time.After(heartbeatConstTime):
					// 	//进行一次append
					// rf.mu.Lock()
					// rf.chanReset()
					// rf.heartBeat()
					// rf.mu.Unlock()
				}

			}
		}
	}(rf)
	go rf.apply()
	return rf
}
func Make2(client []*labrpc.Client, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	//rpc.RegisterName("HelloService", new(Raft))
	go func() {
		rpc.RegisterName(client[me].ClusterName, rf)
		listener, err := net.Listen("tcp", "localhost:"+client[me].Port)
		fmt.Printf("serverName := %s \t listener := %s\n", client[me].ClusterName, client[me].Port)
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
	rf.persister = persister
	rf.me = me
	rf.client = client
	rf.menkan = len(rf.client)/2 + 1
	rf.applyCh = applyCh
	rf.currentTerm = 1
	rf.log = make([]Entry, 1)
	rf.log[0] = Entry{Term: rf.currentTerm}
	rf.sendApply = make(chan int, 1000)
	rf.heartBeatchs = make([]HBchs, len(rf.client))
	//3B
	rf.lastIncludedIndex = -1
	rf.lastIncludedTerm = 0
	for i := range rf.heartBeatchs {
		rf.heartBeatchs[i].c = make(chan int, 1000)
	}
	rf.chanReset()
	// Your initialization code here (2A, 2B, 2C).-------------------------------

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	go func(rf *Raft) {
		for {
			if rf.killed() {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				rf.convert(follower)
				return
			}
			rf.mu.Lock()
			st := rf.state
			rf.mu.Unlock()

			switch st {
			case follower:
				//
				select {
				case <-rf.findBiggerChan:
				case <-rf.appendChan:
					//follower收到有效append，重置超时
				case <-time.After(electionConstTime()):
					//超时啦，进行选举
					rf.mu.Lock()
					rf.chanReset()
					rf.convert(candidate)
					rf.election()
					rf.mu.Unlock()
				}
			case candidate:
				select {
				case <-rf.findBiggerChan:

					//发现了更大地 term ，转为follower
				case <-rf.appendChan:
					//candidate 收到有效心跳，转回follower
				case <-rf.voteGrantedChan:
					//candidate 收到多数投票结果，升级为 leader
				case <-time.After(electionConstTime()):
					//没有投票结果，也没有收到有效append，重新giao
					rf.mu.Lock()
					rf.chanReset()
					rf.election()
					rf.mu.Unlock()
				}
				//
			case leader:
				select {
				case <-rf.findBiggerChan:

				case <-rf.appendChan:
					//收到有效地心跳，转为follower
				case <-time.After(heartbeatConstTime):
					// 	//进行一次append
					// rf.mu.Lock()
					// rf.chanReset()
					// rf.heartBeat()
					// rf.mu.Unlock()
				}

			}
		}
	}(rf)
	go rf.apply()
	return rf
}

func electionConstTime() time.Duration {
	return time.Duration(150+rand.Intn(300)) * time.Millisecond
}

func (rf *Raft) apply() {
	for {
		select {
		case index := <-rf.sendApply:
			fmt.Printf("%d %d ************************************\n", rf.me, index)

			for i := rf.lastApplied + 1; i <= index; i++ {
				rf.mu.Lock()
				if i <= rf.lastIncludedIndex {
					DPrintf("%d已经接受snapshot，continue", rf.me)
					rf.mu.Unlock()
					continue
				}
				command := rf.logGet(i).Command
				rf.mu.Unlock()
				msg := ApplyMsg{
					CommandValid: true,
					Command:      command,
					CommandIndex: i,
				}

				rf.applyCh <- msg
				rf.lastApplied = i
			}
		}
	}
}

//3B
func (rf *Raft) GetStateSize() int {
	return rf.persister.RaftStateSize()
}

func (rf *Raft) SaveSnapshot(snapshots []byte) {
	DPrintf("%d 进行一次 snapshot", rf.me)
	state := rf.persister.ReadRaftState()
	rf.persister.SaveStateAndSnapshot(state, snapshots)
}
func (rf *Raft) GetSnapshots() []byte {
	return rf.persister.ReadSnapshot()
}
