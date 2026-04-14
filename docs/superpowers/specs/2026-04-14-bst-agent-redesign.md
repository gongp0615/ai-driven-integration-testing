# BST-Agent Redesign Spec

Date: 2026-04-14

## Summary

Redesign the BST-Agent demo to increase research value: replace 3 shallow bugs with a 4-layer 7-bug system, give Agent true autonomy (self-reading source code + maintaining knowledge file), add 2x2 ablation matrix, and overhaul evaluation metrics.

## Scope

- Code: `ai-integration-test-demo/` all Go source files
- README: chapters 4-6 and appendix A (chapters 1-3 minimal edits)
- Language: Chinese for README

## Decision Record

| Decision | Choice |
|----------|--------|
| Scope | Code + README sync |
| Bug strategy | Full replace with 4-layer system |
| CodeAnalyzer | Agent reads source autonomously, maintains knowledge.md |
| Ablation | 2x2 matrix (code yes/no x step yes/no) |
| Metrics | Full set: P/R/F1 + level score + false positive + exploration efficiency |
| README language | Chinese |

---

## 1. Bug Design (7 bugs, 4 layers)

### Difficulty Distribution

| Difficulty | Count | Bug IDs |
|-----------|-------|---------|
| Simple | 1 | B1 |
| Medium | 4 | B2, B3, B4, B5 |
| Hidden | 2 | B6, B7 |

### B1: RemoveItem negative count (L1 Simple, Critical, bag.go)

Retained from original Bug #2. RemoveItem lacks `count <= 0` check while AddItem has it.

Discovery: code symmetry comparison OR construct `removeitem(2001, -1)`.

```go
// bag.go — RemoveItem
func (b *Bag) RemoveItem(itemID, count int) bool {
    it, ok := b.items[itemID]
    if !ok || it.Count < count { // count=-1 passes this check
        return false
    }
    it.Count -= count // count=-1 → Count += 1 → item duplication
    // ...
}
```

### B2: Task repeat trigger after completion (L2 Medium, High, task.go)

Replaces original Bug #1 (hardcoded progress). onItemAdded calls Progress regardless of task state. Progress does not check `State == StateCompleted`. Result: adding items for a completed task re-publishes `task.completed`, causing duplicate achievement unlocks and mail sends.

Discovery: complete a task, then additem for same item again → Step → observe duplicate `task.completed` in logs. Requires specific operation sequence + Step observation.

```go
// task.go
func (ts *TaskSystem) onItemAdded(e event.Event) {
    // ... parse itemID, playerID ...
    taskMapping := map[int]int{2001: 3001, 2002: 3002}
    if tid, ok := taskMapping[itemID]; ok {
        ts.Progress(tid, 1) // called regardless of task state
    }
}

func (ts *TaskSystem) Progress(taskID, delta int) {
    t, ok := ts.tasks[taskID]
    if !ok {
        return
    }
    // Bug: no t.State == StateCompleted check
    t.Progress += delta
    if t.Progress >= t.Target {
        t.State = StateCompleted
        ts.bus.Publish(event.Event{
            Type: "task.completed",
            Data: map[string]any{"playerID": ts.playerID, "taskID": taskID},
        })
    }
}
```

Also: `Progress(tid, 1)` hardcoded `1` is changed to pass actual count from event data. This removes the original Bug #1 (hardcoded 1) and replaces it with the more subtle repeat-trigger bug.

### B3: collector_100 wrong counter (L2 Medium, Medium, achievement.go)

`onItemAdded` counts unlocked achievements instead of item types in bag. Achievement name `collector_100` implies "collect items", but trigger condition is "2 other achievements unlocked".

Code logic is complete and runnable — no crash. Only discoverable by understanding the semantic mismatch between name and implementation.

```go
// achievement.go — onItemAdded
func (as *AchievementSystem) onItemAdded(e event.Event) {
    collectorAchID := 4003
    if ach, exists := as.achs[collectorAchID]; exists && ach.State == AchLocked {
        totalItems := 0
        for _, a := range as.achs {
            if a.State == AchUnlocked {
                totalItems++ // Bug: counts achievements, not item types
            }
        }
        if totalItems >= 2 {
            as.Unlock(collectorAchID)
        }
    }
}
```

### B4: ClaimReward no independent idempotency (L3 Medium-Hard, High, signin.go)

Replaces original Bug #3 (more subtle version). CheckIn has `Claimed` guard, but ClaimReward has none. Agent must understand CheckIn and ClaimReward are separate actions each needing independent idempotency.

```go
// signin.go
func (ss *SignInSystem) CheckIn(day int) {
    d, ok := ss.days[day]
    if !ok { return }
    if d.Claimed { return } // CheckIn has guard
    d.Claimed = true
    ss.bus.Publish(event.Event{Type: "signin.claimed", ...})
}

func (ss *SignInSystem) ClaimReward(day int) {
    d, ok := ss.days[day]
    if !ok { return }
    // Bug: no independent claimedReward guard
    ss.bus.Publish(event.Event{Type: "signin.reward", ...})
}
```

### B5: Equip does not consume bag items (L3 Medium-Hard, Critical, equipment.go)

Equip updates equipment slot but does not call Bag.RemoveItem. Result: item exists in both bag and equipment slot simultaneously — item duplication via second path.

Discovery: additem(3001,1) → Step → observe [Bag] add + [Equipment] auto-equip → Query bag shows 3001 still present → Query equipment shows weapon=3001 → item exists in two places.

```go
// equipment.go
func (es *EquipmentSystem) Equip(slot EquipSlot, itemID int) {
    es.slots[slot] = itemID // does not call Bag.RemoveItem
    // ...
}
```

### B6: mail.claimed event has no subscriber (L4 Hidden, High, mail.go)

ClaimAttachment publishes `mail.claimed` event with item data, but no module subscribes to it. Mail attachment claims to give items but nothing actually adds them to bag.

Discovery: trigger sign-in → mail with attachment → claimmail → observe `mail.claimed` event → Query bag → items not added → analyze event flow → `mail.claimed` has zero subscribers.

```go
// mail.go — no module subscribes to "mail.claimed"
func (ms *MailSystem) ClaimAttachment(mailID int) {
    // ... validation passes
    m.Claimed = true
    ms.bus.Publish(event.Event{
        Type: "mail.claimed",
        Data: map[string]any{
            "playerID": ms.playerID, "mailId": mailID,
            "itemID": m.Attachment.ItemID, "count": m.Attachment.Count,
        },
    })
    // Bug: Bag does not subscribe to mail.claimed
}
```

### B7: Sign-in day 7 reward ID conflicts with equipable item (L4 Hidden, Medium, signin.go)

Day 7 reward is itemID 3001, which is also the weapon equipable item ID. If B6 were fixed (Bag subscribes to mail.claimed and calls AddItem), claiming day 7 reward would trigger the full item.added → Equipment auto-equip → equip.success → Achievement chain.

Discovery: requires cross-module ID space analysis. Only discoverable by understanding that signin reward IDs overlap with equipment item IDs.

```go
// signin.go
var defaultRewards = map[int][2]int{
    // ...
    7: {3001, 1}, // 3001 is also equipableItems weapon ID
}
```

### Correlation Map (10 correlations)

| ID | Correlation | Related Bug |
|----|------------|-------------|
| R1 | Bag.item.added → Task.onItemAdded (item 2001→task 3001) | B2 |
| R2 | Bag.item.added → Task.onItemAdded (item 2002→task 3002) | B2 |
| R3 | Task.task.completed → Achievement.onTaskCompleted | B2 |
| R4 | Achievement internal: ≥2 unlocked → collector_100 | B3 |
| R5 | Bag.item.added → Equipment.onItemAdded (auto-equip) | B5 |
| R6 | Equipment.equip.success → Achievement.onEquipSuccess | — |
| R7 | SignIn.signin.claimed → Mail.onSignInClaimed | — |
| R8 | Achievement.achievement.unlocked → Mail.onAchievementUnlocked | — |
| R9 | Mail.mail.claimed → NO SUBSCRIBER (broken chain) | B6 |
| R10 | Task.task.completed (repeat) → duplicate Achievement unlock | B2 |

---

## 2. Agent Tool Chain & Knowledge Management

### 2.1 Tool Definitions

| Tool | Purpose | Mode Availability |
|------|---------|------------------|
| `send_command` | Interact with game server | all modes |
| `read_file` | Read source code file | C, D (code-available modes) |
| `search_code` | Search keyword in source | C, D (code-available modes) |
| `update_knowledge` | Overwrite knowledge.md | C, D (code-available modes) |

### 2.2 send_command enhancements

- Add `batch` sub-command: executes all pending operations, returns all logs (for ablation groups A, C)
- Existing `next` command unchanged (for ablation groups B, D)
- Existing query/inject commands unchanged

### 2.3 read_file tool

```
Input:  {"path": "internal/bag/bag.go"}
Output: file content as UTF-8 text
Restriction: path must be under internal/ or ai/ directories
Error: file not found or path outside allowed dirs
```

### 2.4 search_code tool

```
Input:  {"directory": "internal/", "pattern": "Subscribe"}
Output: matching lines (filename:line_number:content), max 50 matches
Implementation: strings.Contains (substring match, no regex)
```

### 2.5 update_knowledge tool

```
Input:  {"content": "full content of knowledge.md"}
Output: success confirmation
Behavior: full overwrite of knowledge.md
```

Agent reads current knowledge via `read_file("knowledge.md")`, modifies, writes back via `update_knowledge`.

### 2.6 Knowledge File

Location: `ai-integration-test-demo/knowledge.md`

Initial state: empty file created at agent startup.

Format: free-form markdown maintained by Agent. Example structure:
- Modules section (discovered module names, functions, events)
- Event Flow section (verified correlations with confidence)
- Potential Bugs section (observations with evidence)
- Notes section (anything else Agent wants to remember)

### 2.7 Agent Loop Changes

Current: system_prompt (with Code Summary injected) → LLM → send_command → ...
New: system_prompt (NO Code Summary) → LLM → multi-tool dispatch → ...

`handleToolCall` routes to:
- `send_command` → WebSocket (existing)
- `read_file` → os.ReadFile with path validation
- `search_code` → file walk + substring search
- `update_knowledge` → os.WriteFile to knowledge.md

Iteration limit: 60 → 80 (extra turns for code reading).

### 2.8 CodeAnalyzer New Role

- Retained but optional, controlled by `--quick-start` flag
- Default: codeanalyzer does NOT run, Agent explores code autonomously
- `--quick-start`: codeanalyzer runs and injects summary (backward compatibility)
- Code in `codeanalyzer/analyzer.go` unchanged

---

## 3. Ablation Matrix & Modes

### 3.1 2x2 Matrix

| Group | Code Access | Step | Tools | Scenario Name |
|-------|------------|------|-------|---------------|
| A | No | No | send_command (batch only, no playermgr) | `batch-only` |
| B | No | Yes | send_command (step mode, with playermgr) | `step-only` |
| C | Yes | No | read_file, search_code, update_knowledge, send_command (batch) | `code-batch` |
| D | Yes | Yes | all tools + send_command (step mode) | `dual` |

### 3.2 Tool availability per group

| Tool | A | B | C | D |
|------|---|---|---|---|
| send_command (query/playermgr) | No | Yes | No | Yes |
| send_command (inject) | Yes | Yes | Yes | Yes |
| send_command (next/step) | No | Yes | No | Yes |
| send_command (batch) | Yes | No | Yes | No |
| read_file | No | No | Yes | Yes |
| search_code | No | No | Yes | Yes |
| update_knowledge | No | No | Yes | Yes |

### 3.3 Scenario Routing

```go
// main.go
func getAgentMode(scenario string) string {
    switch scenario {
    case "batch-only":    return "batch-only"    // A
    case "step-only":     return "step-only"      // B
    case "code-batch":    return "code-batch"      // C
    case "dual":          return "dual"            // D
    case "autonomous-discovery": return "dual"     // backward compat
    default:              return "dual"
    }
}
```

### 3.4 Makefile Targets

```makefile
test-batch-only: build   # Group A
test-step-only: build    # Group B
test-code-batch: build   # Group C
test-dual: build         # Group D
test-discovery: test-dual # alias for backward compat
```

---

## 4. Prompt Design

### 4.1 Principle

Tell Agent WHAT tools it has, NOT what to test. No mentions of "edge cases", "negative counts", "duplicate claims". Protocol docs list command names only, no business semantics.

### 4.2 Four Group Prompts

**Group A (batch-only)**: Minimal. Only enqueue + batch commands listed. No query capability. No code access. Agent operates blind.

**Group B (step-only)**: Query + Inject + Step commands listed. No code access. Agent must infer everything from runtime.

**Group C (code-batch)**: read_file, search_code, update_knowledge tools described. send_command with batch only. Agent can pre-analyze but cannot observe step-by-step causality.

**Group D (dual)**: All tools. Full workflow suggestion (read code → update knowledge → step-by-step verify → update knowledge → report).

### 4.3 Prompt Level (0/1/2)

Implemented via flags, not hardcoded prompt variants:

| Level | Extra content | Flag |
|-------|--------------|------|
| Level 0 | None | default |
| Level 1 | Requirements doc | `--doc-file requirements.md` |
| Level 2 | Requirements doc + expert rules | `--doc-file` + `--rules-file` |

`prompt.BuildPrompt(mode string, opts PromptOptions)` takes options struct.

### 4.4 Report Format

All prompts require Agent to output a structured JSON block in final report:

```json
{
  "correlations": [{"id":"R1","source":"Bag","target":"Task","event":"item.added","confidence":"high"}],
  "bugs": [{"id":"B1","module":"bag","description":"...","severity":"Critical","evidence":"code+log"}],
  "false_positives": [],
  "iterations": 45,
  "files_read": 8,
  "steps_executed": 15
}
```

---

## 5. Server Changes

### 5.1 New `batch` command

```go
func (s *Server) handleBatch(req Request) Response {
    var allLogs []string
    for {
        logs := s.bp.Next()
        if len(logs) == 0 { break }
        allLogs = append(allLogs, logs...)
    }
    return Response{Ok: true, Log: allLogs}
}
```

### 5.2 Breakpoint Controller

No changes. Batch command just loops `Next()` until empty.

### 5.3 A/B mode restrictions

For groups A and B (no code access), `send_command` tool definition omits `playermgr` for group A. This is controlled by `tools.Definitions(mode)` which returns different tool sets per mode.

---

## 6. Evaluation Script

### 6.1 Ground Truth File

New file: `scripts/ground_truth.json`

Contains the canonical list of 10 correlations and 7 bugs with levels. Used by evaluation script for automated comparison.

### 6.2 summarize_results.py Rewrite

Parse JSON report blocks from agent output. Calculate:

| Metric | Formula |
|--------|---------|
| Correlation Recall | correct_correlations / total_correlations |
| Correlation Precision | correct_correlations / reported_correlations |
| Bug Recall | correct_bugs / total_bugs |
| Bug Precision | correct_bugs / reported_bugs |
| F1 (bugs) | 2 * P * R / (P + R) |
| Bug Level Score | {L1: found/1, L2: found/2, L3: found/2, L4: found/2} |
| False Positive Rate | false_bugs / reported_bugs |
| Exploration Efficiency | (correct_correlations + correct_bugs) / iterations |
| File Coverage | files_read / source_file_count |

### 6.3 Fallback

If JSON parsing fails, fall back to regex-based parsing (existing behavior) with a warning.

---

## 7. README Changes

### 7.1 Chapters to rewrite

- Chapter 4 (Implementation): new bug table, module interaction diagram, tool chain description, knowledge file mechanism
- Chapter 5 (Experiments): 2x2 ablation matrix, new metrics, ground truth design, expected results placeholders
- Chapter 6 (Discussion): metric analysis, level score discussion, honest autonomy assessment
- Appendix A: new run log example

### 7.2 Chapters to minimally edit

- Section 3.6 "Code Analysis Preprocessing" → rewrite as "Agent Autonomous Code Exploration"
- Section 3.7 "Agent Architecture" → update diagram to show multi-tool dispatch
- Section 3.8 "Self-Testing Loop" → add knowledge file maintenance step

### 7.3 Chapters to keep as-is

- Chapters 1-2 (Introduction, Related Work)
- Sections 3.1-3.5 (Core idea, breakpoint primitives, dual-channel theory, Prompt concept, communication)

---

## 8. File Change List

### New files

| File | Purpose |
|------|---------|
| `ai/knowledge/knowledge.go` | read_file, search_code, update_knowledge tool implementations |
| `scripts/ground_truth.json` | canonical correlation and bug list |
| `knowledge.md` | Agent-maintained knowledge file (created at runtime) |

### Modified files

| File | Changes |
|------|---------|
| `internal/bag/bag.go` | Keep B1 bug as-is |
| `internal/task/task.go` | Replace Bug #1 with B2: remove StateCompleted check in Progress |
| `internal/achievement/achievement.go` | Keep B3 bug as-is (wrong counter already exists) |
| `internal/signin/signin.go` | Replace Bug #3 with B4: ClaimReward without independent guard |
| `internal/equipment/equipment.go` | Keep B5 as-is (Equip already doesn't consume items) |
| `internal/mail/mail.go` | Keep B6 as-is (mail.claimed already has no subscriber) |
| `internal/server/server.go` | Add `batch` command handler |
| `internal/player/manager.go` | Adjust initial data for B7 (day 7 reward = 3001 already exists) |
| `ai/agent/agent.go` | Multi-tool dispatch, knowledge file init, iteration limit 80 |
| `ai/tools/tools.go` | Add read_file, search_code, update_knowledge tool definitions; mode-based filtering |
| `ai/prompt/system.go` | Rewrite all 4 group prompts + PromptOptions struct |
| `ai/codeanalyzer/analyzer.go` | No code changes |
| `cmd/server/main.go` | New scenarios, --quick-start flag, --doc-file, --rules-file flags, knowledge file init |
| `Makefile` | New targets: test-batch-only, test-step-only, test-code-batch, test-dual |
| `scripts/summarize_results.py` | Full rewrite with JSON parsing and new metrics |
| `README.md` | Rewrite chapters 4-6, appendix A; edit sections 3.6-3.8 |
| `go.mod` | No changes needed |

### Deleted behavior

- Default injection of Code Summary into system prompt (now requires --quick-start flag)
