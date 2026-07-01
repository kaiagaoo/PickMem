package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qwgao/pickmem/internal/vault"
)

// ActiveResourceURI is the fixed URI that clients read to fetch the
// currently-picked context block. Stable — cited in READMEs, install
// helpers, and downstream clients; do not change without a version bump.
const ActiveResourceURI = "pickmem://active"

// NewServer wires the pickmem MCP server: one resource (pickmem://active)
// and four tools. The server holds a *vault.Store; each request re-reads
// the vault so external edits (Obsidian, other pickmem invocations) are
// picked up between calls.
func NewServer(store *vault.Store, version string) *sdkmcp.Server {
	if version == "" {
		version = "dev"
	}
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "pickmem",
		Version: version,
	}, nil)

	registerResource(srv, store)
	registerTools(srv, store)
	return srv
}

// ---------- resource: pickmem://active ----------

func registerResource(srv *sdkmcp.Server, store *vault.Store) {
	srv.AddResource(&sdkmcp.Resource{
		URI:         ActiveResourceURI,
		Name:        "PickMem active memory",
		Description: "The currently-picked slice of the user's memory vault. Only these items are meant to inform your responses; the user chose them for this session.",
		MIMEType:    "text/markdown",
	}, func(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		// Re-read from disk so an Obsidian edit or a `pickmem pick` in
		// another shell is visible without restarting the server.
		if err := store.Reload(); err != nil {
			return nil, fmt.Errorf("reload vault: %w", err)
		}
		text, err := AssembleActive(store)
		if err != nil {
			return nil, err
		}
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      ActiveResourceURI,
				MIMEType: "text/markdown",
				Text:     text,
			}},
		}, nil
	})
}

// ---------- tools ----------

// getActiveArgs has no fields — kept as an empty struct so the SDK's
// generic AddTool can still infer a valid empty input schema.
type getActiveArgs struct{}

type listLensesArgs struct{}

type useLensArgs struct {
	Name string `json:"name" jsonschema:"the lens name to activate (must already exist)"`
}

type proposeArgs struct {
	ChatText string `json:"chat_text" jsonschema:"the conversation or note text to extract candidate memories from"`
}

func registerTools(srv *sdkmcp.Server, store *vault.Store) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "get_active_memory",
		Description: "Return the currently-picked memory as a single markdown block. Same content as the pickmem://active resource; provided as a tool for clients that don't wire resources to the model automatically.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ getActiveArgs) (*sdkmcp.CallToolResult, any, error) {
		if err := store.Reload(); err != nil {
			return nil, nil, err
		}
		text, err := AssembleActive(store)
		if err != nil {
			return nil, nil, err
		}
		return textResult(text), nil, nil
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "list_lenses",
		Description: "List saved lenses in the user's vault. Returns each lens name and the number of items it contains.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ listLensesArgs) (*sdkmcp.CallToolResult, any, error) {
		if err := store.Reload(); err != nil {
			return nil, nil, err
		}
		ls, err := store.LoadLenses()
		if err != nil {
			return nil, nil, err
		}
		type lensSummary struct {
			Name  string `json:"name"`
			Items int    `json:"items"`
		}
		out := make([]lensSummary, 0, len(ls))
		for _, l := range ls {
			out = append(out, lensSummary{Name: l.Name, Items: len(l.ItemIDs)})
		}
		return jsonResult(out), nil, nil
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "use_lens",
		Description: "Activate a lens: replace the current active selection with the lens's item ids and set active_lens. Persists to pickmem/active.json. Returns the new assembled context so the model sees it immediately.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args useLensArgs) (*sdkmcp.CallToolResult, any, error) {
		if err := store.Reload(); err != nil {
			return nil, nil, err
		}
		ls, err := store.LoadLenses()
		if err != nil {
			return nil, nil, err
		}
		lens, ok := vault.FindLens(ls, args.Name)
		if !ok {
			return errorResult(fmt.Sprintf("lens %q not found", args.Name)), nil, nil
		}
		// Drop ids of since-deleted notes so we don't persist dangling
		// references — same behavior as the TUI's applyLens.
		live := make([]string, 0, len(lens.ItemIDs))
		for _, id := range lens.ItemIDs {
			if _, ok := store.Get(id); ok {
				live = append(live, id)
			}
		}
		if err := store.SaveActive(vault.Active{
			ActiveLens: lens.Name,
			ItemIDs:    live,
		}); err != nil {
			return nil, nil, err
		}
		return textResult(assemble(store, vault.Active{
			ActiveLens: lens.Name,
			ItemIDs:    live,
		})), nil, nil
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "propose_memories",
		Description: "Extract candidate memory items from the given chat text and stage them as pending notes in pickmem/inbox/. Does NOT activate — the user must accept from the picker. Rules-based extraction only in this build; AI classification is opt-in in a future release.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args proposeArgs) (*sdkmcp.CallToolResult, any, error) {
		if err := store.Reload(); err != nil {
			return nil, nil, err
		}
		result, err := ProposeFromChat(store, args.ChatText)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(result), nil, nil
	})
}

// ---------- helpers ----------

func textResult(s string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: s}},
	}
}

func jsonResult(v any) *sdkmcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// This can only happen for pathological Go values; return a
		// readable error instead of blowing up the whole call.
		return errorResult(fmt.Sprintf("marshal result: %v", err))
	}
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
	}
}

// errorResult reports a tool-level (non-transport) error. Setting IsError
// lets the client show it as a failed call rather than as content.
func errorResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: msg}},
	}
}
