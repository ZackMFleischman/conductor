package portalapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		apiError(w, 503, "STREAM_LIMIT", "Too many live connections")
		return
	}
	id := r.PathValue("project")
	b, err := s.snapshot(r.Context(), id)
	if err != nil {
		readError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	write := func(text string) bool {
		if controller.SetWriteDeadline(time.Now().Add(s.options.WriteTimeout)) != nil {
			return false
		}
		if _, e := fmt.Fprint(w, text); e != nil {
			return false
		}
		if controller.Flush() != nil {
			return false
		}
		// Do not let an idle stream's last write deadline expire between heartbeats.
		return controller.SetWriteDeadline(time.Time{}) == nil
	}
	changed := func(revision string) bool {
		// JSON-encode payloads; IDs are hashes supplied by the trusted Reader, not raw
		// request Last-Event-ID. Always invalidate on connect instead of promising replay.
		payload, _ := json.Marshal(map[string]string{"revision": revision})
		return write(fmt.Sprintf("event: board.changed\nid: %s\ndata: %s\n\n", revision, payload))
	}
	if !changed(b.Revision) {
		return
	}
	last := b.Revision
	poll := time.NewTicker(s.options.PollInterval)
	defer poll.Stop()
	heartbeat := time.NewTicker(s.options.HeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if !write(": keep-alive\n\n") {
				return
			}
		case <-poll.C:
			b, err = s.snapshot(r.Context(), id)
			if err != nil {
				write("event: board.error\ndata: {\"code\":\"STORE_UNAVAILABLE\"}\n\n")
				return
			}
			if b.Revision != last {
				if !changed(b.Revision) {
					return
				}
				last = b.Revision
			}
		}
	}
}
