package core

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/store"
	"time"
)

type Service struct{ Store *store.Store }
type Fault struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func (e *Fault) Error() string { return e.Message }
func Fail(code, message string) error {
	return &Fault{Code: code, Message: message, Details: map[string]any{}}
}
func UUID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func JSON(v any) json.RawMessage {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func ValidateRequest(id string) error {
	if len(id) == 0 || len(id) > 200 {
		return Fail("USAGE", "--request must contain 1 to 200 bytes")
	}
	return nil
}
func (s *Service) Mutate(ctx context.Context, project, request, operation, actor string, payload any, fn func(*sql.Conn) (json.RawMessage, error)) (json.RawMessage, error) {
	if e := ValidateRequest(request); e != nil {
		return nil, e
	}
	return s.Store.Write(ctx, store.Request{ID: request, Operation: operation, ProjectID: project, ActorID: actor, Payload: JSON(payload)}, fn)
}
