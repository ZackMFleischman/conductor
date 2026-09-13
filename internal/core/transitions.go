package core

import (
	"context"
	"database/sql"
)

// CheckTicketClaimTx runs optional restrictions on the acquisition connection.
// Ownership, live-session, assignment and revision checks remain in ticketMutation.
func CheckTicketClaimTx(ctx context.Context, c *sql.Conn, projectID, ticketID, sessionID string) error {
	if e := CheckTicketDependenciesTx(ctx, c, projectID, ticketID); e != nil {
		return e
	}
	if e := CheckWorkflowEligibilityTx(ctx, c, projectID, ticketID); e != nil {
		return e
	}
	return CheckTeamClaimTx(ctx, c, projectID, ticketID, sessionID)
}
