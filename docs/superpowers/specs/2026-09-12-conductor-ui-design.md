# Conductor portal UI design

Design direction · 2026-09-12 · Kanban first, richer document reading later

## Delivery boundary

Build and use the CLI tracer before implementing the portal. The first portal is a thin Kanban view over the same ticket operations. The board is the primary interface; do not build a table-based workbench, full document viewer, or analysis dashboard as prerequisites.

Implement the portal with React, TypeScript, and MUI (Material UI). Use a shared MUI theme, standard accessible controls, and core components such as AppBar, Paper/Card, Chip, Drawer, Dialog, TextField, Button, and Snackbar. Style the Kanban columns with layout primitives; no paid grid component is required. The Go server exposes the existing core operations through HTTP and serves the compiled frontend assets. React does not own a separate ticket rules engine. Keep the initial state/data-fetching layer small and add dependencies only when needed.

The interactive board sample accompanying this specification is a design study using in-memory sample data. It is not the application or an agent integration.

The sample depicts the layout and flows; it is not the production React/MUI implementation. Translate it into MUI components when the portal increment begins.

## First portal

Use a compact header with project identity, observed connection status, and a simple ticket search. Below it, show Ready, In progress, Needs QA, and Done columns. Backlog and cancelled work can be accessed through a filter as needed. An epic has a type badge and child references; it is not an automatically claimable execution task.

Cards show key/type, title, a short latest update, intended assignee or active owner, worktree, and last update time. Distinguish intended assignment from an active claim. A claim does not prove the agent process is running: show last contact separately and label stale availability as unknown. A blocked ticket stays in its workflow column and shows an amber blocker reason. Column counts are actual ticket counts, not estimated completion percentages.

Click a card to open a small right-hand detail drawer: title, state, owner/session, description, criteria, latest progress, dependency/child references, and plain source links. Markdown bodies are stored as text at first and displayed as escaped plain text. No parser or document-resolution system is required for this increment.

For Needs QA, show the submitted evidence and test instructions before Accept QA / Request changes. Request changes requires a reason and returns the ticket to Ready; this does not launch or resume an agent. Board actions call the same core transitions as the CLI. Drag-and-drop is optional subsequent polish and cannot bypass ownership, dependency, or QA constraints.

Use neutral surfaces, thin dividers, system fonts, and a restrained teal accent. Status colors always have labels. Support light/dark appearance. Keep cards readable rather than dense with controls. On narrow screens, horizontally scroll columns within the board; selecting a card opens a full-width drawer with a clear close/back action.

## Incremental UI additions

1. **Working board:** cards, columns, source data refresh, simple details, QA actions, errors and disconnected state.
2. **Rendered ticket Markdown:** descriptions, criteria, updates, QA instructions, and an edit/preview flow; structured checklist remains optional and separate from Markdown task-list syntax.
3. **Linked document sidebar:** open an explicitly linked plan/design beside its ticket. Show title, source worktree/path, live/snapshot badge, and read/capture version. Start with safe read-only rendering.
4. **Reading refinements:** relative document links, heading navigation, rendered/source toggle, live-update indicators, full-width mode and resize behavior.
5. **Operational refinements:** problem and activity views, saved retrospective links/coverage, richer filters and optional table view, driven by usage.

The later sidebar replaces or compresses the board as needed so ticket and document remain readable together. Closing the document restores the previous board position and ticket selection. At narrow widths, show one reading pane at a time. Long paths and code scroll or wrap within their pane.

Missing documents, deleted worktrees, denied paths, unsafe/unsupported previews, and uncaptured snapshot assets must have explicit states. Never substitute another branch's document. Markdown rendering and source-file access follow the main design's sanitization and containment rules. Browser access over SSH uses the same routes and never assumes files exist on the Mac.

## Workflow problem view, when added

Original observation first; expected/actual behavior, appended correction, then retrospective interpretation and improvement ticket. Distinguish recovered from reviewed, delivered from verified effective. The view presents saved reports; it does not pretend to launch a reviewer agent. Reporting is already usable through the CLI before this page exists.

## Delivery and PR view, when added

Keep ticket Kanban states unchanged. Details can link to one or more deliveries, with included scope and sibling tickets. A delivery groups changes landing together; it shows direct landing or a linked PR, independently observed review/merge status, last checked time, and integration result with the exact tested commit. Ticket QA accepted, PR approved, merged, and integrated successfully have separate labels. PR-free projects show no missing-PR warning.

Start with source links and notes in the first board. Add a compact delivery detail view only after structured delivery tracking ships; no second Kanban or mandatory provider connection. If the target commit differs from the tested commit, show that the current target is unverified without erasing the historical result.

## Interaction requirements

- Keyboard-reachable controls, visible focus, labeled icon actions, Escape to close the drawer/dialog, and focus restoration.
- Empty columns and searches use concise explanatory text. Network loss leaves the last data readable with an explicit stale indicator.
- Mutations show pending, success, or failure. A stale revision preserves user input and reports a conflict.
- Cards and status labels work in both themes without depending on color alone.
- Validate live board refresh, QA transitions, and independent CLI operation before adding richer views.
- Verify Windows browser use and Mac access to the Linux portal over an actual SSH forward.

The initial mockup explores board density, card selection, plain details, and a sample QA transition. These are demonstration interactions only; no changes persist after reload.
