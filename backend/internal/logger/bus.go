package logger

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ScopeRuntime = "runtime"
	ScopeAccess  = "access"
	ScopeSQL     = "sql"
	ScopeAgent   = "agent"
	ScopeClient  = "client"
	ScopeAudit   = "audit"
)

type Entry struct {
	ID        int64          `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Scope     string         `json:"scope"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields"`
	RequestID string         `json:"requestId"`
	RunID     string         `json:"runId"`
}

type Bus struct {
	mu       sync.RWMutex
	capacity int
	nextID   int64
	entries  []Entry
	scopes   map[string]struct{}
}

func NewBus(capacity int) *Bus {
	if capacity <= 0 {
		capacity = 2000
	}

	return &Bus{
		capacity: capacity,
		entries:  make([]Entry, 0, capacity),
		scopes: map[string]struct{}{
			ScopeRuntime: {},
			ScopeAccess:  {},
			ScopeSQL:     {},
			ScopeAgent:   {},
			ScopeClient:  {},
		},
	}
}

func (b *Bus) Publish(scope, level, message string, fields map[string]any) Entry {
	scope = normalizeScope(scope)
	entryFields := cloneFields(fields)

	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	entry := Entry{
		ID:        b.nextID,
		Timestamp: time.Now(),
		Level:     strings.ToLower(strings.TrimSpace(level)),
		Scope:     scope,
		Message:   message,
		Fields:    entryFields,
		RequestID: stringField(entryFields, "request_id", "requestId"),
		RunID:     stringField(entryFields, "run_id", "runId"),
	}

	if len(b.entries) >= b.capacity {
		b.entries = append(b.entries[1:], entry)
	} else {
		b.entries = append(b.entries, entry)
	}
	b.scopes[scope] = struct{}{}

	return entry
}

func (b *Bus) Query(scope string, cursor int64, limit int) ([]Entry, int64) {
	scope = normalizeScope(scope)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	matches := make([]Entry, 0, limit)
	for _, entry := range b.entries {
		if scope != "" && scope != "all" && entry.Scope != scope {
			continue
		}
		if cursor > 0 && entry.ID <= cursor {
			continue
		}
		matches = append(matches, entry)
	}

	if cursor == 0 && len(matches) > limit {
		matches = matches[len(matches)-limit:]
	} else if len(matches) > limit {
		matches = matches[:limit]
	}

	nextCursor := cursor
	if len(matches) > 0 {
		nextCursor = matches[len(matches)-1].ID
	}

	return matches, nextCursor
}

func (b *Bus) ListScopes() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	items := make([]string, 0, len(b.scopes))
	for scope := range b.scopes {
		items = append(items, scope)
	}
	sort.Strings(items)
	return items
}

func cloneFields(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}

	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func normalizeScope(scope string) string {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope == "" {
		return ScopeRuntime
	}
	return scope
}

func stringField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			if text, ok := value.(string); ok {
				return text
			}
		}
	}
	return ""
}
