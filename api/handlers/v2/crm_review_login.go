package v2

import (
	"math"
	"strings"

	"github.com/google/uuid"
)

// This reports configuration and key shapes, never login profile values.
// Completeness is not a claim that the bootstrap authorization is compatible.
func crmNativeReviewLogin(raw map[string]any, catalog []map[string]any) (map[string]any, bool) {
	value, ok := raw["login_compatibility"].(map[string]any)
	if !ok {
		return nil, false
	}
	clientCount, ok := crmReviewBoundedCount(value["client_type_count"], 2)
	if !ok {
		return nil, false
	}
	clients, ok := value["client_types"].([]any)
	if !ok || len(clients) != clientCount {
		return nil, false
	}
	catalogSlugs := make(map[string]bool, len(catalog))
	for _, table := range catalog {
		slug, _ := table["slug"].(string)
		catalogSlugs[slug] = true
	}
	seenClients, tables := map[string]bool{}, map[string]bool{}
	safeClients := make([]map[string]any, 0, clientCount)
	complete := clientCount == 2
	for _, item := range clients {
		row, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		id, ok := row["guid"].(string)
		if !ok || !crmReviewLoginClient(id) || seenClients[id] {
			return nil, false
		}
		slug, ok := crmReviewNullableIdentifier(row, "table_slug")
		if !ok {
			return nil, false
		}
		loginTable, _ := slug.(string)
		if loginTable == "" {
			loginTable = "user"
		}
		complete = complete && catalogSlugs[loginTable]
		seenClients[id], tables[loginTable] = true, true
		safeClients = append(safeClients, map[string]any{"guid": id, "table_slug": slug})
	}
	connectionCount, ok := crmReviewBoundedCount(value["connection_count"], crmReviewConnectionLimit)
	if !ok {
		return nil, false
	}
	connections, ok := value["connections"].([]any)
	if !ok || len(connections) != connectionCount {
		return nil, false
	}
	safeConnections := make([]map[string]any, 0, connectionCount)
	seenConnections := make(map[string]bool, connectionCount)
	for _, item := range connections {
		row, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		id, idOK := row["guid"].(string)
		client, clientOK := row["client_type_id"].(string)
		if !idOK || !crmReviewCanonicalUUID(id) || seenConnections[id] || !clientOK || !crmReviewLoginClient(client) {
			return nil, false
		}
		slug, slugOK := crmReviewNullableIdentifier(row, "table_slug")
		field, fieldOK := crmReviewNullableIdentifier(row, "field_slug")
		if !slugOK || !fieldOK {
			return nil, false
		}
		seenConnections[id] = true
		complete = complete && seenClients[client]
		safeConnections = append(safeConnections, map[string]any{"guid": id, "table_slug": slug, "field_slug": field, "client_type_id": client})
	}
	columns, ok := value["login_columns"].([]any)
	if !ok || len(columns) != 4*len(tables) || len(columns) > 8 {
		return nil, false
	}
	safeColumns := make([]map[string]any, 0, len(columns))
	seenColumns := make(map[string]bool, len(columns))
	for _, item := range columns {
		row, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		table, tableOK := row["table"].(string)
		column, columnOK := row["column"].(string)
		present, presentOK := row["present"].(bool)
		if !tableOK || !tables[table] || !columnOK || !crmReviewLoginKey(column) || !presentOK || seenColumns[table+"/"+column] {
			return nil, false
		}
		typeValue, exists := row["type"]
		if !exists {
			return nil, false
		}
		if present {
			text, ok := typeValue.(string)
			if !ok || text == "" || len(text) > 256 || strings.ContainsAny(text, "\r\n\x00") {
				return nil, false
			}
		} else if typeValue != nil {
			return nil, false
		}
		complete = complete && present
		seenColumns[table+"/"+column] = true
		safeColumns = append(safeColumns, map[string]any{"table": table, "column": column, "present": present, "type": typeValue})
	}
	return map[string]any{"client_type_count": clientCount, "client_types": safeClients, "connection_count": connectionCount, "connections": safeConnections, "login_columns": safeColumns, "metadata_complete": complete}, true
}

func crmReviewBoundedCount(value any, limit int) (int, bool) {
	count, ok := value.(float64)
	if !ok || math.IsNaN(count) || math.IsInf(count, 0) || count < 0 || count > float64(limit) || count != math.Trunc(count) {
		return 0, false
	}
	return int(count), true
}

func crmReviewNullableIdentifier(row map[string]any, key string) (any, bool) {
	value, exists := row[key]
	if !exists {
		return nil, false
	}
	if value == nil || value == "" {
		return value, true
	}
	text, ok := value.(string)
	return value, ok && crmNativeReviewIdentifier(text)
}

func crmReviewLoginClient(value string) bool {
	return value == crmReviewClientType || value == crmReviewOperatorClientType
}

func crmReviewLoginKey(value string) bool {
	switch value {
	case "guid", "user_id_auth", "client_type_id", "role_id":
		return true
	}
	return false
}

func crmReviewCanonicalUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed != uuid.Nil && len(value) == 36 && parsed.String() == value
}
