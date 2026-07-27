package audit_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/olelishna/urlshortener/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockNotifier struct {
	mock.Mock
}

func (m *mockNotifier) Notify(event audit.Event) error {
	args := m.Called(event)
	if args.Get(0) == nil {
		return nil
	}

	return args.Error(0)
}

func TestNewEvent(t *testing.T) {
	e := audit.NewEvent("shorten", "user-123", "https://example.com")

	assert.Equal(t, "shorten", e.Action)
	assert.Equal(t, "user-123", e.UserID)
	assert.Equal(t, "https://example.com", e.URL)
	assert.GreaterOrEqual(t, e.TS, int64(1000000000))
	assert.LessOrEqual(t, e.TS, int64(99999999999))
}

func TestNewEvent_Follow(t *testing.T) {
	e := audit.NewEvent("follow", "user-456", "https://redirected.com")

	assert.Equal(t, "follow", e.Action)
	assert.Equal(t, "user-456", e.UserID)
	assert.Equal(t, "https://redirected.com", e.URL)
}

func TestNewManager(t *testing.T) {
	m := audit.NewManager()
	assert.NotNil(t, m)
	assert.NotNil(t, m.Observers)
	assert.Empty(t, m.Observers)
}

func TestManager_Register(t *testing.T) {
	m := audit.NewManager()
	assert.Empty(t, m.Observers)

	mock1 := new(mockNotifier)
	m.Register(mock1)
	assert.Len(t, m.Observers, 1)

	mock2 := new(mockNotifier)
	m.Register(mock2)
	assert.Len(t, m.Observers, 2)
}

func TestManager_Notify(t *testing.T) {
	m := audit.NewManager()
	mock1 := new(mockNotifier)
	mock2 := new(mockNotifier)

	m.Register(mock1)
	m.Register(mock2)

	event := audit.NewEvent("shorten", "u1", "https://test.com")

	mock1.On("Notify", mock.MatchedBy(func(e audit.Event) bool { return e.Action == "shorten" })).
		Return(nil).
		Once()
	mock2.On("Notify", mock.MatchedBy(func(e audit.Event) bool { return e.Action == "shorten" })).
		Return(nil).
		Once()

	m.Notify(event)

	time.Sleep(50 * time.Millisecond)
	mock1.AssertExpectations(t)
	mock2.AssertExpectations(t)
}

func TestManager_NotifyError(t *testing.T) {
	m := audit.NewManager()
	mock1 := new(mockNotifier)
	m.Register(mock1)

	mock1.On("Notify", mock.MatchedBy(func(e audit.Event) bool { return e.Action == "shorten" })).
		Return(assert.AnError).
		Once()

	m.Notify(audit.NewEvent("shorten", "u1", "https://test.com"))

	time.Sleep(50 * time.Millisecond)
	mock1.AssertExpectations(t)
}

func TestManager_NotifyMultipleEvents(t *testing.T) {
	m := audit.NewManager()
	mock1 := new(mockNotifier)
	m.Register(mock1)

	mock1.On("Notify", mock.MatchedBy(func(e audit.Event) bool { return e.Action == "follow" })).
		Return(nil).
		Times(10)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			m.Notify(audit.NewEvent("follow", "u2", "https://example.com"))
		}

		close(done)
	}()

	<-done
	time.Sleep(100 * time.Millisecond)
	mock1.AssertExpectations(t)
}

func TestFileNotifier_NewFileNotifier(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"

	fn, err := audit.NewFileNotifier(tmpFile)
	require.NoError(t, err)
	require.NotNil(t, fn)
	assert.Equal(t, tmpFile, fn.Path)
	require.NotNil(t, fn.File)

	err = fn.Close()
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

func TestFileNotifier_NewFileNotifier_InvalidPath(t *testing.T) {
	fn, err := audit.NewFileNotifier("/nonexistent/dir/audit.log")
	assert.Error(t, err)
	assert.Nil(t, fn)
}

func TestFileNotifier_Notify(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"

	fn, err := audit.NewFileNotifier(tmpFile)
	require.NoError(t, err)
	defer fn.Close()

	event := audit.NewEvent("shorten", "user-42", "https://example.com/path")
	err = fn.Notify(event)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var decoded audit.Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "shorten", decoded.Action)
	assert.Equal(t, "user-42", decoded.UserID)
	assert.Equal(t, "https://example.com/path", decoded.URL)
	assert.GreaterOrEqual(t, decoded.TS, int64(1000000000))
}

func TestFileNotifier_Notify_AppendMultiple(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"

	fn, err := audit.NewFileNotifier(tmpFile)
	require.NoError(t, err)
	defer fn.Close()

	e1 := audit.NewEvent("shorten", "u1", "https://first.com")
	err = fn.Notify(e1)
	require.NoError(t, err)

	e2 := audit.NewEvent("follow", "u2", "https://second.com")
	err = fn.Notify(e2)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	lineCount := 0

	for _, c := range data {
		if c == '\n' {
			lineCount++
		}
	}

	assert.Equal(t, 2, lineCount, "should have 2 lines")

	lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	require.Len(t, lines, 2)

	var first, second audit.Event
	err = json.Unmarshal(lines[0], &first)
	require.NoError(t, err)
	assert.Equal(t, "shorten", first.Action)
	assert.Equal(t, "u1", first.UserID)

	err = json.Unmarshal(lines[1], &second)
	require.NoError(t, err)
	assert.Equal(t, "follow", second.Action)
	assert.Equal(t, "u2", second.UserID)
}

func TestFileNotifier_Notify_MarshalError(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"

	fn, err := audit.NewFileNotifier(tmpFile)
	require.NoError(t, err)
	defer fn.Close()

	event := audit.NewEvent("shorten", "user-1", "https://test.com")
	err = fn.Notify(event)
	assert.NoError(t, err)
}

func TestFileNotifier_Close(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"

	fn, err := audit.NewFileNotifier(tmpFile)
	require.NoError(t, err)

	err = fn.Close()
	require.NoError(t, err)

	err = fn.Close()
	assert.NoError(t, err)
}

func TestHTTPNotifier_NewHTTPNotifier(t *testing.T) {
	hn := audit.NewHTTPNotifier("http://localhost:9999/audit")
	assert.NotNil(t, hn)
	assert.Equal(t, "http://localhost:9999/audit", hn.Url)
	assert.NotNil(t, hn.Client)
}

func TestHTTPNotifier_Notify_NoServer(t *testing.T) {
	hn := audit.NewHTTPNotifier("http://nonexistent:12345/nope")

	event := audit.NewEvent("shorten", "user-99", "https://error-test.com")
	err := hn.Notify(event)
	assert.Error(t, err)
}

func TestHTTPNotifier_Notify_ConnectedServer(t *testing.T) {
	var receivedEvent audit.Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		err = json.Unmarshal(body, &receivedEvent)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hn := audit.NewHTTPNotifier(server.URL)
	event := audit.NewEvent("follow", "user-77", "https://redirected.com")

	err := hn.Notify(event)
	require.NoError(t, err)

	assert.Equal(t, "follow", receivedEvent.Action)
	assert.Equal(t, "user-77", receivedEvent.UserID)
	assert.Equal(t, "https://redirected.com", receivedEvent.URL)
}

func TestHTTPNotifier_Notify_BadJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	defer server.Close()

	hn := audit.NewHTTPNotifier(server.URL)
	event := audit.NewEvent("shorten", "u1", "https://test.com")

	err := hn.Notify(event)

	assert.NoError(t, err)
}

func TestHTTPNotifier_Notify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	hn := audit.NewHTTPNotifier(server.URL)
	event := audit.NewEvent("shorten", "u1", "https://test.com")

	err := hn.Notify(event)

	assert.NoError(t, err)
}
