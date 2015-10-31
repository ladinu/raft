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

// NodeState is a struct that contain the state of a given node
type NodeState struct {
	CurrentTerm int
	VotedFor    int
	State       string
	ID          int
	Votes       int
}

func raft(id int, voteC chan utils.RequestVote, appendC chan utils.AppendEntries, cluster *utils.Cluster) {
	var electionTimer = time.NewTimer(utils.RandomDuration())
	var self = NodeState{0, -1, follower, id, 0}
	var voteReplyC = make(chan utils.RequestVoteReply)

	fmt.Printf("RNode %v became FOLLOWER with term %v\n", self.ID, self.CurrentTerm)
	for {
		select {
		case msg := <-appendC:
			// fmt.Printf("RNode: %v, Ping from %v with term %v\n", self.ID, msg.LeaderID, msg.Term)
			if msg.Term > self.CurrentTerm {
				self.CurrentTerm = msg.Term
				if self.State != follower {
					self.State = follower
					fmt.Printf("RNode %v became FOLLOWER with term %v. Message from leader %v \n", self.ID, self.CurrentTerm, msg.LeaderID)
				}
				electionTimer.Reset(utils.RandomDuration())
				// fmt.Printf("Heartbeat from leader '%v' with term %v\n", msg.LeaderID, msg.Term)
			}
		case msg := <-voteC:
			if msg.Term > self.CurrentTerm {
				self.CurrentTerm = msg.Term
				self.State = follower
				self.VotedFor = msg.CandidateID
				electionTimer.Reset(utils.RandomDuration())
				fmt.Printf("RNode '%v' became FOLLOWER with term %v and voted for RNode %v\n", self.ID, self.CurrentTerm, msg.CandidateID)
				go func(term int) { msg.ReplyChan <- utils.RequestVoteReply{term, true} }(self.CurrentTerm)
			} else {
				go func(term int) { msg.ReplyChan <- utils.RequestVoteReply{term, false} }(self.CurrentTerm)
			}
		case response := <-voteReplyC:
			if response.Term > self.CurrentTerm {
				self.CurrentTerm = response.Term
				self.State = follower
				electionTimer.Reset(utils.RandomDuration())
				fmt.Printf("RNode %v became FOLLOWER with term %v\n", self.ID, self.CurrentTerm)
			} else {
				if response.VoteGranted {
					self.Votes = self.Votes + 1
				}
				if self.Votes >= int(math.Floor(float64(cluster.NodeCount)/2.0)+1.0) {
					self.State = leader
					electionTimer.Stop()
					paceMaker := time.NewTicker(time.Millisecond * 500)
					fmt.Printf("RNode %v became LEADER with term %v\n", self.ID, self.CurrentTerm)
					// boom := 0
					for _ = range paceMaker.C {
						fmt.Printf(".")
						// if boom > -1 {
						cluster.BroadCastAppendEntries(self.ID, utils.AppendEntries{self.CurrentTerm, self.ID})
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
			self.Votes = 1
			self.VotedFor = self.ID
			electionTimer.Reset(utils.RandomDuration())
			fmt.Printf("RNode %v became CANDIDATE with term %v\n", self.ID, self.CurrentTerm)
			cluster.RequestVotes(self.ID, utils.RequestVote{self.CurrentTerm, self.ID, voteReplyC})
		default:
		}
	}
}

func node(id int, cluster *utils.Cluster) utils.NodeChan {
	var requestVoteC = make(chan utils.RequestVote)
	var appendEntriesC = make(chan utils.AppendEntries)
	var nodeChan = utils.NodeChan{id, requestVoteC, appendEntriesC}
	go raft(id, requestVoteC, appendEntriesC, cluster)
	return nodeChan
}

func main() {
	var s string
	var c = 2

	var cluster = utils.MakeCluster(c)
	for idx := 0; idx < c; idx++ {
		cluster.Join(node(idx, cluster))
	}

	fmt.Scanln(&s)
}
