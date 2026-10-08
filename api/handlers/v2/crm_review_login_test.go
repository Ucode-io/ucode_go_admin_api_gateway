package v2

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func reviewFixtureLoginMetadata() map[string]any {
	columns := []any{}
	for _, key := range []string{"guid", "user_id_auth", "client_type_id", "role_id"} {
		columns = append(columns, map[string]any{"table": "users", "column": key, "present": true, "type": "uuid", "value": "secret-fixture", "default": "secret-fixture"})
	}
	return map[string]any{
		"client_type_count": 2.,
		"client_types": []any{
			map[string]any{"guid": crmReviewClientType, "table_slug": "users", "password": "secret-fixture"},
			map[string]any{"guid": crmReviewOperatorClientType, "table_slug": "users", "token": "secret-fixture"},
		},
		"connection_count": 1.,
		"connections":      []any{map[string]any{"guid": "33333333-3333-4333-8333-333333333333", "table_slug": "contacts", "field_slug": "contacts_id", "client_type_id": crmReviewOperatorClientType, "data": "secret-fixture"}},
		"login_columns":    columns, "unapproved": "secret-fixture",
	}
}

func reviewLoginReport(raw map[string]any) (map[string]any, bool) {
	return crmNativeReviewReport(map[string]any{"data": []any{map[string]any{"result": raw}}})
}

func TestCRMNativeReviewLoginMetadataOnly(t *testing.T) {
	report, ok := reviewLoginReport(reviewFixtureReport())
	if !ok {
		t.Fatal("complete login metadata rejected")
	}
	login := report["login_compatibility"].(map[string]any)
	if login["metadata_complete"] != true || login["connection_count"] != 1 || len(login) != 6 {
		t.Fatal("login metadata coverage lost")
	}
	// A protected connection target is a reportable configuration fact, never
	// automatic compatibility or authorization to read that business table.
	if login["connections"].([]map[string]any)[0]["table_slug"] != "contacts" || report["safe_to_activate"] != false {
		t.Fatal("protected target was hidden or authorized")
	}
	encoded, err := json.Marshal(report)
	if err != nil || strings.Contains(string(encoded), "secret-fixture") || strings.Contains(string(encoded), "compatible\"") {
		t.Fatal("login values or compatibility authority leaked")
	}
	for _, term := range []string{"SELECT *", "password", "app_secret", "attributes", "EXECUTE", "INSERT ", "UPDATE ", "DELETE "} {
		if strings.Contains(strings.ToUpper(crmReviewLoginSQL), strings.ToUpper(term)) {
			t.Fatal("login metadata query expanded", term)
		}
	}
	if !strings.Contains(crmReviewLoginSQL, "LIMIT 65") || !strings.Contains(crmReviewLoginSQL, "LEFT JOIN pg_catalog.pg_attribute") || !strings.Contains(crmReviewLoginSQL, crmReviewClientType) || !strings.Contains(crmReviewLoginSQL, crmReviewOperatorClientType) {
		t.Fatal("fixed login metadata bound/missing-key evidence changed")
	}
}

func TestCRMNativeReviewLoginMissingFactsAreExplicit(t *testing.T) {
	for _, mode := range []string{"missing-column", "zero-clients", "one-client", "empty-connections", "fallback-null", "fallback-empty"} {
		t.Run(mode, func(t *testing.T) {
			raw := reviewFixtureReport()
			value := raw["login_compatibility"].(map[string]any)
			columns := value["login_columns"].([]any)
			wantComplete := mode == "empty-connections"
			switch mode {
			case "missing-column":
				columns[0].(map[string]any)["present"] = false
				columns[0].(map[string]any)["type"] = nil
			case "zero-clients":
				value["client_type_count"], value["client_types"], value["login_columns"] = 0., []any{}, []any{}
				value["connection_count"], value["connections"] = 0., []any{}
			case "one-client":
				value["client_type_count"], value["client_types"] = 1., value["client_types"].([]any)[:1]
			case "empty-connections":
				value["connection_count"], value["connections"] = 0., []any{}
			case "fallback-null", "fallback-empty":
				for _, item := range value["client_types"].([]any) {
					if mode == "fallback-null" {
						item.(map[string]any)["table_slug"] = nil
					} else {
						item.(map[string]any)["table_slug"] = ""
					}
				}
				for _, item := range columns {
					item.(map[string]any)["table"] = "user"
				}
			}
			report, ok := reviewLoginReport(raw)
			if !ok || report["login_compatibility"].(map[string]any)["metadata_complete"] != wantComplete {
				t.Fatal("missing mapping/key was denied or treated as complete", mode)
			}
			if mode == "missing-column" {
				fact := report["login_compatibility"].(map[string]any)["login_columns"].([]map[string]any)[0]
				if fact["present"] != false || fact["type"] != nil {
					t.Fatal("missing key evidence erased")
				}
			}
		})
	}
}

func TestCRMNativeReviewLoginRejectsMalformed(t *testing.T) {
	for _, mode := range []string{"missing", "missing-client-count", "fractional-client-count", "nan-client-count", "client-count-mismatch", "foreign-client", "duplicate-client", "missing-client-slug", "invalid-client-slug", "missing-connection-count", "infinite-connection-count", "negative-connection-count", "connection-count-mismatch", "connection-overflow", "bad-connection-guid", "duplicate-connection", "foreign-connection-client", "missing-connection-field", "invalid-connection-slug", "missing-columns", "wrong-column-count", "unmapped-column-table", "duplicate-column", "password-column", "missing-present", "missing-type", "absent-nonnull-type", "present-empty-type", "type-control"} {
		t.Run(mode, func(t *testing.T) {
			raw := reviewFixtureReport()
			value := raw["login_compatibility"].(map[string]any)
			clients := value["client_types"].([]any)
			connections := value["connections"].([]any)
			connection := connections[0].(map[string]any)
			columns := value["login_columns"].([]any)
			column := columns[0].(map[string]any)
			switch mode {
			case "missing":
				delete(raw, "login_compatibility")
			case "missing-client-count":
				delete(value, "client_type_count")
			case "fractional-client-count":
				value["client_type_count"] = 1.5
			case "nan-client-count":
				value["client_type_count"] = math.NaN()
			case "client-count-mismatch":
				value["client_type_count"] = 1.
			case "foreign-client":
				clients[0].(map[string]any)["guid"] = "44444444-4444-4444-8444-444444444444"
			case "duplicate-client":
				clients[1].(map[string]any)["guid"] = crmReviewClientType
			case "missing-client-slug":
				delete(clients[0].(map[string]any), "table_slug")
			case "invalid-client-slug":
				clients[0].(map[string]any)["table_slug"] = "users; DROP TABLE deals"
			case "missing-connection-count":
				delete(value, "connection_count")
			case "infinite-connection-count":
				value["connection_count"] = math.Inf(1)
			case "negative-connection-count":
				value["connection_count"] = -1.
			case "connection-count-mismatch":
				value["connection_count"] = 0.
			case "connection-overflow":
				value["connection_count"] = 65.
			case "bad-connection-guid":
				connection["guid"] = "not-a-uuid"
			case "duplicate-connection":
				value["connections"], value["connection_count"] = append(connections, connection), 2.
			case "foreign-connection-client":
				connection["client_type_id"] = "foreign"
			case "missing-connection-field":
				delete(connection, "field_slug")
			case "invalid-connection-slug":
				connection["table_slug"] = "https://injected.invalid"
			case "missing-columns":
				delete(value, "login_columns")
			case "wrong-column-count":
				value["login_columns"] = columns[:3]
			case "unmapped-column-table":
				column["table"] = "deals"
			case "duplicate-column":
				columns[1] = column
			case "password-column":
				column["column"] = "password"
			case "missing-present":
				delete(column, "present")
			case "missing-type":
				delete(column, "type")
			case "absent-nonnull-type":
				column["present"] = false
			case "present-empty-type":
				column["type"] = ""
			case "type-control":
				column["type"] = "uuid\nsecret"
			}
			if _, ok := reviewLoginReport(raw); ok {
				t.Fatal("malformed login projection accepted", mode)
			}
		})
	}
}

func TestCRMNativeReviewLoginConnectionLimit(t *testing.T) {
	raw := reviewFixtureReport()
	value := raw["login_compatibility"].(map[string]any)
	connections := make([]any, 0, crmReviewConnectionLimit)
	for i := 0; i < crmReviewConnectionLimit; i++ {
		connections = append(connections, map[string]any{"guid": fmt.Sprintf("33333333-3333-4333-8333-%012x", i), "table_slug": nil, "field_slug": "", "client_type_id": crmReviewClientType})
	}
	value["connections"], value["connection_count"] = connections, float64(len(connections))
	if _, ok := reviewLoginReport(raw); !ok {
		t.Fatal("bounded complete connection list rejected")
	}
}
