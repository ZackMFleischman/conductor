import type { BoardSnapshot, BoardSource } from '../board';

// Fictional demo records, never a snapshot of the shared Conductor registry.
const fixture: BoardSnapshot = {
  project: { id: 'demo-conductor', name: 'Conductor', description: 'A shared view of work, ownership, and what needs attention.' },
  tickets: [
    { id: 'demo-101', key: 'DEMO-101', title: 'Define the next small increment', description: 'Write a focused outcome and clear acceptance criteria so the next piece of work can be picked up with confidence.', status: 'ready', assignee: null, blockers: [] },
    { id: 'demo-102', key: 'DEMO-102', title: 'Make handoffs easy to follow', description: 'Describe what changed, where to find the evidence, and which decisions still need a human review.', status: 'ready', assignee: { id: 'agent-docs', name: 'docs-agent' }, blockers: [] },
    { id: 'demo-103', key: 'DEMO-103', title: 'Build the read-only work board', description: 'Bring ticket descriptions, ownership, and status together in a readable board. Keep sample data clearly identified throughout the experience.', status: 'in_progress', assignee: { id: 'agent-portal', name: 'portal-agent' }, blockers: [] },
    { id: 'demo-104', key: 'DEMO-104', title: 'Clarify the review checklist', description: 'Give reviewers a short set of observable checks for each delivered increment, including the exact source commit and validation evidence.', status: 'in_progress', assignee: { id: 'agent-docs', name: 'docs-agent' }, blockers: [] },
    { id: 'demo-105', key: 'DEMO-105', title: 'Connect the board to live work', description: 'Replace the fixture adapter once the read API and status meanings have been agreed with the core coordinator.', status: 'blocked', assignee: { id: 'agent-portal', name: 'portal-agent' }, blockers: [{ ticketKey: 'DEMO-108', reason: 'Waiting for an agreed read API contract.' }] },
    { id: 'demo-106', key: 'DEMO-106', title: 'Verify a complete handoff', description: 'Exercise the delivered increment from a clean checkout and attach the observations before marking the work accepted.', status: 'blocked', assignee: null, blockers: [{ reason: 'A reviewer and a validation environment are still needed.' }] },
    { id: 'demo-108', key: 'DEMO-108', title: 'Review the board data contract', description: 'Check that the proposed snapshot includes readable descriptions, explicit assignment, and useful blocker reasons without exposing storage details.', status: 'review', assignee: { id: 'agent-core', name: 'core-coordinator' }, blockers: [] },
    { id: 'demo-109', key: 'DEMO-109', title: 'Establish workspace boundaries', description: 'Give each parallel effort a clear ownership boundary and an isolated branch, with shared changes requested through tracked integration work.', status: 'done', assignee: { id: 'agent-core', name: 'core-coordinator' }, blockers: [] },
    { id: 'demo-110', key: 'DEMO-110', title: 'Document local setup', description: 'List the commands needed to start the portal from a clean checkout and describe how to verify the fixture board.', status: 'ready', assignee: null, blockers: [] },
    { id: 'demo-111', key: 'DEMO-111', title: 'Check narrow-screen readability', description: 'Inspect long descriptions and blocker reasons on a small screen. Keep every field readable without horizontal page overflow.', status: 'ready', assignee: { id: 'agent-qa', name: 'qa-agent' }, blockers: [] },
    { id: 'demo-112', key: 'DEMO-112', title: 'Highlight search matches', description: 'Make matching text easy to find in ticket titles, descriptions, assignment and blocker details as the search changes.', status: 'in_progress', assignee: { id: 'agent-qa', name: 'qa-agent' }, blockers: [] },
    { id: 'demo-113', key: 'DEMO-113', title: 'Confirm service access', description: 'Verify that the agreed read boundary returns only the project data the viewer is allowed to inspect.', status: 'blocked', assignee: { id: 'agent-qa', name: 'qa-agent' }, blockers: [{ reason: 'Access rules need coordinator review.' }] },
  ],
};

export const fixtureBoardSource: BoardSource = {
  kind: 'fixture',
  async load(signal) {
    signal?.throwIfAborted();
    return structuredClone(fixture);
  },
};
