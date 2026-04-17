package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/example/ai-integration-test-demo/ai/knowledge"
	"github.com/example/ai-integration-test-demo/ai/lsp"
	"github.com/example/ai-integration-test-demo/ai/prompt"
	"github.com/example/ai-integration-test-demo/ai/session"
	"github.com/example/ai-integration-test-demo/ai/tools"
	oai "github.com/sashabaranov/go-openai"
)

type Agent struct {
	provider   Provider
	session    *session.Session
	model      string
	mode       string
	promptOpts prompt.PromptOptions
	fm         *knowledge.FileManager
	lspClient  *lsp.Client
}

// New creates an Agent with the appropriate LLM provider.
// Provider selection: "codex" → Codex CLI, "anthropic:" prefix → Anthropic API, otherwise → OpenAI API.
// lspClient may be nil if LSP is unavailable.
func New(apiKey, model, baseURL string, sess *session.Session, mode string, promptOpts prompt.PromptOptions, fm *knowledge.FileManager, lspClient *lsp.Client) *Agent {
	var prov Provider
	if apiKey == "codex" {
		log.Printf("using Codex CLI provider (model=%s)", model)
		prov = NewCodexProvider(model)
	} else if strings.HasPrefix(baseURL, "https://open.bigmodel.cn/api/anthropic") || strings.Contains(baseURL, "/anthropic") {
		log.Printf("using Anthropic API provider (model=%s, baseURL=%s)", model, baseURL)
		prov = NewAnthropicProvider(apiKey, baseURL, model)
	} else {
		log.Printf("using OpenAI API provider (model=%s, baseURL=%s)", model, baseURL)
		prov = NewOpenAIProvider(apiKey, baseURL)
	}

	return &Agent{
		provider:   prov,
		session:    sess,
		model:      model,
		mode:       mode,
		promptOpts: promptOpts,
		fm:         fm,
		lspClient:  lspClient,
	}
}

func (a *Agent) Run(ctx context.Context, taskDesc string) (string, error) {
	sysPrompt := prompt.BuildPrompt(a.mode, a.promptOpts)

	messages := []oai.ChatCompletionMessage{
		{Role: oai.ChatMessageRoleSystem, Content: sysPrompt},
		{Role: oai.ChatMessageRoleUser, Content: taskDesc},
	}

	toolDefs := tools.Definitions(a.mode, a.lspClient != nil)

	maxIter := 80
	if a.mode == "code-only" {
		maxIter = 15
	}

	warnAt := maxIter - 8
	forcedAt := maxIter - 3

	for i := 0; i < maxIter; i++ {
		log.Printf("\n--- iteration %d/%d ---", i+1, maxIter)

		if i == warnAt {
			messages = append(messages, oai.ChatCompletionMessage{
				Role:    oai.ChatMessageRoleUser,
				Content: "[SYSTEM] You are approaching the iteration limit. Start wrapping up now. Produce your final Correlation Map and Defect Report. Do NOT make more tool calls.",
			})
		}

		req := oai.ChatCompletionRequest{
			Model:    a.model,
			Messages: messages,
		}

		if i < forcedAt && len(toolDefs) > 0 {
			req.Tools = toolDefs
		}

		resp, err := a.provider.ChatCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("%s error: %w", a.provider.Name(), err)
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		// Log the agent's reasoning/thinking text if present
		if thinking := strings.TrimSpace(choice.Message.Content); thinking != "" {
			log.Printf("[Thinking] %s", thinking)
		}

		if choice.FinishReason == oai.FinishReasonStop {
			log.Printf("--- agent finished (iteration %d) ---", i+1)
			return choice.Message.Content, nil
		}

		if choice.FinishReason == oai.FinishReasonToolCalls {
			if i >= forcedAt {
				messages = append(messages, oai.ChatCompletionMessage{
					Role:       oai.ChatMessageRoleTool,
					Content:    "[SYSTEM] Iteration limit reached. You MUST produce your final report now without any more tool calls.",
					ToolCallID: choice.Message.ToolCalls[0].ID,
				})
				continue
			}
			for _, tc := range choice.Message.ToolCalls {
				result, err := a.handleToolCall(ctx, tc)
				if err != nil {
					result = fmt.Sprintf("error: %v", err)
				}
				messages = append(messages, oai.ChatCompletionMessage{
					Role:       oai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
			continue
		}

		log.Printf("unexpected finish reason: %s", choice.FinishReason)
	}

	return "", fmt.Errorf("max iterations reached")
}

func (a *Agent) handleToolCall(ctx context.Context, tc oai.ToolCall) (string, error) {
	switch tc.Function.Name {
	case "send_command":
		var params tools.SendCommandParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		log.Printf("AI → %s %+v", params.Cmd, params)
		result, err := a.session.SendCommand(params)
		if err != nil {
			return "", err
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		return string(out), nil

	case "read_file":
		var params tools.ReadFileParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		log.Printf("AI → read_file %s", params.Path)
		content, err := a.fm.ReadFile(params.Path)
		if err != nil {
			return err.Error(), nil
		}
		return content, nil

	case "search_code":
		var params tools.SearchCodeParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		log.Printf("AI → search_code %s %q", params.Directory, params.Pattern)
		results, err := a.fm.SearchCode(params.Directory, params.Pattern)
		if err != nil {
			return err.Error(), nil
		}
		out, _ := json.MarshalIndent(results, "", "  ")
		return string(out), nil

	case "update_knowledge":
		var params tools.UpdateKnowledgeParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		log.Printf("AI → update_knowledge (%d bytes)", len(params.Content))
		if err := a.fm.UpdateKnowledge(params.Content); err != nil {
			return err.Error(), nil
		}
		return "knowledge.md updated", nil

	case "register_cmd":
		var params tools.RegisterCmdParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		log.Printf("AI → register_cmd name=%s target=%s action=%s", params.Name, params.Target, params.Action)
		result, err := a.session.SendCommand(tools.SendCommandParams{
			Cmd:    "register_cmd",
			Name:   params.Name,
			Target: params.Target,
			Action: params.Action,
			Desc:   params.Description,
		})
		if err != nil {
			return "", err
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		return string(out), nil

	case "lsp_references":
		var params tools.LSPReferencesParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if a.lspClient == nil {
			return "LSP not available — use search_code as fallback", nil
		}
		log.Printf("AI → lsp_references %s %s", params.File, params.Symbol)
		refs, err := a.lspClient.References(ctx, params.File, params.Symbol)
		if err != nil {
			return fmt.Sprintf("lsp_references error: %v", err), nil
		}
		out, _ := json.MarshalIndent(refs, "", "  ")
		return string(out), nil

	case "lsp_definition":
		var params tools.LSPDefinitionParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if a.lspClient == nil {
			return "LSP not available — use read_file as fallback", nil
		}
		log.Printf("AI → lsp_definition %s %s", params.File, params.Symbol)
		defs, err := a.lspClient.Definition(ctx, params.File, params.Symbol)
		if err != nil {
			return fmt.Sprintf("lsp_definition error: %v", err), nil
		}
		out, _ := json.MarshalIndent(defs, "", "  ")
		return string(out), nil

	case "lsp_symbols":
		var params tools.LSPSymbolsParams
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if a.lspClient == nil {
			return "LSP not available — use search_code as fallback", nil
		}
		log.Printf("AI → lsp_symbols %q", params.Query)
		syms, err := a.lspClient.Symbols(ctx, params.Query)
		if err != nil {
			return fmt.Sprintf("lsp_symbols error: %v", err), nil
		}
		out, _ := json.MarshalIndent(syms, "", "  ")
		return string(out), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}
