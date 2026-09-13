package gitctx

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func canonicalPath(path string) (string, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", &os.PathError{Op: "canonical", Path: path, Err: err}
	}
	// Resolve the target through its handle. filepath.EvalSymlinks also
	// enumerates ancestors to normalize casing, which can fail when the caller
	// can traverse those ancestors and access the target but cannot list them.
	// Omitting FILE_FLAG_OPEN_REPARSE_POINT follows symlinks and junctions.
	handle, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", &os.PathError{Op: "canonical", Path: path, Err: err}
	}
	defer windows.CloseHandle(handle)
	buffer := make([]uint16, 260)
	for {
		// Zero requests FILE_NAME_NORMALIZED | VOLUME_NAME_DOS.
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
		if err != nil {
			return "", &os.PathError{Op: "canonical", Path: path, Err: err}
		}
		if n < uint32(len(buffer)) {
			return normalizeWindowsFinalPath(windows.UTF16ToString(buffer[:n])), nil
		}
		// An insufficient-buffer result includes the terminating NUL.
		buffer = make([]uint16, n)
	}
}

func normalizeWindowsFinalPath(path string) string {
	path = strings.ToLower(path)
	if strings.HasPrefix(path, `\\?\unc\`) {
		return `\\` + path[len(`\\?\unc\`):]
	}
	return strings.TrimPrefix(path, `\\?\`)
}
