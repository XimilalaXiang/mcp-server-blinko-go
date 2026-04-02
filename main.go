package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	blinkoURL string
	blinkoKey string
	authToken string
)

func init() {
	blinkoURL = strings.TrimRight(os.Getenv("BLINKO_DOMAIN"), "/")
	blinkoKey = os.Getenv("BLINKO_API_KEY")
	authToken = os.Getenv("MCP_AUTH_TOKEN")

	if blinkoURL == "" {
		blinkoURL = "http://localhost:1111"
	}
	if !strings.HasPrefix(blinkoURL, "http") {
		blinkoURL = "https://" + blinkoURL
	}
}

func apiRequest(method, path string, body io.Reader) (json.RawMessage, error) {
	url := blinkoURL + "/api/v1" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if blinkoKey != "" {
		req.Header.Set("Authorization", "Bearer "+blinkoKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return json.RawMessage(data), nil
}

func jsonPretty(v json.RawMessage) string {
	var out any
	if json.Unmarshal(v, &out) == nil {
		b, _ := json.MarshalIndent(out, "", "  ")
		return string(b)
	}
	return string(v)
}

func handleUpsertNote(noteType int) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content := req.GetString("content", "")
		if content == "" {
			return mcp.NewToolResultError("content is required"), nil
		}

		payload := map[string]any{
			"content": content,
			"type":    noteType,
		}
		body, _ := json.Marshal(payload)
		result, err := apiRequest("POST", "/note/upsert", strings.NewReader(string(body)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(jsonPretty(result)), nil
	}
}

func handleShareNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)
	noteID, ok := args["noteId"]
	if !ok {
		return mcp.NewToolResultError("noteId is required"), nil
	}

	payload := map[string]any{"noteId": noteID}
	if pw := req.GetString("password", ""); pw != "" {
		payload["password"] = pw
	}
	if cancel, ok := args["isCancel"]; ok {
		payload["isCancel"] = cancel
	}

	body, _ := json.Marshal(payload)
	result, err := apiRequest("POST", "/note/share", strings.NewReader(string(body)))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(jsonPretty(result)), nil
}

func handleSearchNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	searchText := req.GetString("searchText", "")
	if searchText == "" {
		return mcp.NewToolResultError("searchText is required"), nil
	}

	payload := map[string]any{
		"searchText":   searchText,
		"size":         getIntArg(req, "size", 5),
		"isUseAiQuery": true,
	}

	if t, ok := getArgs(req)["type"]; ok {
		payload["type"] = t
	}
	if v, ok := getArgs(req)["isArchived"]; ok {
		payload["isArchived"] = v
	}
	if v, ok := getArgs(req)["hasTodo"]; ok {
		payload["hasTodo"] = v
	}
	if v := req.GetString("startDate", ""); v != "" {
		payload["startDate"] = v
	}
	if v := req.GetString("endDate", ""); v != "" {
		payload["endDate"] = v
	}

	body, _ := json.Marshal(payload)
	result, err := apiRequest("POST", "/note/list", strings.NewReader(string(body)))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(jsonPretty(result)), nil
}

func handleReviewDaily(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := apiRequest("POST", "/note/daily-review", strings.NewReader("{}"))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(jsonPretty(result)), nil
}

func handleClearRecycleBin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := apiRequest("POST", "/note/clear-recycle", strings.NewReader("{}"))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(jsonPretty(result)), nil
}

func getArgs(req mcp.CallToolRequest) map[string]any {
	if m, ok := req.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func getIntArg(req mcp.CallToolRequest, key string, def int) int {
	args := getArgs(req)
	if v, ok := args[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return def
}

func bearerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+authToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Unauthorized. Provide Authorization: Bearer <token> header."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	s := server.NewMCPServer(
		"blinko-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithInstructions("MCP server for Blinko — a self-hosted note service. Provides tools to create flash notes, normal notes, todos, search, review, share, and manage notes."),
	)

	s.AddTool(mcp.NewTool("upsert_blinko_flash_note",
		mcp.WithDescription("Create a flash note (type 0) in Blinko. Flash notes are quick, short-form notes."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Text content of the flash note")),
	), handleUpsertNote(0))

	s.AddTool(mcp.NewTool("upsert_blinko_note",
		mcp.WithDescription("Create a normal note (type 1) in Blinko. Standard notes for longer content."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Text content of the note")),
	), handleUpsertNote(1))

	s.AddTool(mcp.NewTool("upsert_blinko_todo",
		mcp.WithDescription("Create a todo (type 2) in Blinko."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Text content of the todo")),
	), handleUpsertNote(2))

	s.AddTool(mcp.NewTool("share_blinko_note",
		mcp.WithDescription("Share a note or cancel sharing."),
		mcp.WithNumber("noteId", mcp.Required(), mcp.Description("ID of the note to share")),
		mcp.WithString("password", mcp.Description("Optional six-digit password for sharing")),
		mcp.WithBoolean("isCancel", mcp.Description("Whether to cancel sharing (default: false)")),
	), handleShareNote)

	s.AddTool(mcp.NewTool("search_blinko_notes",
		mcp.WithDescription("Search notes in Blinko with various filters."),
		mcp.WithString("searchText", mcp.Required(), mcp.Description("Search keyword")),
		mcp.WithNumber("size", mcp.Description("Number of results to return (default: 5)")),
		mcp.WithNumber("type", mcp.Description("Note type: -1 for all, 0 for flash, 1 for normal")),
		mcp.WithBoolean("isArchived", mcp.Description("Search in archived notes")),
		mcp.WithBoolean("hasTodo", mcp.Description("Search only in notes with todos")),
		mcp.WithString("startDate", mcp.Description("Start date in ISO format")),
		mcp.WithString("endDate", mcp.Description("End date in ISO format")),
	), handleSearchNotes)

	s.AddTool(mcp.NewTool("review_blinko_daily_notes",
		mcp.WithDescription("Get today's notes for daily review."),
	), handleReviewDaily)

	s.AddTool(mcp.NewTool("clear_blinko_recycle_bin",
		mcp.WithDescription("Clear the recycle bin in Blinko. This action is irreversible."),
	), handleClearRecycleBin)

	transport := os.Getenv("MCP_TRANSPORT")
	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = "8080"
	}

	switch transport {
	case "sse":
		log.Printf("Starting Blinko MCP SSE server on :%s", port)
		sseServer := server.NewSSEServer(s,
			server.WithSSEEndpoint("/sse"),
			server.WithMessageEndpoint("/message"),
		)
		if authToken != "" {
			log.Printf("Bearer auth enabled (token: %s...%s)", authToken[:4], authToken[len(authToken)-4:])
		}
		mux := http.NewServeMux()
		mux.Handle("/", bearerAuthMiddleware(sseServer))
		httpSrv := &http.Server{Addr: ":" + port, Handler: mux}
		if err := httpSrv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	case "http":
		log.Printf("Starting Blinko MCP HTTP server on :%s", port)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + port); err != nil {
			log.Fatal(err)
		}
	default:
		log.Println("Starting Blinko MCP server (stdio)")
		if err := server.ServeStdio(s); err != nil {
			log.Fatal(err)
		}
	}
}
