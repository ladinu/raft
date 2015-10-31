package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ladinu/raft/utils"
)

const (
	leader    = "leader"
	follower  = "follower"
	candidate = "candidate"
)

type rpc struct {
	Type   string
	Leader string
	Term   int
	C      chan string
}

// NodeState is a struct that contain the state of a given node
type NodeState struct {
	CurrentTerm int
	VotedFor    int
	State       string
	ID          int
}

// RequestVoteRPC struct
type RequestVoteRPC struct {
	Term        int
	CandidateID int
	ReplyChan   chan RequestVoteRPCReply
}

// RequestVoteRPCReply struct
type RequestVoteRPCReply struct {
	Term        int
	VoteGranted bool
}

// AppendEntriesRPC struct
type AppendEntriesRPC struct {
	Term     int
	LeaderID int
}

func node(id int, appendEntriesC chan AppendEntriesRPC, requestVoteC chan RequestVoteRPC, peerCount int) {
	var electionTimer = time.NewTimer(utils.RandomDuration())
	var self = NodeState{0, -1, follower, id}
	var voteReplyC = make(chan RequestVoteRPCReply)
	var votes = 0
	fmt.Printf("Node '%v' became FOLLOWER with term %v\n", self.ID, self.CurrentTerm)

	for {
		select {
		case msg := <-appendEntriesC:
			if msg.Term > self.CurrentTerm {
				self.CurrentTerm = msg.Term
				if self.State != follower {
					self.State = follower
					fmt.Printf("Node '%v' became FOLLOWER with term %v. Message from leader '%v' \n", self.ID, self.CurrentTerm, msg.LeaderID)
				}
				electionTimer.Reset(utils.RandomDuration())
				// fmt.Printf("Heartbeat from leader '%v' with term %v\n", msg.LeaderID, msg.Term)
			}
		case msg := <-requestVoteC:
			fmt.Printf("Node '%v' got message from node %v\n", self.ID, msg.CandidateID)
			if msg.Term > self.CurrentTerm {
				self.CurrentTerm = msg.Term
				self.State = follower
				self.VotedFor = msg.CandidateID
				electionTimer.Reset(utils.RandomDuration())
				fmt.Printf("Node '%v' became FOLLOWER with term %v and voted for Node '%v'\n", self.ID, self.CurrentTerm, msg.CandidateID)
				go func(term int) { msg.ReplyChan <- RequestVoteRPCReply{term, true} }(self.CurrentTerm)
			} else {
				go func(term int) { msg.ReplyChan <- RequestVoteRPCReply{term, false} }(self.CurrentTerm)
			}
		case response := <-voteReplyC:
			if response.Term > self.CurrentTerm {
				self.CurrentTerm = response.Term
				self.State = follower
				electionTimer.Reset(utils.RandomDuration())
				fmt.Printf("Node '%v' became FOLLOWER with term %v\n", self.ID, self.CurrentTerm)
			} else {
				if response.VoteGranted {
					votes = votes + 1
				}
				if votes >= int(math.Floor(float64(peerCount)/2.0)+1.0) {
					self.State = leader
					electionTimer.Stop()
					paceMaker := time.NewTicker(time.Millisecond * 500)
					fmt.Printf("Node '%v' became LEADER with term %v\n", self.ID, self.CurrentTerm)
					// boom := 0
					for _ = range paceMaker.C {
						fmt.Printf("❤︎ ")
						// if boom > -1 {
						go func(term, id int) { appendEntriesC <- AppendEntriesRPC{term, id} }(self.CurrentTerm, self.ID)
						// boom = boom + 1
						// } else {
						// 	fmt.Println("Leader EXPLODEDE!!!")
						// 	paceMaker.Stop()
						// 	<-make(chan string)
						// }
					}
				}
			}
		case <-electionTimer.C:
			self.State = candidate
			self.CurrentTerm = self.CurrentTerm + 1
			votes = 1
			self.VotedFor = self.ID
			electionTimer.Reset(utils.RandomDuration())
			fmt.Printf("Node '%v' timedout and became CANDIDATE with term %v\n", self.ID, self.CurrentTerm)
			go func(term int, id int, c chan RequestVoteRPCReply) {
				requestVoteC <- RequestVoteRPC{term, id, c}
			}(self.CurrentTerm, self.ID, voteReplyC)
		default:
		}
	}
}

func main() {
	// var clusterC = make(chan rpc)
	// go node(clusterC)
	var s string
	// var appendEntriesC = make(chan AppendEntriesRPC)
	// var requestVoteC = make(chan RequestVoteRPC)
	var c = 5

	// for index := 1; index <= c; index++ {
	// 	go node(index, appendEntriesC, requestVoteC, c)
	// }
	var cluster = utils.MakeCluster(c)

	var raft = func(id int, voteC chan utils.RequestVote, appendC chan utils.AppendEntries, cluster *utils.Cluster) {
		fmt.Printf("RNode %v\n", id)
		var electionTimer = time.NewTimer(utils.RandomDuration())
		for {
			select {
			case msg := <-voteC:
				fmt.Printf("Vote request from %v\n", msg.CandidateID)
			case <-electionTimer.C:
				fmt.Printf("RNode %v TIMEOUT\n", id)
				cluster.RequestVotes(id, utils.RequestVote{0, id, make(chan utils.RequestVoteReplyC)})
				electionTimer.Reset(utils.RandomDuration())
			default:
			}
		}
	}

	var node = func(id int, c *utils.Cluster) utils.NodeChan {
		var requestVoteC = make(chan utils.RequestVote)
		var appendEntriesC = make(chan utils.AppendEntries)
		var nodeChan = utils.NodeChan{id, requestVoteC, appendEntriesC}
		go raft(id, requestVoteC, appendEntriesC, cluster)
		return nodeChan
	}

	for idx := 0; idx < c; idx++ {
		cluster.Join(node(idx, cluster))
	}

	// var raft = func(global chan string, id int) {
	// 	fmt.Printf("Node %v\n", id)
	// 	for {
	// 		select {
	// 		case <-global:
	// 			fmt.Printf("Node %v got message from com0\n", id)
	// 		default:
	// 		}
	// 	}
	// }
	//
	// var unit = func(id int, global chan string) chan string {
	// 	var self = make(chan string)
	// 	go raft(self, id)
	// 	return self
	// }
	//
	// var com0 = make(chan string)
	// var peersChans = make([]chan string, c)
	//
	// for index := range peersChans {
	// 	peersChans[index] = unit(index+1, com0)
	// }
	//
	// var broadCast = func() {
	// 	for {
	// 		select {
	// 		case <-com0:
	// 			for _, peerChan := range peersChans {
	// 				go func(pc chan string) { pc <- "" }(peerChan)
	// 			}
	// 		default:
	// 		}
	// 	}
	// }
	//
	// go broadCast()
	//
	// paceMaker := time.NewTicker(time.Millisecond * 3000)
	// for _ = range paceMaker.C {
	// 	fmt.Println("TIMEOUT")
	// 	go func() { com0 <- "foo" }()
	// }

	fmt.Scanln(&s)
}
