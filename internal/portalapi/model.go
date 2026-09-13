// Package portalapi exposes a read-only presentation boundary for the local portal.
package portalapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/store"
)

var (
	ErrNotFound          = errors.New("project not found")
	ErrInvalidData       = errors.New("invalid board data")
	ErrUnsupportedSchema = fmt.Errorf("portal requires schema %d", store.SchemaVersion)
)

type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Reference struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
}
type Assignee struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Blocker struct {
	Reason    string `json:"reason"`
	TicketKey string `json:"ticketKey,omitempty"`
}
type Ticket struct {
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      string      `json:"status"`
	Assignee    *Assignee   `json:"assignee"`
	Ancestors   []Reference `json:"ancestors"`
	Blockers    []Blocker   `json:"blockers"`
}
type Board struct {
	Project  Project  `json:"project"`
	Tickets  []Ticket `json:"tickets"`
	Revision string   `json:"revision"`
}
type Reader interface {
	Projects(context.Context) ([]Project, error)
	Board(context.Context, string) (Board, error)
}
