package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// ActiveResourceURI is the fixed URI that clients read to fetch the
// currently-picked context block. Stable — cited in READMEs, install
// helpers, and downstream clients; do not change without a version bump.
const ActiveResourceURI = "pickmem://active"

// serverInstructions is surfaced to the client at initialize time (the
// MCP protocol's standard place for "how to use this server"). Clients
// that pass it through to the model (Claude Desktop does) use it as
// steering, not just documentation — this is what makes the model check
// memory proactively instead of only on an explicit "check my memory"
// prompt. Without this, get_active_memory sits unused most of the time:
// its tool description alone isn't enough signal for a model to reach
// for it unprompted.
const serverInstructions = `PickMem exposes a slice of the user's personal memory that they deliberately selected for this session — not their whole history, just what they picked.

Call get_active_memory (or read the pickmem://active resource) near the start of the conversation, and again whenever the user asks something that might depend on personal context: preferences, facts about their life, past decisions, ongoing projects. Don't wait to be told to check memory — that defeats the purpose of the user having picked it.

If the user references something that sounds like it should be in memory but get_active_memory comes back empty or unrelated, say so plainly rather than guessing — the user may need to run their picker again.

Use list_lenses and use_lens if the user mentions a saved lens by name or asks to switch context (e.g. "switch to my Job-Hunt lens").

Use propose_memories only when the user asks you to save or remember something from the conversation. It stages candidates for review — it never activates them, so tell the user to run their review step to finish the save.`

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
	}, &sdkmcp.ServerOptions{
		Instructions: serverInstructions,
	})

	registerResource(srv, store)
	registerTools(srv, store)
	return srv
}

// ---------- resource: pickmem://active ----------

func registerResource(srv *sdkmcp.Server, store *vault.Store) {
	srv.AddResource(&sdkmcp.Resource{
		URI:         ActiveResourceURI,
		Name:        "PickMem active memory",
		Description: "The user's deliberately-picked memory for this session. Read this near the start of the conversation and whenever a question might depend on personal context — don't wait for the user to ask you to check it.",
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
		Description: "Fetch the user's currently-picked memory (same content as the pickmem://active resource). Call this proactively near the start of a conversation and whenever the user's question might depend on personal context, preferences, or facts they've saved — not only when explicitly asked to check memory.",
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
