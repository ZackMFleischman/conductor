package core

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"regexp"
)

type Project struct {
	ID        string `json:"project_id"`
	CommonDir string `json:"common_dir"`
	Prefix    string `json:"prefix"`
}

func (s *Service) Project(ctx context.Context, id, common string) (Project, error) {
	var p Project
	var e error
	if id != "" {
		e = s.Store.DB.QueryRowContext(ctx, "SELECT id,common_dir,prefix FROM projects WHERE id=?", id).Scan(&p.ID, &p.CommonDir, &p.Prefix)
	} else {
		e = s.Store.DB.QueryRowContext(ctx, "SELECT id,common_dir,prefix FROM projects WHERE common_dir=?", common).Scan(&p.ID, &p.CommonDir, &p.Prefix)
	}
	if e == sql.ErrNoRows {
		return p, Fail("NOT_FOUND", "project is not registered")
	}
	return p, e
}
func (s *Service) Init(ctx context.Context, g gitctx.Context, prefix, request string) (json.RawMessage, error) {
	if !regexp.MustCompile(`^[A-Z][A-Z0-9]{0,15}$`).MatchString(prefix) {
		return nil, Fail("USAGE", "prefix must be 1-16 uppercase letters or digits, starting with a letter")
	}
	scope := fmt.Sprintf("registration:%x", sha256.Sum256([]byte(g.CommonDir)))
	return s.Mutate(ctx, scope, request, "init", "local-user", map[string]any{"common_dir": g.CommonDir, "prefix": prefix}, func(c *sql.Conn) (json.RawMessage, error) {
		var p Project
		e := c.QueryRowContext(ctx, "SELECT id,common_dir,prefix FROM projects WHERE common_dir=?", g.CommonDir).Scan(&p.ID, &p.CommonDir, &p.Prefix)
		if e == nil {
			if p.Prefix != prefix {
				return nil, Fail("PROJECT_EXISTS", "project already has another prefix")
			}
			return JSON(p), nil
		}
		if e != sql.ErrNoRows {
			return nil, e
		}
		p = Project{UUID(), g.CommonDir, prefix}
		_, e = c.ExecContext(ctx, "INSERT INTO projects(id,common_dir,prefix) VALUES(?,?,?)", p.ID, p.CommonDir, p.Prefix)
		return JSON(p), e
	})
}
