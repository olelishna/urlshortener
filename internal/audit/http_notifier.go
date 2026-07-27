package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// HTTPNotifier sends audit events as JSON POST requests to a remote server.
type HTTPNotifier struct {
	Url    string
	Client *http.Client
}

// NewHTTPNotifier creates a new HTTPNotifier with the given URL.
func NewHTTPNotifier(url string) *HTTPNotifier {
	return &HTTPNotifier{
		Url:    url,
		Client: &http.Client{},
	}
}

// Notify marshals the event to JSON and sends it via POST.
func (n *HTTPNotifier) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	resp, err := n.Client.Post(n.Url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
