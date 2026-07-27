package audit

// Notifier is the Observer interface for audit notifications.
type Notifier interface {
	Notify(event Event) error
}

// Manager is the Subject in the Observer pattern, holding a list of observers.
type Manager struct {
	Observers []Notifier
}

// NewManager creates a new Manager with an empty observer list.
func NewManager() *Manager {
	return &Manager{Observers: make([]Notifier, 0)}
}

// Register adds a Notifier to the observer list.
func (m *Manager) Register(n Notifier) {
	m.Observers = append(m.Observers, n)
}

// Notify sends an event to all registered observers asynchronously.
func (m *Manager) Notify(event Event) {
	for _, obs := range m.Observers {
		go func(o Notifier) {
			_ = o.Notify(event)
		}(obs)
	}
}
