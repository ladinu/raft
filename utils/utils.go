package utils

import (
	"math/rand"
	"time"
)

func randRange(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

// RandomDuration returns a random duration between 150 and 350 milliseconds
func RandomDuration() time.Duration {
	return time.Duration(randRange(3000, 5000)) * time.Millisecond
}

// RequestVote struct
type RequestVote struct {
	Term        int
	CandidateID int
	ReplyChan   chan RequestVoteReply
}

// RequestVoteReply struct
type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

// AppendEntries struct
type AppendEntries struct {
	Term     int
	LeaderID int
}

// NodeChan contains channels used by a node in the cluster
type NodeChan struct {
	NodeID         int
	RequestVoteC   chan RequestVote
	AppendEntriesC chan AppendEntries
}

type clusterVote struct {
	nodeID int
	msg    RequestVote
}

type clusterAppendEntries struct {
	nodeID int
	msg    AppendEntries
}

// Cluster is structure which represent a Mesh network topology
type Cluster struct {
	_clusterVoteC          chan clusterVote
	_clusterAppendEntriesC chan clusterAppendEntries
	NodeCount              int
	NodeChanMap            map[int]NodeChan
}

// MakeCluster return an instance of a cluster
func MakeCluster(nodeCount int) *Cluster {
	var c = &Cluster{make(chan clusterVote), make(chan clusterAppendEntries), nodeCount, make(map[int]NodeChan)}
	// Broadcast messages to the cluster
	go func(cluster *Cluster) {
		for {
			select {
			case m := <-c._clusterVoteC:
				for _, v := range c.NodeChanMap {
					go func(nc NodeChan) { nc.RequestVoteC <- m.msg }(v)
				}
			default:
			}
		}
	}(c)

	go func(cluster *Cluster) {
		for {
			select {
			case m := <-c._clusterAppendEntriesC:
				for _, v := range c.NodeChanMap {
					go func(nc NodeChan) { nc.AppendEntriesC <- m.msg }(v)
				}
			default:
			}
		}
	}(c)
	return c
}

// Join adds a NodeChan to cluster
func (c *Cluster) Join(nc NodeChan) {
	c.NodeChanMap[nc.NodeID] = nc
}

// RequestVotes broadcast ReequestVote messages to cluster
func (c *Cluster) RequestVotes(nodeID int, rv RequestVote) {
	c._clusterVoteC <- clusterVote{nodeID, rv}
}

// BroadCastAppendEntries sends AppendEntries to the cluster
func (c *Cluster) BroadCastAppendEntries(nodeID int, ae AppendEntries) {
	c._clusterAppendEntriesC <- clusterAppendEntries{nodeID, ae}
}
