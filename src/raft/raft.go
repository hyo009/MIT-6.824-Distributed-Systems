package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"labrpc"
	"math/rand"
	"sync"
	"time"
)

// import "bytes"
// import "labgob"
type state int

const (
	// LEADER is leader state
	LEADER state = iota
	// CANDIDATE is candidate state
	CANDIDATE
	// FOLLOWER is follower state
	FOLLOWER

	// HBINTERVAL is haertbeat interval
	HBINTERVAL = 50 * time.Millisecond // 50ms
)

// ApplyMsg 是发送消息
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

// LogEntry is log entry
type LogEntry struct {
	LogIndex int
	LogTerm  int
	LogCmd   interface{}
}

// Raft implements a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// NOTICE: Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// Persistent state on call servers
	currentTerm int        // 此 server 当前所处的 term 编号
	votedFor    int        // 此 server 在此 term 中投票给了谁，是 peers 中的索引号
	logs        []LogEntry // 此 server 中保存的 logs

	// Volatile state on all servers:
	commitIndex int // logs 中已经 commited 的 log 的最大索引号
	lastApplied int // logs 中最新元素的索引号

	// Volatile state on leaders:
	nextIndex  []int // 下一个要发送给 follower 的 log 的索引号
	matchIndex []int // leader 与 follower 共有的 log 的最大的索引号

	state     state
	voteCount int

	chanCommit    chan struct{}
	chanHeartbeat chan struct{}
	chanGrantVote chan struct{}
	chanLeader    chan struct{}
	chanApply     chan ApplyMsg
}

// GetState 可以获取 raft 对象的状态
// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// NOTICE: Your code here (2A).

	term = rf.currentTerm

	isleader = rf.state == LEADER

	return term, isleader
}
func (rf *Raft) getLastIndex() int {
	return rf.logs[len(rf.logs)-1].LogIndex
}
func (rf *Raft) getLastTerm() int {
	return rf.logs[len(rf.logs)-1].LogTerm
}

// IsLeader 反馈 rf 是否是 Leader
func (rf *Raft) IsLeader() bool {
	return rf.state == LEADER
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// NOTICE: Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// NOTICE: Your code here (2C).
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
}

// RequestVoteArgs 获取投票参数
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// NOTICE: Your data here (2A, 2B).

	Term         int // candidate's term
	CandidateID  int // candidate requesting vote
	LastLogIndex int // index of candidate's last log entry
	LastLogTerm  int // term of candidate's last log entry
}

// RequestVoteReply 投票回复
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// NOTICE: Your data here (2A).

	Term        int  // 投票人的 currentTerm
	VoteGranted bool // 返回 true，表示获得投票
}

// RequestVote 投票工作
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// NOTICE: Your code here (2A, 2B).
	reply.Term = rf.currentTerm

	if rf.lastApplied <= args.LastLogIndex && rf.currentTerm <= args.LastLogTerm {
		reply.VoteGranted = true
		rf.votedFor = args.CandidateID
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// AppendEntriesArgs 是添加 log 的参数
type AppendEntriesArgs struct {
	Term         int // leader 的 term
	LeaderID     int // leader 的 ID
	PrevLogIndex int // index of log entry immediately preceding new ones
	PrevLogTerm  int // term of prevLogIndex entry

	Entries []LogEntry // 需要添加的 log 单元，为空时，表示此条消息是 heartBeat

	LeaderCommit int // leader 的 commitIndex
}

// AppendEntriesReply 是 flower 回复 leader 的内容
type AppendEntriesReply struct {
	term    int  // 回复者的 term
	success bool // 返回 true，如果回复者满足 prevLogIndex 和 prevLogTerm
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	return rf.peers[server].Call("Raft.AppendEntries", args, reply)
}

// Start 启动
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// NOTICE: Your code here (2B).

	return index, term, isLeader
}

// Kill is
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// NOTICE: Your code here, if desired.
}

// rf 为自己拉票，以便赢得选举
func (rf *Raft) canvass() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateID:  rf.me,
		LastLogTerm:  rf.getLastTerm(),
		LastLogIndex: rf.getLastIndex(),
	}

	for i := range rf.peers {
		if i != rf.me && rf.state == CANDIDATE {
			go func(i int) {
				var reply RequestVoteReply
				rf.sendRequestVote(i, &args, &reply)

				// NOTICE: 后续如何处理
			}(i)
		}
	}

	return
}

func (rf *Raft) boatcastAppendEntries() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	N := rf.commitIndex
	last := rf.getLastIndex()
	baseIndex := rf.logs[0].LogIndex
	for i := rf.commitIndex + 1; i <= last; i++ {
		num := 1
		for j := range rf.peers {
			if j != rf.me && rf.matchIndex[j] >= i && rf.logs[i-baseIndex].LogTerm == rf.currentTerm {
				num++
			}
		}
		if 2*num > len(rf.peers) {
			N = i
		}
	}
	if N != rf.commitIndex {
		rf.commitIndex = N
		rf.chanCommit <- struct{}{}
	}

	for i := range rf.peers {
		if i != rf.me && rf.state == LEADER {

			//copy(args.Entries, rf.log[args.PrevLogIndex + 1:])

			if rf.nextIndex[i] > baseIndex {
				var args AppendEntriesArgs
				args.Term = rf.currentTerm
				args.LeaderID = rf.me
				args.PrevLogIndex = rf.nextIndex[i] - 1
				//	fmt.Printf("baseIndex:%d PrevLogIndex:%d\n",baseIndex,args.PrevLogIndex )
				args.PrevLogTerm = rf.logs[args.PrevLogIndex-baseIndex].LogTerm
				//args.Entries = make([]LogEntry, len(rf.log[args.PrevLogIndex + 1:]))
				args.Entries = make([]LogEntry, len(rf.logs[args.PrevLogIndex+1-baseIndex:]))
				copy(args.Entries, rf.logs[args.PrevLogIndex+1-baseIndex:])
				args.LeaderCommit = rf.commitIndex
				go func(i int, args AppendEntriesArgs) {
					var reply AppendEntriesReply
					rf.sendAppendEntries(i, &args, &reply)
				}(i, args)
			} else {
				var args InstallSnapshotArgs
				args.Term = rf.currentTerm
				args.LeaderID = rf.me
				args.LastIncludedIndex = rf.logs[0].LogIndex
				args.LastIncludedTerm = rf.logs[0].LogTerm
				args.Data = rf.persister.snapshot
				go func(server int, args InstallSnapshotArgs) {
					reply := &InstallSnapshotReply{}
					rf.sendInstallSnapshot(server, args, reply)
				}(i, args)
			}
		}
	}
}

// InstallSnapshotArgs is
type InstallSnapshotArgs struct {
	Term              int
	LeaderID          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

// InstallSnapshotReply is
type InstallSnapshotReply struct {
	Term int
}

func (rf *Raft) sendInstallSnapshot(server int, args InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	if ok {
		if reply.Term > rf.currentTerm {
			rf.currentTerm = reply.Term
			rf.state = FOLLOWER
			rf.votedFor = -1
			return ok
		}

		rf.nextIndex[server] = args.LastIncludedIndex + 1
		rf.matchIndex[server] = args.LastIncludedIndex
	}
	return ok
}

// Make is
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
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// NOTICE: Your initialization code here (2A, 2B, 2C).

	rf.state = FOLLOWER
	rf.votedFor = -1
	rf.logs = append(rf.logs, LogEntry{})
	rf.currentTerm = 0

	rf.chanCommit = make(chan struct{}, 100)
	rf.chanHeartbeat = make(chan struct{}, 100)
	rf.chanGrantVote = make(chan struct{}, 100)
	rf.chanLeader = make(chan struct{}, 100)
	rf.chanApply = applyCh

	go func() {
		for {
			switch rf.state {
			case FOLLOWER:
				select {
				case <-rf.chanHeartbeat:
				case <-rf.chanGrantVote:
				case <-time.After(time.Millisecond * time.Duration(rand.Int63()%333+550)):
					rf.state = CANDIDATE
				}
			case LEADER:
				rf.boatcastAppendEntries()
				time.Sleep(HBINTERVAL)
			case CANDIDATE:
				rf.mu.Lock()
				rf.currentTerm++
				rf.votedFor = rf.me
				rf.voteCount = 1
				rf.persist()
				rf.mu.Unlock()
				go rf.canvass()
				select {
				case <-time.After(time.Millisecond * time.Duration(rand.Int63()%333+550)):
				case <-rf.chanHeartbeat:
					rf.state = FOLLOWER
				case <-rf.chanLeader:
					rf.mu.Lock()
					rf.state = LEADER
					rf.nextIndex = make([]int, len(rf.peers))
					rf.matchIndex = make([]int, len(rf.peers))
					for i := range rf.peers {
						rf.nextIndex[i] = rf.getLastIndex() + 1
					}
					rf.mu.Unlock()
				}
			}
		}
	}()

	go func() {
		for {

			select {
			case <-rf.chanCommit:
				rf.mu.Lock()
				commitIndex := rf.commitIndex
				baseIndex := rf.logs[0].LogIndex
				for i := rf.lastApplied + 1; i <= commitIndex; i++ {
					msg := ApplyMsg{CommandIndex: i, Command: rf.logs[i-baseIndex].LogCmd}
					applyCh <- msg
					//fmt.Printf("me:%d %v\n",rf.me,msg)
					rf.lastApplied = i
				}
				rf.mu.Unlock()
			}
		}
	}()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	return rf
}

func (rf *Raft) heartBeat() {

	return
}
