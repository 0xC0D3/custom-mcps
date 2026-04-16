package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
	"github.com/0xC0D3/custom-mcps/framework/server"
)

// ListZonesInput defines the input parameters for the list_zones tool.
type ListZonesInput struct {
	// Page is the page number of results to return.
	Page int `json:"page" jsonschema:"description=Page number,minimum=1,default=1"`
	// PerPage is the number of results per page.
	PerPage int `json:"perPage" jsonschema:"description=Results per page,minimum=5,maximum=50,default=20"`
}

// ListDNSRecordsInput defines the input parameters for the list_dns_records tool.
type ListDNSRecordsInput struct {
	// ZoneID is the Cloudflare zone identifier.
	ZoneID string `json:"zoneId" mcp:"required" jsonschema:"description=The Cloudflare zone ID"`
	// RecordType filters results by DNS record type.
	RecordType string `json:"recordType" jsonschema:"description=Filter by record type,enum=A|AAAA|CNAME|MX|TXT|NS|SRV"`
	// Name filters results by record name.
	Name string `json:"name" jsonschema:"description=Filter by record name"`
}

// CreateDNSRecordInput defines the input parameters for the create_dns_record tool.
type CreateDNSRecordInput struct {
	// ZoneID is the Cloudflare zone identifier.
	ZoneID string `json:"zoneId" mcp:"required" jsonschema:"description=The Cloudflare zone ID"`
	// Type is the DNS record type to create.
	Type string `json:"type" mcp:"required" jsonschema:"description=DNS record type,enum=A|AAAA|CNAME|MX|TXT"`
	// Name is the DNS record name (e.g. example.com).
	Name string `json:"name" mcp:"required" jsonschema:"description=DNS record name (e.g. example.com)"`
	// Content is the DNS record value (e.g. 192.168.1.1).
	Content string `json:"content" mcp:"required" jsonschema:"description=DNS record content (e.g. 192.168.1.1)"`
	// TTL is the time-to-live in seconds. Use 1 for automatic.
	TTL int `json:"ttl" jsonschema:"description=TTL in seconds (1 for auto),minimum=1,default=1"`
	// Proxied indicates whether the record is proxied through Cloudflare.
	Proxied *bool `json:"proxied" jsonschema:"description=Whether the record is proxied through Cloudflare"`
}

// registerTools registers all Cloudflare DNS tools on the given server.
func registerTools(srv *server.Server) {
	server.Tool(srv, "list_zones", "List Cloudflare DNS zones accessible by the configured API token", handleListZones)
	server.Tool(srv, "list_dns_records", "List DNS records for a specific Cloudflare zone", handleListDNSRecords)
	server.Tool(srv, "create_dns_record", "Create a new DNS record in a Cloudflare zone", handleCreateDNSRecord)
}

// handleListZones returns a list of DNS zones.
// TODO: Replace with actual Cloudflare API calls.
func handleListZones(_ context.Context, input ListZonesInput) (*protocol.CallToolResult, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 20
	}

	zones := []map[string]any{
		{
			"id":          "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
			"name":        "example.com",
			"status":      "active",
			"paused":      false,
			"type":        "full",
			"nameServers": []string{"aria.ns.cloudflare.com", "kyle.ns.cloudflare.com"},
		},
		{
			"id":          "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3",
			"name":        "myapp.dev",
			"status":      "active",
			"paused":      false,
			"type":        "full",
			"nameServers": []string{"beth.ns.cloudflare.com", "otto.ns.cloudflare.com"},
		},
	}

	result := map[string]any{
		"zones":   zones,
		"page":    page,
		"perPage": perPage,
		"total":   2,
	}

	return jsonResult(result)
}

// handleListDNSRecords returns DNS records for a given zone.
// TODO: Replace with actual Cloudflare API calls.
func handleListDNSRecords(_ context.Context, input ListDNSRecordsInput) (*protocol.CallToolResult, error) {
	records := []map[string]any{
		{
			"id":       "b12c34d56e78f90a1b2c3d4e5f6a7b8c",
			"type":     "A",
			"name":     "example.com",
			"content":  "203.0.113.50",
			"ttl":      1,
			"proxied":  true,
			"zoneId":   input.ZoneID,
			"zoneName": "example.com",
		},
		{
			"id":       "c23d45e67f89a01b2c3d4e5f6a7b8c9d",
			"type":     "CNAME",
			"name":     "www.example.com",
			"content":  "example.com",
			"ttl":      3600,
			"proxied":  true,
			"zoneId":   input.ZoneID,
			"zoneName": "example.com",
		},
		{
			"id":       "d34e56f78a90b12c3d4e5f6a7b8c9d0e",
			"type":     "MX",
			"name":     "example.com",
			"content":  "mail.example.com",
			"ttl":      3600,
			"proxied":  false,
			"priority": 10,
			"zoneId":   input.ZoneID,
			"zoneName": "example.com",
		},
		{
			"id":       "e45f67a89b01c23d4e5f6a7b8c9d0e1f",
			"type":     "TXT",
			"name":     "example.com",
			"content":  "v=spf1 include:_spf.google.com ~all",
			"ttl":      1,
			"proxied":  false,
			"zoneId":   input.ZoneID,
			"zoneName": "example.com",
		},
	}

	// Apply filters if provided.
	filtered := make([]map[string]any, 0, len(records))
	for _, r := range records {
		if input.RecordType != "" && r["type"] != input.RecordType {
			continue
		}
		if input.Name != "" && r["name"] != input.Name {
			continue
		}
		filtered = append(filtered, r)
	}

	result := map[string]any{
		"records": filtered,
		"total":   len(filtered),
	}

	return jsonResult(result)
}

// handleCreateDNSRecord creates a new DNS record in a zone.
// TODO: Replace with actual Cloudflare API calls.
func handleCreateDNSRecord(_ context.Context, input CreateDNSRecordInput) (*protocol.CallToolResult, error) {
	ttl := input.TTL
	if ttl < 1 {
		ttl = 1
	}

	proxied := false
	if input.Proxied != nil {
		proxied = *input.Proxied
	}

	record := map[string]any{
		"id":       "f56a78b90c12d34e5f6a7b8c9d0e1f2a",
		"type":     input.Type,
		"name":     input.Name,
		"content":  input.Content,
		"ttl":      ttl,
		"proxied":  proxied,
		"zoneId":   input.ZoneID,
		"zoneName": "example.com",
		"created":  "2026-04-16T10:00:00Z",
		"modified": "2026-04-16T10:00:00Z",
	}

	result := map[string]any{
		"success": true,
		"record":  record,
		"message": fmt.Sprintf("DNS record %s %s -> %s created successfully", input.Type, input.Name, input.Content),
	}

	return jsonResult(result)
}

// jsonResult marshals v to indented JSON and wraps it in a CallToolResult
// with a single text content block.
func jsonResult(v any) (*protocol.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return &protocol.CallToolResult{
		Content: []protocol.Content{protocol.TextContent(string(data))},
	}, nil
}
