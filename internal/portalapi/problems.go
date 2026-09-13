package portalapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
)

type ProblemSummary struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Summary   string `json:"summary"`
	TicketKey string `json:"ticketKey,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	NoteCount int    `json:"noteCount"`
}
type ProblemNote struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	Reporter  string `json:"reporter"`
}
type ProblemDetail struct {
	ProblemSummary
	Expected   string        `json:"expected"`
	Actual     string        `json:"actual"`
	Correction string        `json:"correction"`
	Evidence   string        `json:"evidence"`
	Reporter   string        `json:"reporter"`
	Notes      []ProblemNote `json:"notes"`
}
type ProblemReader interface {
	Problem(context.Context, string, string) (ProblemDetail, error)
}

func readProblems(ctx context.Context, tx *sql.Tx, project string) ([]ProblemSummary, error) {
	rows, err := tx.QueryContext(ctx, `SELECT p.id,p.display_key,p.summary,COALESCE(t.display_key,''),p.created_at,COALESCE(MAX(e.created_at),p.created_at),COUNT(e.id)
 FROM problems p LEFT JOIN tickets t ON t.id=p.ticket_id AND t.project_id=p.project_id
 LEFT JOIN events e ON e.problem_id=p.id AND e.project_id=p.project_id AND e.kind='problem.append'
 WHERE p.project_id=? GROUP BY p.id ORDER BY p.created_at DESC,p.rowid DESC`, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ProblemSummary, 0)
	for rows.Next() {
		var p ProblemSummary
		if err = rows.Scan(&p.ID, &p.Key, &p.Summary, &p.TicketKey, &p.CreatedAt, &p.UpdatedAt, &p.NoteCount); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r DBReader) Problem(ctx context.Context, project, id string) (ProblemDetail, error) {
	tx, err := r.beginSnapshot(ctx)
	if err != nil {
		return ProblemDetail{}, err
	}
	defer tx.Rollback()
	summaries, err := readProblems(ctx, tx, project)
	if err != nil {
		return ProblemDetail{}, err
	}
	detail := ProblemDetail{Notes: make([]ProblemNote, 0)}
	for _, p := range summaries {
		if p.ID == id {
			detail.ProblemSummary = p
			break
		}
	}
	if detail.ID == "" {
		return ProblemDetail{}, ErrNotFound
	}
	err = tx.QueryRowContext(ctx, `SELECT p.expected,p.actual,p.correction,p.evidence,COALESCE(a.name,'Unknown agent') FROM problems p LEFT JOIN sessions s ON s.id=p.session_id AND s.project_id=p.project_id LEFT JOIN agents a ON a.id=s.agent_id AND a.project_id=p.project_id WHERE p.project_id=? AND p.id=?`, project, id).Scan(&detail.Expected, &detail.Actual, &detail.Correction, &detail.Evidence, &detail.Reporter)
	if err != nil {
		return ProblemDetail{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT e.id,e.body,e.created_at,COALESCE(a.name,'Unknown agent') FROM events e LEFT JOIN sessions s ON s.id=e.actor_id AND s.project_id=e.project_id LEFT JOIN agents a ON a.id=s.agent_id AND a.project_id=e.project_id WHERE e.project_id=? AND e.problem_id=? AND e.kind='problem.append' ORDER BY e.seq`, project, id)
	if err != nil {
		return ProblemDetail{}, err
	}
	for rows.Next() {
		var n ProblemNote
		if err = rows.Scan(&n.ID, &n.Body, &n.CreatedAt, &n.Reporter); err != nil {
			rows.Close()
			return ProblemDetail{}, err
		}
		detail.Notes = append(detail.Notes, n)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return ProblemDetail{}, err
	}
	if err = tx.Commit(); err != nil {
		return ProblemDetail{}, err
	}
	return detail, nil
}
func (s *server) problem(w http.ResponseWriter, r *http.Request) {
	reader, ok := s.reader.(ProblemReader)
	if !ok {
		apiError(w, 404, "NOT_FOUND", "Problem report not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.options.ReadTimeout)
	defer cancel()
	p, err := reader.Problem(ctx, r.PathValue("project"), r.PathValue("problem"))
	if errors.Is(err, ErrNotFound) {
		apiError(w, 404, "NOT_FOUND", "Problem report not found")
		return
	}
	if err != nil {
		readError(w, err)
		return
	}
	apiJSON(w, 200, p)
}
