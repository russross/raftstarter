package main

import "sync"

// Server represents the state of a single server, and is the endpoint for RPC calls.
type Server struct {
	sync.Mutex

	// the current state: one of "follower", "leader", or "candidate"
	State string

	// all servers persistent state
	CurrentTerm int
	VotedFor    string

	// all servers volatile state
	CommitIndex   int
	LastApplied   int
	CurrentLeader string

	// leader-only volatile state
	NextIndex  map[string]int
	MatchIndex map[string]int

	// the state machine
	Data map[string]string

	// cache of most recent completed requests, one per client
	// maps client ID to (non-redirect) response
	MostRecent map[string]*ClientResponse
}

type ClientRequest struct {
	ClientID     string
	ClientSerial int

	// "get", "put", or "delete"
	Operation string
	Key       string
	Value     string
}

type ClientResponse struct {
	// if non-empty, repeat the request at the given address and ignore other fields
	RedirectTo string

	ClientSerial int
	Successful   bool
	Result       string
}

type LogEntry struct {
	ID   int
	Term int

	*ClientRequest
}

type AppendEntriesRequest struct {
	Term         int
	LeaderID     string
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []*LogEntry
	LeaderCommit int
}

type AppendEntriesResponse struct {
	Term    int
	Success bool
}

type RequestVoteRequest struct {
	Term         int
	CandidateID  string
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteResponse struct {
	Term        int
	VoteGranted bool
}
