package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	applicationaudit "kubeclaw/backend/internal/application/audit"
	"kubeclaw/backend/internal/logger"
)

type Entry = logger.Entry

type QueryInput struct {
	Scope  string `json:"scope"`
	Cursor int64  `json:"cursor"`
	Limit  int    `json:"limit"`
}

type QueryResult struct {
	Scope      string  `json:"scope"`
	Cursor     int64   `json:"cursor"`
	NextCursor int64   `json:"nextCursor"`
	Entries    []Entry `json:"entries"`
}

type ClientLogInput struct {
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	RequestID string         `json:"requestId"`
	RunID     string         `json:"runId"`
	Route     string         `json:"route"`
	Fields    map[string]any `json:"fields"`
}

type Service struct {
	bus *logger.Bus
}

func NewService(bus *logger.Bus) *Service {
	return &Service{bus: bus}
}

func (s *Service) ListScopes() []string {
	scopes := s.bus.ListScopes()
	hasAudit := false
	for _, scope := range scopes {
		if scope == logger.ScopeAudit {
			hasAudit = true
			break
		}
	}
	if !hasAudit {
		scopes = append(scopes, logger.ScopeAudit)
		sort.Strings(scopes)
	}
	return scopes
}

func (s *Service) Query(ctx context.Context, input QueryInput, auditLogs []applicationaudit.Record) (*QueryResult, error) {
	scope := strings.TrimSpace(strings.ToLower(input.Scope))
	if scope == "" {
		scope = logger.ScopeRuntime
	}

	if scope == logger.ScopeAudit {
		entries, nextCursor := queryAuditEntries(input.Cursor, input.Limit, auditLogs)
		return &QueryResult{
			Scope:      scope,
			Cursor:     input.Cursor,
			NextCursor: nextCursor,
			Entries:    entries,
		}, nil
	}

	entries, nextCursor := s.bus.Query(scope, input.Cursor, input.Limit)
	return &QueryResult{
		Scope:      scope,
		Cursor:     input.Cursor,
		NextCursor: nextCursor,
		Entries:    entries,
	}, nil
}

func (s *Service) RecordClient(_ context.Context, input ClientLogInput) Entry {
	fields := map[string]any{}
	for key, value := range input.Fields {
		fields[key] = value
	}
	if input.Route != "" {
		fields["route"] = input.Route
	}
	if input.RequestID != "" {
		fields["request_id"] = input.RequestID
	}
	if input.RunID != "" {
		fields["run_id"] = input.RunID
	}

	level := strings.TrimSpace(strings.ToLower(input.Level))
	if level == "" {
		level = "info"
	}

	return s.bus.Publish(logger.ScopeClient, level, input.Message, fields)
}

func queryAuditEntries(cursor int64, limit int, records []applicationaudit.Record) ([]Entry, int64) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	entries := make([]Entry, 0, limit)
	for index := len(records) - 1; index >= 0; index-- {
		record := records[index]
		if cursor > 0 && record.ID <= cursor {
			continue
		}

		details := map[string]any{}
		_ = json.Unmarshal([]byte(record.Details), &details)

		fields := map[string]any{
			"action": record.Action,
			"target": record.Target,
			"ip":     record.IP,
		}
		for key, value := range details {
			fields[key] = value
		}

		message := fmt.Sprintf("%s -> %s", record.Action, record.Target)
		requestID, _ := details["requestId"].(string)
		runID, _ := details["runId"].(string)
		entries = append(entries, Entry{
			ID:        record.ID,
			Timestamp: record.CreatedAt,
			Level:     "info",
			Scope:     logger.ScopeAudit,
			Message:   message,
			Fields:    fields,
			RequestID: requestID,
			RunID:     runID,
		})
	}

	if cursor == 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	} else if len(entries) > limit {
		entries = entries[:limit]
	}

	nextCursor := cursor
	if len(entries) > 0 {
		nextCursor = entries[len(entries)-1].ID
	}

	return entries, nextCursor
}
