package portalapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Attachments are live file references, not uploaded or managed copies.
type Attachment struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	MediaType  string `json:"mediaType"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	source     string
}

// Angle brackets or backticks support file names with spaces. Bare absolute
// paths support existing screenshot references without changing ticket bodies.
var attachmentReferences = regexp.MustCompile("(?m)^[ \\t]{0,3}\\[[^\\]\\r\\n]+\\]:[ \\t]*(?:<([^>\\r\\n]+)>|([^\\s]+))|!?\\[[^\\]\\r\\n]*\\]\\(<([^>\\r\\n]+)>\\)|!?\\[[^\\]\\r\\n]*\\]\\(([^\\s)]+)(?:\\s+\"[^\"]*\")?\\)|`([^`\\r\\n]+)`|(?:[A-Za-z]:[/\\\\]|/)[^\\s<>\"`|)]+")

func attachmentSources(ticket Ticket) []string {
	var sources []string
	text := strings.Join([]string{ticket.Description, ticket.Summary, ticket.Evidence, ticket.QA}, "\n")
	for _, match := range attachmentReferences.FindAllStringSubmatch(text, 200) {
		value := match[0]
		for _, group := range match[1:] {
			if group != "" {
				value = group
				break
			}
		}
		if value == match[0] {
			value = strings.TrimRight(value, ".,;:")
		}
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		sources = append(sources, value)
	}
	return sources
}

func attachmentPath(root, source string) (string, error) {
	if root == "" || strings.Contains(source, "://") || strings.HasPrefix(source, "\\\\") || strings.HasPrefix(source, "//") {
		return "", os.ErrNotExist
	}
	path := filepath.FromSlash(source)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return "", os.ErrNotExist
	}
	// Windows alternate data streams and Git internals are never attachments.
	for _, part := range strings.FieldsFunc(relative, func(r rune) bool { return r == '/' || r == '\\' }) {
		if strings.EqualFold(part, ".git") || strings.Contains(part, ":") {
			return "", os.ErrNotExist
		}
	}
	return relative, nil
}

func openAttachment(root, source string) (*os.File, error) {
	relative, err := attachmentPath(root, source)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, relative))
	if err != nil {
		return nil, err
	}
	relative, err = attachmentPath(root, resolved)
	if err != nil {
		return nil, err
	}
	// OpenRoot also enforces containment during resolution (including symlink races).
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	file, err := dir.Open(relative)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, os.ErrNotExist
	}
	return file, nil
}

func localAttachments(root, projectID string, ticket Ticket) []Attachment {
	result := make([]Attachment, 0)
	seen := map[string]bool{}
	if root == "" {
		return result
	}
	for _, source := range attachmentSources(ticket) {
		relative, err := attachmentPath(root, source)
		if err != nil {
			continue
		}
		identity := filepath.ToSlash(relative)
		// File names are case-insensitive on the supported Windows host.
		if filepath.Separator == '\\' {
			identity = strings.ToLower(identity)
		}
		if seen[identity] {
			continue
		}
		seen[identity] = true
		f, err := openAttachment(root, relative)
		if err != nil {
			continue
		}
		info, err := f.Stat()
		f.Close()
		if err != nil {
			continue
		}
		hash := sha256.Sum256([]byte(identity))
		id := hex.EncodeToString(hash[:])
		mediaType := mime.TypeByExtension(strings.ToLower(filepath.Ext(relative)))
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		result = append(result, Attachment{ID: id, Name: filepath.Base(relative), URL: "/api/v1/projects/" + url.PathEscape(projectID) + "/tickets/" + url.PathEscape(ticket.ID) + "/attachments/" + id, MediaType: mediaType, Size: info.Size(), ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z"), source: relative})
		if len(result) == 50 {
			break
		}
	}
	return result
}

func (s *server) attachment(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.options.ReadTimeout)
	defer cancel()
	board, err := s.reader.Board(ctx, r.PathValue("project"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
		} else {
			readError(w, err)
		}
		return
	}
	for _, ticket := range board.Tickets {
		if ticket.ID != r.PathValue("ticket") {
			continue
		}
		for _, a := range ticket.Attachments {
			if a.ID != r.PathValue("attachment") {
				continue
			}
			f, err := openAttachment(board.Project.root, a.source)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil {
				http.NotFound(w, r)
				return
			}
			disposition := "attachment"
			switch a.MediaType {
			case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif", "image/bmp":
				disposition = "inline"
			}
			w.Header().Set("Content-Type", a.MediaType)
			w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": a.Name}))
			w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
			w.Header().Set("Cache-Control", "no-store")
			http.ServeContent(w, r, a.Name, info.ModTime(), f)
			return
		}
	}
	http.NotFound(w, r)
}
