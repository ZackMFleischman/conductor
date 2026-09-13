package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Request struct {
	ID, Operation, ProjectID, ActorID string
	Payload                           json.RawMessage
}

var ErrRequestConflict = errors.New("REQUEST_CONFLICT")

func (s *Store) Write(ctx context.Context, r Request, mutate func(*sql.Conn) (json.RawMessage, error)) (json.RawMessage, error) {
	if len(r.ID) == 0 || len(r.ID) > 200 || r.Operation == "" || r.ProjectID == "" || r.ActorID == "" {
		return nil, errors.New("invalid request scope")
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(r.Payload))
	d.UseNumber()
	if err := d.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return nil, errors.New("payload must contain exactly one JSON value")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	hash := hex.EncodeToString(sum[:])
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return nil, err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	var op, actor, oldHash, result string
	err = conn.QueryRowContext(ctx, "SELECT operation,actor_id,payload_hash,result FROM requests WHERE project_id=? AND request_id=?", r.ProjectID, r.ID).Scan(&op, &actor, &oldHash, &result)
	if err == nil {
		if op != r.Operation || actor != r.ActorID || (oldHash != hash && !legacyTicketHash(r.Operation, value, oldHash)) {
			return nil, ErrRequestConflict
		}
		return json.RawMessage(result), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	body, err := mutate(conn)
	if err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("mutation returned invalid JSON")
	}
	if _, err = conn.ExecContext(ctx, "INSERT INTO requests(project_id,request_id,operation,actor_id,payload_hash,result) VALUES(?,?,?,?,?,?)", r.ProjectID, r.ID, r.Operation, r.ActorID, hash, string(body)); err != nil {
		return nil, err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return nil, err
	}
	return body, nil
}

// Compatibility is restricted to known historical ticket payload fields and their
// exact empty values. Nonempty evidence and any other changed field still conflict.
func legacyTicketHash(op string, value any, want string) bool {
	switch op {
	case "ticket.create", "ticket.assign", "ticket.claim", "ticket.note", "ticket.submit", "ticket.accept", "ticket.reject", "ticket.release", "ticket.block":
	default:
		return false
	}
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	candidate := make(map[string]any, len(m))
	for k, v := range m {
		candidate[k] = v
	}
	commit, hasCommit := m["Commit"]
	validation, hasValidation := m["Validation"]
	for _, includeCommit := range []bool{false, true} {
		if !hasCommit || commit == "" {
			if includeCommit {
				candidate["Commit"] = ""
			} else {
				delete(candidate, "Commit")
			}
		}
		for _, includeValidation := range []bool{false, true} {
			if !hasValidation || validation == nil {
				if includeValidation {
					candidate["Validation"] = nil
				} else {
					delete(candidate, "Validation")
				}
			}
			b, e := json.Marshal(candidate)
			if e != nil {
				return false
			}
			sum := sha256.Sum256(b)
			if hex.EncodeToString(sum[:]) == want {
				return true
			}
		}
	}

	return false
}
