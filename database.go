package main

import (
	"database/sql"
	"fmt"
	"log"
)

var dbSetup = `
CREATE TABLE IF NOT EXISTS state (
	current_term integer not null,
	voted_for text
);

INSERT INTO state
SELECT 0, NULL WHERE NOT EXISTS (SELECT 1 FROM state);

CREATE TABLE IF NOT EXISTS log (
	id integer primary key,
	term integer not null,
	client_id text not null,
	client_serial integer not null,

	operation text not null,
	key text not null,
	value text not null
);
`

func initDB(id string) (*sql.DB, error) {
	filename := fmt.Sprintf("raft-%s.db", id)
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Printf("error opening db %q: %v", filename, err)
		return nil, err
	}
	_, err = db.Exec(dbSetup)
	if err != nil {
		log.Printf("db setup error: %v", err)
		db.Close()
		return nil, err
	}
	return db, nil
}

// getPersistent loads the persistent state of a server from the database.
func (s *Server) getPersistent(tx *sql.Tx) error {
	var currentTerm int
	var votedFor string
	err := tx.QueryRow(`SELECT current_term, voted_for FROM state`).Scan(currentTerm, votedFor)
	if err != nil {
		log.Printf("db error getting state: %v", err)
		return err
	}
	s.CurrentTerm = currentTerm
	s.VotedFor = votedFor
	return nil
}

// putPersistent saves the persistent state of a server to the database.
func (s *Server) putPersistent(tx *sql.Tx) error {
	votedFor := &s.VotedFor
	if len(*votedFor) == 0 {
		votedFor = nil
	}
	_, err := tx.Exec(`UPDATE state SET current_term = ?, voted_for = ?`, s.CurrentTerm, votedFor)
	if err != nil {
		log.Printf("db error putting state: %v", err)
		return err
	}
	return nil
}

// verifyLogAt confirms the existence of a log entry with the given index and term.
func verifyLogAt(tx *sql.Tx, index, term int) (bool, error) {
	found := false
	err := tx.QueryRow(`SELECT 1 FROM log WHERE id = ? AND term = ?`).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		log.Printf("db error checking log entry %d: %v", index, err)
		return false, err
	}
	return true, nil
}

// getLastLogEntry returns the index and term of the last entry in the log.
func getLastLogEntry(tx *sql.Tx) (index, term int, err error) {
	err = tx.QueryRow(`SELECT index, term FROM log ORDER BY index LIMIT 1`).Scan(&index, &term)
	if err == sql.ErrNoRows {
		return -1, -1, nil
	} else if err != nil {
		log.Printf("db error checking last log entry index and term: %v", err)
		return 0, 0, err
	}
	return index, term, nil
}

// getLogEntries retrieves log entries in the range [from, to)
func getLogEntries(tx *sql.Tx, from, to int) ([]*LogEntry, error) {
	var out []*LogEntry
	rows, err := tx.Query(`SELECT id, term, client_id, client_serial, operation, key, value `+
		`FROM log WHERE id >= ? AND id < ? ORDER BY id ASC`, from, to)
	if err != nil {
		log.Printf("db error loading log entries [%d,%d): %v", from, to, err)
		return nil, err
	}

	for rows.Next() {
		l := new(LogEntry)
		out = append(out, l)
		err := rows.Scan(
			&l.ID,
			&l.Term,
			&l.ClientRequest.ClientID,
			&l.ClientRequest.ClientSerial,
			&l.ClientRequest.Operation,
			&l.ClientRequest.Key,
			&l.ClientRequest.Value)
		if err != nil {
			log.Printf("db error scanning log entry: %v", err)
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("db error reading log entries: %v", err)
		return nil, err
	}
	return out, nil
}

// saveLogEntries saves a slice of log entries, which must be in order by index.
func (s *Server) saveLogEntries(tx *sql.Tx, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// truncate the log if applicable
	_, err := tx.Exec(`DELETE FROM log WHERE id >= ?`, entries[0].ID)
	if err != nil {
		log.Printf("db error truncating log: %v", err)
		return err
	}

	for _, elt := range entries {
		_, err := tx.Exec(`INSERT INTO log (id, term, client_id, client_serial, operation, key, value) `+
			`VALUES (?,?,?,?,?,?,?)`,
			elt.ID,
			elt.Term,
			elt.ClientRequest.ClientID,
			elt.ClientRequest.ClientSerial,
			elt.ClientRequest.Operation,
			elt.ClientRequest.Key,
			elt.ClientRequest.Value)
		if err != nil {
			log.Printf("db error inserting log entry: %v", err)
			return err
		}
	}

	return nil
}
