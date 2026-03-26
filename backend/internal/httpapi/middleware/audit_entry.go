package middleware

import "github.com/gin-gonic/gin"

const auditEntriesKey = "audit_entries"

type AuditEntry struct {
	Action  string
	Target  string
	Details map[string]any
}

func AppendAuditEntry(c *gin.Context, entry AuditEntry) {
	if entry.Action == "" {
		return
	}

	items, _ := c.Get(auditEntriesKey)
	if existing, ok := items.([]AuditEntry); ok {
		c.Set(auditEntriesKey, append(existing, entry))
		return
	}

	c.Set(auditEntriesKey, []AuditEntry{entry})
}

func auditEntriesFromContext(c *gin.Context) []AuditEntry {
	items, _ := c.Get(auditEntriesKey)
	existing, _ := items.([]AuditEntry)
	return existing
}
