package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type RetrospectiveBeginInput struct {
	SessionID string `json:"session_id"`
	Limit     int    `json:"limit"`
}

type RetrospectiveListInput struct {
	Limit  int
	Cursor string
	Due    bool
}
type RetrospectiveDecisionSummary struct {
	Seq            int64   `json:"seq"`
	ID             string  `json:"id"`
	GroupKey       string  `json:"group_key"`
	Action         string  `json:"action"`
	ReviewAfter    string  `json:"review_after"`
	RevisitTrigger string  `json:"revisit_trigger"`
	TicketID       *string `json:"ticket_id"`
	RecurrenceOf   *string `json:"recurrence_of"`
}
type RetrospectiveDecisionPage struct {
	Items      []RetrospectiveDecisionSummary `json:"items"`
	NextCursor string                         `json:"next_cursor,omitempty"`
}

func (s *Service) ListRetrospectiveDecisions(ctx context.Context, project string, in RetrospectiveListInput) (RetrospectiveDecisionPage, error) {
	p := RetrospectiveDecisionPage{Items: []RetrospectiveDecisionSummary{}}
	if in.Limit == 0 {
		in.Limit = 50
	}
	if in.Limit < 1 || in.Limit > 100 {
		return p, Fail("INVALID_INPUT", "limit must be between 1 and 100")
	}
	var cursor int64
	var err error
	if in.Cursor != "" {
		cursor, err = strconv.ParseInt(in.Cursor, 10, 64)
		if err != nil || cursor < 0 {
			return p, Fail("INVALID_INPUT", "invalid cursor")
		}
	}
	query := `SELECT rowid,id,group_key,action,review_after,revisit_trigger,ticket_id,recurrence_of FROM retrospective_decisions WHERE project_id=? AND rowid>?`
	args := []any{project, cursor}
	if in.Due {
		query += " AND julianday(review_after)<=julianday(?) AND id IN (SELECT last_decision_id FROM retrospective_groups WHERE project_id=?)"
		args = append(args, Now())
		args = append(args, project)
	}
	query += " ORDER BY rowid LIMIT ?"
	args = append(args, in.Limit+1)
	rows, err := s.Store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var v RetrospectiveDecisionSummary
		if err = rows.Scan(&v.Seq, &v.ID, &v.GroupKey, &v.Action, &v.ReviewAfter, &v.RevisitTrigger, &v.TicketID, &v.RecurrenceOf); err != nil {
			return p, err
		}
		p.Items = append(p.Items, v)
	}
	if err = rows.Err(); err != nil {
		return p, err
	}
	if len(p.Items) > in.Limit {
		p.Items = p.Items[:in.Limit]
		p.NextCursor = strconv.FormatInt(p.Items[len(p.Items)-1].Seq, 10)
	}
	return p, nil
}

type RetrospectiveBatchInput struct {
	SessionID string `json:"session_id"`
	BatchID   string `json:"batch_id"`
}
type RetrospectiveChange struct {
	Seq        int64           `json:"seq"`
	ProblemID  string          `json:"problem_id"`
	Kind       string          `json:"kind"`
	Body       string          `json:"body"`
	Payload    json.RawMessage `json:"payload"`
	Covered    bool            `json:"covered"`
	DecisionID *string         `json:"decision_id"`
}
type RetrospectiveBatch struct {
	ID         string                `json:"id"`
	FromSeq    int64                 `json:"from_seq"`
	ThroughSeq int64                 `json:"through_seq"`
	Status     string                `json:"status"`
	Changes    []RetrospectiveChange `json:"changes"`
}
type RetrospectiveDecisionInput struct {
	SessionID      string  `json:"session_id"`
	BatchID        string  `json:"batch_id"`
	GroupKey       string  `json:"group_key"`
	Action         string  `json:"action"`
	Observation    string  `json:"observation"`
	InferredCause  string  `json:"inferred_cause"`
	Rationale      string  `json:"rationale"`
	RevisitTrigger string  `json:"revisit_trigger"`
	ReviewAfter    string  `json:"review_after"`
	EventSeqs      []int64 `json:"event_seqs"`
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	TicketID       string  `json:"ticket_id"`
	MilestoneID    string  `json:"milestone_id"`
	Condition      string  `json:"condition"`
}
type RetrospectiveDecision struct {
	ID string `json:"id"`
	RetrospectiveDecisionInput
	RecurrenceOf *string `json:"recurrence_of"`
	CreatedAt    string  `json:"created_at"`
}

type retrospectiveReader interface {
	ticketReader
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func retrospectiveBatch(ctx context.Context, c retrospectiveReader, project, id string) (RetrospectiveBatch, error) {
	b := RetrospectiveBatch{Changes: []RetrospectiveChange{}}
	err := c.QueryRowContext(ctx, "SELECT id,from_seq,through_seq,status FROM retrospective_batches WHERE project_id=? AND id=?", project, id).Scan(&b.ID, &b.FromSeq, &b.ThroughSeq, &b.Status)
	if err == sql.ErrNoRows {
		return b, Fail("NOT_FOUND", "retrospective batch not found")
	}
	if err != nil {
		return b, err
	}
	rows, err := c.QueryContext(ctx, `SELECT e.seq,e.problem_id,e.kind,e.body,e.payload,EXISTS(SELECT 1 FROM retrospective_citations rc WHERE rc.project_id=e.project_id AND rc.event_seq=e.seq),(SELECT decision_id FROM retrospective_citations rc WHERE rc.project_id=e.project_id AND rc.event_seq=e.seq) FROM events e WHERE e.project_id=? AND e.seq>? AND e.seq<=? AND e.kind IN('problem.add','problem.append') ORDER BY e.seq`, project, b.FromSeq, b.ThroughSeq)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var v RetrospectiveChange
		var payload string
		if err = rows.Scan(&v.Seq, &v.ProblemID, &v.Kind, &v.Body, &payload, &v.Covered, &v.DecisionID); err != nil {
			return b, err
		}
		v.Payload = json.RawMessage(payload)
		b.Changes = append(b.Changes, v)
	}
	return b, rows.Err()
}

type RetrospectiveEffectivenessInput struct {
	SessionID  string `json:"session_id"`
	DecisionID string `json:"decision_id"`
	Outcome    string `json:"outcome"`
	Evidence   string `json:"evidence"`
}
type RetrospectiveEffectiveness struct {
	ID string `json:"id"`
	RetrospectiveEffectivenessInput
	CreatedAt string `json:"created_at"`
}
type RetrospectiveDecisionDetail struct {
	RetrospectiveDecision
	Effectiveness          []RetrospectiveEffectiveness `json:"effectiveness"`
	EffectivenessTruncated bool                         `json:"effectiveness_truncated"`
}

func (s *Service) AppendRetrospectiveEffectiveness(ctx context.Context, project, request string, in RetrospectiveEffectivenessInput) (json.RawMessage, error) {
	if in.Outcome != "improved" && in.Outcome != "unchanged" && in.Outcome != "worse" && in.Outcome != "unknown" {
		return nil, Fail("INVALID_INPUT", "outcome must be improved, unchanged, worse, or unknown")
	}
	if err := validateProblemText("evidence", in.Evidence, true, 65536); err != nil {
		return nil, err
	}
	return s.Mutate(ctx, project, request, "retrospective.effectiveness", in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		if err := validateProblemSession(ctx, c, project, in.SessionID); err != nil {
			return nil, err
		}
		var id string
		err := c.QueryRowContext(ctx, "SELECT id FROM retrospective_decisions WHERE project_id=? AND id=?", project, in.DecisionID).Scan(&id)
		if err == sql.ErrNoRows {
			return nil, Fail("NOT_FOUND", "decision not found")
		}
		if err != nil {
			return nil, err
		}
		v := RetrospectiveEffectiveness{ID: UUID(), RetrospectiveEffectivenessInput: in, CreatedAt: Now()}
		if _, err = c.ExecContext(ctx, "INSERT INTO retrospective_effectiveness(id,project_id,decision_id,session_id,outcome,evidence,created_at) VALUES(?,?,?,?,?,?,?)", v.ID, project, in.DecisionID, in.SessionID, in.Outcome, in.Evidence, v.CreatedAt); err != nil {
			return nil, err
		}
		if err = retrospectiveEvent(ctx, c, project, in.SessionID, "retrospective.effectiveness", v); err != nil {
			return nil, err
		}
		return JSON(v), nil
	})
}
func (s *Service) GetRetrospectiveDecision(ctx context.Context, project, id string) (RetrospectiveDecisionDetail, error) {
	d := RetrospectiveDecisionDetail{Effectiveness: []RetrospectiveEffectiveness{}}
	var ticket *string
	err := s.Store.DB.QueryRowContext(ctx, `SELECT id,batch_id,group_key,session_id,action,observation,inferred_cause,rationale,revisit_trigger,review_after,ticket_id,recurrence_of,created_at,milestone_id,condition_text FROM retrospective_decisions WHERE project_id=? AND id=?`, project, id).Scan(&d.ID, &d.BatchID, &d.GroupKey, &d.SessionID, &d.Action, &d.Observation, &d.InferredCause, &d.Rationale, &d.RevisitTrigger, &d.ReviewAfter, &ticket, &d.RecurrenceOf, &d.CreatedAt, &d.MilestoneID, &d.Condition)
	if err == sql.ErrNoRows {
		return d, Fail("NOT_FOUND", "decision not found")
	}
	if err != nil {
		return d, err
	}
	if ticket != nil {
		d.TicketID = *ticket
	}
	rows, err := s.Store.DB.QueryContext(ctx, "SELECT event_seq FROM retrospective_citations WHERE project_id=? AND decision_id=? ORDER BY event_seq", project, id)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var seq int64
		if err = rows.Scan(&seq); err != nil {
			rows.Close()
			return d, err
		}
		d.EventSeqs = append(d.EventSeqs, seq)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return d, err
	}
	rows, err = s.Store.DB.QueryContext(ctx, "SELECT id,session_id,outcome,evidence,created_at FROM retrospective_effectiveness WHERE project_id=? AND decision_id=? ORDER BY rowid DESC LIMIT 101", project, id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		v := RetrospectiveEffectiveness{RetrospectiveEffectivenessInput: RetrospectiveEffectivenessInput{DecisionID: id}}
		if err = rows.Scan(&v.ID, &v.SessionID, &v.Outcome, &v.Evidence, &v.CreatedAt); err != nil {
			return d, err
		}
		d.Effectiveness = append(d.Effectiveness, v)
	}
	if len(d.Effectiveness) > 100 {
		d.EffectivenessTruncated = true
		d.Effectiveness = d.Effectiveness[:100]
	}
	return d, rows.Err()
}

type RetrospectiveStatus struct {
	Watermark       int64   `json:"watermark"`
	OpenBatchID     *string `json:"open_batch_id"`
	EpicTicketID    *string `json:"epic_ticket_id"`
	PendingChanges  int     `json:"pending_changes"`
	DeferredTickets int     `json:"deferred_tickets"`
	DueDecisions    int     `json:"due_decisions"`
}

func (s *Service) RetrospectiveStatus(ctx context.Context, project string) (RetrospectiveStatus, error) {
	var v RetrospectiveStatus
	err := s.Store.DB.QueryRowContext(ctx, `SELECT COALESCE((SELECT watermark FROM retrospective_checkpoints WHERE project_id=?),0),(SELECT id FROM retrospective_batches WHERE project_id=? AND status='open'),(SELECT ticket_id FROM retrospective_epics WHERE project_id=?), (SELECT count(*) FROM retrospective_deferrals WHERE project_id=? AND released_at IS NULL),(SELECT count(*) FROM retrospective_decisions WHERE project_id=? AND julianday(review_after)<=julianday(?) AND id IN (SELECT last_decision_id FROM retrospective_groups WHERE project_id=?))`, project, project, project, project, project, Now(), project).Scan(&v.Watermark, &v.OpenBatchID, &v.EpicTicketID, &v.DeferredTickets, &v.DueDecisions)
	if err != nil {
		return v, err
	}
	err = s.Store.DB.QueryRowContext(ctx, "SELECT count(*) FROM events WHERE project_id=? AND seq>? AND kind IN('problem.add','problem.append')", project, v.Watermark).Scan(&v.PendingChanges)
	return v, err
}
func retrospectiveEvent(ctx context.Context, c *sql.Conn, project, session, kind string, payload any) error {
	_, err := c.ExecContext(ctx, "INSERT INTO events(id,project_id,actor_id,kind,payload,created_at) VALUES(?,?,?,?,?,?)", UUID(), project, session, kind, string(JSON(payload)), Now())
	if err != nil {
		return err
	}
	_, err = c.ExecContext(ctx, "UPDATE sessions SET last_seen_at=? WHERE project_id=? AND id=?", Now(), project, session)
	return err
}
func (s *Service) BeginRetrospective(ctx context.Context, project, request string, in RetrospectiveBeginInput) (json.RawMessage, error) {
	if in.Limit == 0 {
		in.Limit = 50
	}
	if in.Limit < 1 || in.Limit > 100 {
		return nil, Fail("INVALID_INPUT", "limit must be between 1 and 100")
	}
	return s.Mutate(ctx, project, request, "retrospective.begin", in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		if err := validateProblemSession(ctx, c, project, in.SessionID); err != nil {
			return nil, err
		}
		var id string
		err := c.QueryRowContext(ctx, "SELECT id FROM retrospective_batches WHERE project_id=? AND status='open'", project).Scan(&id)
		if err == nil {
			b, e := retrospectiveBatch(ctx, c, project, id)
			return JSON(b), e
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		if _, err = c.ExecContext(ctx, "INSERT INTO retrospective_checkpoints(project_id) VALUES(?) ON CONFLICT DO NOTHING", project); err != nil {
			return nil, err
		}
		var from, through int64
		if err = c.QueryRowContext(ctx, "SELECT watermark FROM retrospective_checkpoints WHERE project_id=?", project).Scan(&from); err != nil {
			return nil, err
		}
		if err = c.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),?) FROM (SELECT seq FROM events WHERE project_id=? AND seq>? AND kind IN('problem.add','problem.append') ORDER BY seq LIMIT ?)`, from, project, from, in.Limit).Scan(&through); err != nil {
			return nil, err
		}
		id = UUID()
		if _, err = c.ExecContext(ctx, "INSERT INTO retrospective_batches(id,project_id,session_id,from_seq,through_seq,status,created_at) VALUES(?,?,?,?,?,'open',?)", id, project, in.SessionID, from, through, Now()); err != nil {
			return nil, err
		}
		b, err := retrospectiveBatch(ctx, c, project, id)
		if err != nil {
			return nil, err
		}
		if err = retrospectiveEvent(ctx, c, project, in.SessionID, "retrospective.begin", b); err != nil {
			return nil, err
		}
		return JSON(b), nil
	})
}
func (s *Service) GetRetrospectiveBatch(ctx context.Context, project, id string) (RetrospectiveBatch, error) {
	return retrospectiveBatch(ctx, s.Store.DB, project, id)
}
func (s *Service) CommitRetrospective(ctx context.Context, project, request string, in RetrospectiveBatchInput) (json.RawMessage, error) {
	return s.Mutate(ctx, project, request, "retrospective.commit", in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		if err := validateProblemSession(ctx, c, project, in.SessionID); err != nil {
			return nil, err
		}
		b, err := retrospectiveBatch(ctx, c, project, in.BatchID)
		if err != nil {
			return nil, err
		}
		if b.Status == "committed" {
			return JSON(b), nil
		}
		for _, v := range b.Changes {
			if !v.Covered {
				return nil, Fail("INCOMPLETE_COVERAGE", "every event must have a cited decision before advancing coverage")
			}
		}
		result, err := c.ExecContext(ctx, "UPDATE retrospective_checkpoints SET watermark=? WHERE project_id=? AND watermark=?", b.ThroughSeq, project, b.FromSeq)
		if err != nil {
			return nil, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n != 1 {
			return nil, Fail("COVERAGE_CONFLICT", "coverage changed; resume the persisted batch")
		}
		if _, err = c.ExecContext(ctx, "UPDATE retrospective_batches SET status='committed' WHERE project_id=? AND id=?", project, b.ID); err != nil {
			return nil, err
		}
		b.Status = "committed"
		if err = retrospectiveEvent(ctx, c, project, in.SessionID, "retrospective.commit", b); err != nil {
			return nil, err
		}
		return JSON(b), nil
	})
}
func validateRetrospectiveDecision(in RetrospectiveDecisionInput) error {
	if in.Action != "now" && in.Action != "milestone" && in.Action != "observe" {
		return Fail("INVALID_INPUT", "action must be now, milestone, or observe")
	}
	for _, v := range []struct {
		name, value string
		limit       int
	}{{"batch_id", in.BatchID, 200}, {"group_key", in.GroupKey, 200}, {"observation", in.Observation, 65536}, {"rationale", in.Rationale, 65536}, {"revisit_trigger", in.RevisitTrigger, 4096}} {
		if err := validateProblemText(v.name, v.value, true, v.limit); err != nil {
			return err
		}
	}
	if _, err := time.Parse(time.RFC3339, in.ReviewAfter); err != nil {
		return Fail("INVALID_INPUT", "review_after requires an RFC3339 deadline")
	}
	if len(in.EventSeqs) < 1 || len(in.EventSeqs) > 100 {
		return Fail("INVALID_INPUT", "cite 1 to 100 event sequences")
	}
	seen := map[int64]bool{}
	for _, seq := range in.EventSeqs {
		if seq <= 0 || seen[seq] {
			return Fail("INVALID_INPUT", "event sequences must be unique and positive")
		}
		seen[seq] = true
	}
	if len(JSON(in)) > 256*1024 {
		return Fail("INVALID_INPUT", "decision exceeds 256 KiB")
	}
	if in.Action == "observe" && (in.TicketID != "" || in.Title != "" || in.Body != "" || in.MilestoneID != "" || in.Condition != "") {
		return Fail("INVALID_INPUT", "observe cannot create or defer a ticket")
	}
	if in.Action == "milestone" && (strings.TrimSpace(in.MilestoneID) == "" || strings.TrimSpace(in.Condition) == "") {
		return Fail("INVALID_INPUT", "milestone requires milestone_id and condition")
	}
	if in.Action == "now" && (in.MilestoneID != "" || in.Condition != "") {
		return Fail("INVALID_INPUT", "now cannot specify milestone fields")
	}
	for _, v := range []struct {
		name, value string
		limit       int
	}{{"inferred_cause", in.InferredCause, 65536}, {"milestone_id", in.MilestoneID, 200}, {"condition", in.Condition, 4096}} {
		if err := validateProblemText(v.name, v.value, false, v.limit); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) DecideRetrospective(ctx context.Context, project, request string, in RetrospectiveDecisionInput) (json.RawMessage, error) {
	if err := validateRetrospectiveDecision(in); err != nil {
		return nil, err
	}
	return s.Mutate(ctx, project, request, "retrospective.decide", in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		if err := validateProblemSession(ctx, c, project, in.SessionID); err != nil {
			return nil, err
		}
		b, err := retrospectiveBatch(ctx, c, project, in.BatchID)
		if err != nil {
			return nil, err
		}
		if b.Status != "open" {
			return nil, Fail("INVALID_STATE", "batch is already committed")
		}
		changes := map[int64]RetrospectiveChange{}
		for _, v := range b.Changes {
			changes[v.Seq] = v
		}
		for _, seq := range in.EventSeqs {
			v, ok := changes[seq]
			if !ok {
				return nil, Fail("INVALID_INPUT", "citation is outside this project batch")
			}
			if v.Covered {
				return nil, Fail("COVERAGE_CONFLICT", "event already has a decision; inspect persisted coverage")
			}
		}
		d := RetrospectiveDecision{ID: UUID(), RetrospectiveDecisionInput: in, CreatedAt: Now()}
		var previousTicket, previousDecision *string
		err = c.QueryRowContext(ctx, "SELECT ticket_id,last_decision_id FROM retrospective_groups WHERE project_id=? AND group_key=?", project, in.GroupKey).Scan(&previousTicket, &previousDecision)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		var linked *string
		if previousTicket != nil {
			v, e := readTicket(ctx, c, project, *previousTicket)
			if e != nil {
				return nil, e
			}
			if v.State == "done" {
				d.RecurrenceOf = previousDecision
			} else {
				linked = previousTicket
			}
		}
		if in.Action != "observe" {
			linked, err = retrospectiveRemedy(ctx, c, project, in, linked)
			if err != nil {
				return nil, err
			}
			d.TicketID = *linked
			if in.Action == "milestone" {
				v, e := readTicket(ctx, c, project, *linked)
				if e != nil {
					return nil, e
				}
				if v.Active || (v.Workflow != nil && v.State != "draft") || (v.Workflow == nil && v.State != "ready") {
					return nil, Fail("INVALID_STATE", "only unclaimed draft or ordinary ready remedies can be deferred")
				}
				var milestone string
				e = c.QueryRowContext(ctx, "SELECT milestone_id FROM retrospective_deferrals WHERE project_id=? AND ticket_id=? AND released_at IS NULL", project, *linked).Scan(&milestone)
				if e != nil && e != sql.ErrNoRows {
					return nil, e
				}
				if e == nil && milestone != in.MilestoneID {
					return nil, Fail("DEFERRED", "existing remedy has another milestone; reconcile it explicitly")
				}
				if _, err = c.ExecContext(ctx, `INSERT INTO retrospective_deferrals(project_id,ticket_id,milestone_id,condition_text,review_after) VALUES(?,?,?,?,?) ON CONFLICT(project_id,ticket_id) DO UPDATE SET milestone_id=excluded.milestone_id,condition_text=excluded.condition_text,review_after=excluded.review_after,released_at=NULL WHERE retrospective_deferrals.released_at IS NOT NULL`, project, *linked, in.MilestoneID, in.Condition, in.ReviewAfter); err != nil {
					return nil, err
				}
			}
		} else if linked != nil {
			d.TicketID = *linked
		}
		if _, err = c.ExecContext(ctx, `INSERT INTO retrospective_decisions(id,project_id,batch_id,group_key,session_id,action,observation,inferred_cause,rationale,revisit_trigger,review_after,ticket_id,recurrence_of,created_at,milestone_id,condition_text) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, d.ID, project, in.BatchID, in.GroupKey, in.SessionID, in.Action, in.Observation, in.InferredCause, in.Rationale, in.RevisitTrigger, in.ReviewAfter, linked, d.RecurrenceOf, d.CreatedAt, in.MilestoneID, in.Condition); err != nil {
			return nil, err
		}
		groupTicket := linked
		if groupTicket == nil {
			groupTicket = previousTicket
		}
		if _, err = c.ExecContext(ctx, `INSERT INTO retrospective_groups(project_id,group_key,ticket_id,last_decision_id) VALUES(?,?,?,?) ON CONFLICT(project_id,group_key) DO UPDATE SET ticket_id=excluded.ticket_id,last_decision_id=excluded.last_decision_id`, project, in.GroupKey, groupTicket, d.ID); err != nil {
			return nil, err
		}
		for _, seq := range in.EventSeqs {
			if _, err = c.ExecContext(ctx, "INSERT INTO retrospective_citations(project_id,event_seq,decision_id) VALUES(?,?,?)", project, seq, d.ID); err != nil {
				return nil, err
			}
		}
		if err = retrospectiveEvent(ctx, c, project, in.SessionID, "retrospective.decide", d); err != nil {
			return nil, err
		}
		return JSON(d), nil
	})
}

// The caller holds the Store.Write transaction: semantic grouping, epic creation,
// remedy creation and citations either all commit or all roll back together.
func retrospectiveRemedy(ctx context.Context, c *sql.Conn, project string, in RetrospectiveDecisionInput, linked *string) (*string, error) {
	if linked != nil {
		if in.TicketID != "" {
			v, err := readTicket(ctx, c, project, in.TicketID)
			if err != nil {
				return nil, err
			}
			if v.ID != *linked {
				return nil, Fail("GROUP_CONFLICT", "group already links another active remedy")
			}
		}
		return linked, nil
	}
	var epic string
	err := c.QueryRowContext(ctx, "SELECT ticket_id FROM retrospective_epics WHERE project_id=?", project).Scan(&epic)
	if err == sql.ErrNoRows {
		v, e := CreateWorkflowDraftTx(ctx, c, project, in.SessionID, "Workflow Improvements", "Persistent grouping for evidence-based workflow improvement proposals.", "", "epic")
		if e != nil {
			return nil, e
		}
		epic = v.ID
		if _, err = c.ExecContext(ctx, "INSERT INTO retrospective_epics(project_id,ticket_id) VALUES(?,?)", project, epic); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if in.TicketID != "" {
		v, e := readTicket(ctx, c, project, in.TicketID)
		if e != nil {
			return nil, e
		}
		if v.Active || (v.Workflow != nil && v.State != "draft") || (v.Workflow == nil && v.State != "ready") {
			return nil, Fail("INVALID_STATE", "explicit remedy must be an unclaimed draft or ordinary ready ticket")
		}
		sp, e := ticketMetadata(ctx, c, project, v.ID)
		if e != nil {
			return nil, e
		}
		if sp == nil || sp.ParentID == nil || *sp.ParentID != epic || sp.Kind == "epic" {
			return nil, Fail("INVALID_INPUT", "explicit remedy must belong to the Workflow Improvements epic")
		}
		return &v.ID, nil
	}
	if err := ValidateTicketInput("create", TicketInput{Title: in.Title, Body: in.Body}); err != nil {
		return nil, err
	}
	v, err := CreateWorkflowDraftTx(ctx, c, project, in.SessionID, in.Title, in.Body, epic, "implementation")
	if err != nil {
		return nil, err
	}
	return &v.ID, nil
}

type RetrospectiveMilestoneInput struct {
	SessionID      string `json:"session_id"`
	CoordinationID string `json:"coordination_id"`
	MilestoneID    string `json:"milestone_id"`
	Evidence       string `json:"evidence"`
	Scope          string `json:"scope,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

func (s *Service) ReleaseRetrospectiveMilestone(ctx context.Context, project, request string, in RetrospectiveMilestoneInput) (json.RawMessage, error) {
	if err := validateProblemText("milestone_id", in.MilestoneID, true, 200); err != nil {
		return nil, err
	}
	if err := validateProblemText("evidence", in.Evidence, true, 65536); err != nil {
		return nil, err
	}
	return s.Mutate(ctx, project, request, "retrospective.release-milestone", in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		managed, err := requireOptionalCoordinatorTx(ctx, c, project, in.SessionID, in.CoordinationID)
		if err != nil {
			return nil, err
		}
		if !managed {
			if err = validateProblemText("scope", in.Scope, true, 65536); err != nil {
				return nil, err
			}
			if err = validateProblemText("reason", in.Reason, true, 65536); err != nil {
				return nil, err
			}
		}
		var count int
		err = c.QueryRowContext(ctx, `SELECT count(*) FROM retrospective_deferrals d JOIN claims cl ON cl.project_id=d.project_id AND cl.ticket_id=d.ticket_id WHERE d.project_id=? AND d.milestone_id=? AND d.released_at IS NULL AND cl.released_at IS NULL`, project, in.MilestoneID).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, Fail("ACTIVE_CLAIM", "checkpoint active remedy claims before milestone release")
		}
		// Releasing a scheduling gate invalidates preparation and execution approval;
		// passage never makes the proposed remedy executable by itself.
		if _, err = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET prepared_revision=0,authorized_revision=0 WHERE project_id=? AND ticket_id IN(SELECT ticket_id FROM retrospective_deferrals WHERE project_id=? AND milestone_id=? AND released_at IS NULL)`, project, project, in.MilestoneID); err != nil {
			return nil, err
		}
		if _, err = c.ExecContext(ctx, `UPDATE tickets SET state=CASE WHEN EXISTS(SELECT 1 FROM workflow_ticket_specs s WHERE s.ticket_id=tickets.id) THEN 'draft' ELSE 'ready' END,revision=revision+1,updated_at=? WHERE project_id=? AND id IN(SELECT ticket_id FROM retrospective_deferrals WHERE project_id=? AND milestone_id=? AND released_at IS NULL)`, Now(), project, project, in.MilestoneID); err != nil {
			return nil, err
		}
		result, err := c.ExecContext(ctx, "UPDATE retrospective_deferrals SET released_at=? WHERE project_id=? AND milestone_id=? AND released_at IS NULL", Now(), project, in.MilestoneID)
		if err != nil {
			return nil, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		v := map[string]any{"milestone_id": in.MilestoneID, "released_tickets": n, "evidence": in.Evidence, "scope": in.Scope, "reason": in.Reason}
		if err = retrospectiveEvent(ctx, c, project, in.SessionID, "retrospective.release-milestone", v); err != nil {
			return nil, err
		}
		return JSON(v), nil
	})
}
