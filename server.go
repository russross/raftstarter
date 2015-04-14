package main

import "errors"

func (s *Server) Do(request *ClientRequest, response *ClientResponse) error {
	return errors.New("Not yet implemented")
}

func (s *Server) AppendEntries(request *AppendEntriesRequest, response *AppendEntriesResponse) error {
	return errors.New("Not yet implemented")
}

func (s *Server) RequestVote(request *RequestVoteRequest, response *RequestVoteResponse) error {
	return errors.New("Not yet implemented")
}
