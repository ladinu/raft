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
			if msg.Term >= self.CurrentTerm {
				self.CurrentTerm = msg.Term
				if self.State != follower {
					self.State = follower
					fmt.Printf("Node '%v' became FOLLOWER with term %v. Message from leader '%v' \n", self.ID, self.CurrentTerm, msg.LeaderID)
				}
				electionTimer.Reset(utils.RandomDuration())
				// fmt.Printf("Heartbeat from leader '%v' with term %v\n", msg.LeaderID, msg.Term)
			}
		case msg := <-requestVoteC:
			if msg.Term > self.CurrentTerm {
				self.CurrentTerm = msg.Term
				self.State = follower
				self.VotedFor = msg.CandidateID
				electionTimer.Reset(utils.RandomDuration())
				fmt.Printf("Node '%v' became FOLLOWER with term %v and voted for Node '%v'\n", self.ID, self.CurrentTerm, msg.CandidateID)
				go func() { msg.ReplyChan <- RequestVoteRPCReply{self.CurrentTerm, true} }()
			} else {
				go func() { msg.ReplyChan <- RequestVoteRPCReply{self.CurrentTerm, false} }()
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
					paceMaker := time.NewTicker(time.Second * 1)
					fmt.Printf("Node '%v' became LEADER with term %v\n", self.ID, self.CurrentTerm)
					// boom := 0
					for _ = range paceMaker.C {
						fmt.Printf("❤︎ ")
						// if boom > -1 {
						go func() { appendEntriesC <- AppendEntriesRPC{self.CurrentTerm, self.ID} }()
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
			go func() { requestVoteC <- RequestVoteRPC{self.CurrentTerm, self.ID, voteReplyC} }()
		}
	}
}

func main() {
	// var clusterC = make(chan rpc)
	// go node(clusterC)
	var s string
	var appendEntriesC = make(chan AppendEntriesRPC)
	var requestVoteC = make(chan RequestVoteRPC)
	var c = 4

	for index := 1; index <= c; index++ {
		go node(index, appendEntriesC, requestVoteC, c)
	}

	fmt.Scanln(&s)
}
