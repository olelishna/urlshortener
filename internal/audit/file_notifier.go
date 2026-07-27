package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileNotifier writes audit events as JSON lines to a file.
type FileNotifier struct {
	Path string
	mu   sync.Mutex
	File *os.File
}

// NewFileNotifier opens the given file path for appending.
func NewFileNotifier(path string) (*FileNotifier, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, err
	}

	return &FileNotifier{Path: path, File: file}, nil
}

// Notify appends a JSON-encoded audit event as a new line to the file.
func (n *FileNotifier) Notify(event Event) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = n.File.Write(append(data, '\n'))

	return err
}

// Close closes the underlying file handle.
func (n *FileNotifier) Close() error {
	if n.File == nil {
		return nil
	}
	err := n.File.Close()
	n.File = nil

	return err
}
