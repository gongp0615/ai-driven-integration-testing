# AI-Driven Integration Testing for Game Servers via Deterministic State Machine Breakpoints

---

## Abstract

Large Language Models (LLMs) have dramatically improved code generation efficiency, yet system-level verification still relies heavily on manually written test cases. As development velocity increases and inter-module dependencies grow more complex, traditional approaches face structural bottlenecks. This paper proposes DSMB-Agent, an agent-driven integration testing method for complex systems that shifts the majority of system-level verification work from developers to agents through a three-phase process (Understand-Design-Execute). The method is built on two core mechanisms: (1) a Tree-Structured Command Space — exposing the system under test as a hierarchical structure of "module -> command -> parameter schema," where the agent can both use existing commands and autonomously register new commands within controlled boundaries to extend testing capabilities; and (2) Step-Wise Reasoning — using a breakpoint controller to separate command enqueuing from execution, allowing the agent to observe causal chains after each step and dynamically adjust its testing direction. On an event-driven game server prototype with 6 business modules and 7 seeded defects, we designed three autonomy levels (L0/L1/L2) and a 2x2 ablation study, conducting 3 repeated runs on each of two LLMs (GLM-5.1, GPT-5.4) for a total of 42 runs. Results show that: on GLM-5.1, the step-wise execution group found an average of 3.7/7 defects versus only 1.3/7 for the batch execution group; on GPT-5.4, the code+batch group performed best (5.0/7); in L0 mode, both models successfully registered complete test command sets from scratch and found 3-5 defects; L1 mode on GPT-5.4 achieved a single-run maximum of 6/7 defects, validating the cross-model feasibility of the approach.

**Keywords**: Integration Testing; LLM Agent; Command Tree; Step-Wise Reasoning; Autonomy Levels; Ablation Study

---

## 1 Introduction

### 1.1 Problem and Motivation

The efficiency gains of large language models in code generation, interface implementation, refactoring, and localized fixes have been transformative [1,2]. The direct consequence is that system functionality grows faster and module boundaries shift more frequently. However, the verification side has not kept pace — many teams still rely on manual code review and manually written integration test cases as their safety net. When AI multiplies development speed, manual review and hand-crafted test step enumeration increasingly struggle to match the new production tempo. The more fundamental issue is that the bottleneck in manual review lies not in review speed, but in the upper bound of cognitive load [3].

The traditional integration testing paradigm works as follows: developers first understand the business logic, then translate verification logic into a set of explicit test cases — specifying system entry points, writing operation sequences, selecting input parameters, enumerating boundary conditions, and defining expected outcomes. In systems with few modules and clear dependencies, this paradigm remains effective. However, in complex systems, the real risks often lie not in individual interface return values but in inter-module interactions: a single local state change may propagate along event, message, and callback chains to multiple subsystems. These interactions simultaneously exhibit three characteristics: long paths (three or more layers of event propagation), large state spaces, and hidden relationships (manifesting only at runtime). Even developers deeply familiar with the business logic can hardly enumerate all cross-module impact paths when writing test cases.

**When the complexity of inter-module interactions exceeds the upper bound of human enumeration capability, the failure of traditional methods is not accidental — it is structural.**

### 1.2 Core Idea

This paper proposes transforming system-level verification from "manually writing test steps" to "agent-driven autonomous exploration." Developers provide source code, requirement documents, and safety boundaries; the agent is responsible for understanding system structure, identifying module relationships, selecting exploration paths, generating verification commands, executing tests, and producing reports. The human role shifts from "test step author" to "rule provider and result reviewer."

To enable this transformation, this paper introduces two core designs:

- **Tree-Structured Command Space** — Borrowing the hierarchical approach of CLI design, the system under test is exposed as a command tree of `module -> command -> parameter schema`. The agent interacts with the tree via WebSocket + JSON, and can not only use existing commands but also autonomously register new commands at runtime through `register_cmd` to dynamically extend testing capabilities.
- **Step-Wise Reasoning** — A breakpoint controller separates command enqueuing from execution. After each step execution, the world is held (Hold World). During the world hold, the agent has full autonomous decision-making space: it can not only observe and reason, but also query state, modify data, register new interfaces, or read source code via commands, actively constructing preconditions for the next test step, and then deciding when to advance the world.

### 1.3 Contributions

The main contributions of this paper are:

1. Proposing a tree-structured command space and runtime command registration mechanism that enables agents to discover, use, and extend the testing interfaces of the system under test (Sections 3.1-3.2);
2. Proposing step-wise reasoning with a world hold mechanism that pauses the system world after each execution step, allowing the agent to make autonomous decisions in a quiescent state — querying, modifying, registering interfaces, or reading code — before deciding when to advance (Section 3.3);
3. Designing three autonomy levels (L0/L1/L2) and a 2x2 ablation study with 3 repeated runs on each of two LLMs, quantitatively evaluating the individual contributions of command registration and step-wise execution (Sections 4-5);
4. Across 42 runs on an event-driven system prototype, the GLM-5.1 step-wise group found an average of 3.7/7 defects (vs. batch group 1.3/7), GPT-5.4 code+batch group was optimal (5.0/7), and L1+LSP on GPT-5.4 achieved a single-run maximum of 6/7 (Section 5).

### 1.4 Paper Structure

Section 2 reviews related work; Section 3 details the method design; Section 4 describes the experimental setup; Section 5 reports experimental results; Section 6 discusses findings and implications; Section 7 analyzes threats to validity; Section 8 concludes the paper and outlines future work.

---

## 2 Related Work

### 2.1 Model-Based Testing (MBT)

Model-based testing (MBT) automatically generates test cases by constructing abstract models (state machines, UML models, etc.) of the system under test [4,5]. The core advantage of MBT is its ability to systematically derive path coverage from models, but its bottleneck lies in model construction itself: developers must manually maintain consistency between the model and implementation, and models typically struggle to capture runtime dynamic behavior. The command tree in this paper can be viewed as a lightweight operational model, but the key distinction is that the command tree is directly exposed from the running system without additional modeling, and the agent can dynamically extend this model at runtime.

### 2.2 Automated Exploratory Testing

Tools such as Monkey [6], Sapienz [7], and Stoat [8] in the mobile application testing domain automatically explore GUI state spaces through random or model-based strategies. These approaches align directionally with the agent exploration approach in this paper — both attempt to shift testing from predefined scripts to dynamic exploration. However, GUI exploratory testing primarily focuses on interface-level state coverage, whereas this paper focuses on server-side inter-module event propagation and state coordination, with different testing objectives and observability mechanisms.

### 2.3 LLMs for Software Testing

The application of LLMs in software testing has grown rapidly in recent years. ChatUniTest [9] uses LLMs to generate unit test cases; CodaMosa [10] combines LLMs with search algorithms to improve code coverage; TitanFuzz [11] uses LLMs for fuzz testing of deep learning libraries. These works primarily focus on automated generation of unit tests or API-level tests, whereas this paper focuses on cross-module behavioral verification at the integration testing level. Furthermore, in the above works, the LLM's role is "one-shot test code generation," while in this paper, the agent continuously interacts with the system under test at runtime, dynamically adjusting its testing strategy based on observations.

### 2.4 Fuzzing

Coverage-guided fuzzing (e.g., AFL [12], libFuzzer [13]) directs input generation by monitoring code coverage feedback. The step-wise reasoning in this paper has a conceptual connection with coverage-guided fuzzing in "adjusting exploration direction based on feedback," but two fundamental differences exist: (1) The feedback signal in fuzzing is code coverage, while in this paper it is event propagation chains and business logs — the latter carries semantic information, enabling the agent to perform causal reasoning rather than mere path coverage; (2) Fuzzing generates low-level byte sequences, while the agent in this paper operates on structured commands with business semantics.

### 2.5 Adaptive Testing and Online Testing

Adaptive testing strategies [14,15] dynamically select subsequent test cases based on the results of previously executed tests. The step-wise reasoning mechanism in this paper aligns with this direction, but adaptive testing typically selects from a predefined test pool, whereas the agent in this paper can generate entirely new test actions at runtime (including registering new commands), with an exploration space not limited by a predefined pool.

---

## 3 Method Design

This paper summarizes the agent-driven integration testing process into three iterable phases: **Understand -> Design -> Execute**. The two core mechanisms — tree-structured command space and step-wise reasoning — pervade all three phases.

### 3.1 Tree-Structured Command Space

#### 3.1.1 Structure Definition

The system under test is exposed as a structured command tree. The agent establishes a persistent connection with the system via WebSocket, with all interactions in JSON format. Each command follows a uniform three-level hierarchy:

```
cmd (module) -> action (command) -> params (parameter schema)
```

Using the experimental prototype as an example, the command tree structure is as follows:

```
+-- bag
|    +-- AddItem      {itemId: int, count: int}
|    +-- RemoveItem   {itemId: int, count: int}
|    +-- query        {itemId?: int}
+-- task
|    +-- query        {taskId?: int}
+-- equipment
|    +-- Equip        {slot: string, itemId: int}
|    +-- Unequip      {slot: string}
|    +-- query        {}
+-- signin
|    +-- CheckIn      {day: int}
|    +-- ClaimReward  {day: int}
|    +-- query        {day?: int}
+-- mail
|    +-- ClaimAttachment {mailId: int}
|    +-- query        {mailId?: int}
+-- system
|    +-- next         {}          // step-wise execution
|    +-- batch        {}          // batch execution
|    +-- help         {}          // query command tree
|    +-- register_cmd {name, target, action}
|    +-- listcmd      {}
+-- player
     +-- login        {playerId: int}
```

The corresponding JSON request format:

```json
{"cmd": "bag", "action": "AddItem", "params": {"itemId": 2001, "count": 1}}
```

The agent can query the complete command tree structure via `{"cmd": "system", "action": "help"}`, achieving runtime **discoverability**.

> **Implementation note:** The command tree above is a logical model. The experimental prototype uses a flattened command namespace for simplicity (e.g., `additem` instead of `bag.AddItem`), and players are created automatically at startup without an explicit `player.login` command. The mapping between the logical hierarchy and the flat implementation does not affect the generality of the method.

#### 3.1.2 Extension to Microservice Architectures

This structure naturally extends upward to a four-level hierarchy: **service -> module -> command -> parameters**. After obtaining the service list from a registry, the agent explores layer by layer:

```json
{"service": "game-server", "cmd": "bag", "action": "AddItem", "params": {...}}
```

### 3.2 Runtime Command Registration

If all test commands are predefined by humans, the agent cannot bridge the gap when "existing interfaces are insufficient to verify a suspicion." The `register_cmd` mechanism allows the agent to register new test commands within controlled boundaries:

**Registration workflow:**

1. The agent initiates a registration request:
   ```json
   {"cmd": "system", "action": "register_cmd",
    "params": {"name": "test_remove_negative", "target": "bag", "action": "RemoveItem"}}
   ```
2. The system performs safety validation — checking that the name does not conflict with built-in commands and that the target/action combination is on the allowed whitelist (`isAllowedRawBinding`);
3. After successful registration, the agent can invoke the underlying business method directly through the new command:
   ```json
   {"cmd": "test_remove_negative", "action": "exec", "params": {"itemId": 2001, "count": -1}}
   ```

**Safety constraints:** Each WebSocket connection maintains independent session state, with custom command registration isolated at the session level; the system enforces local loopback address binding (`127.0.0.1`), allowing access only from local processes.

The key value of this mechanism is that the built-in command layer typically includes parameter validation (e.g., `RemoveItem` rejects negative `count`), but the underlying business methods may not have equivalent protection. By registering raw interface commands, the agent can verify defects masked by the command layer.

### 3.3 Step-Wise Reasoning and World Hold

#### 3.3.1 Motivation

Batch execution mode submits all operations at once, and the agent can only observe the final accumulated state — essentially facing the same dilemma as manually pre-written test cases: all operations must be determined before execution, with no ability to adjust direction based on intermediate observations.

The key point of step-wise reasoning is not "finer granularity," but rather creating autonomous thinking and decision-making space for the agent through **Hold World**.

#### 3.3.2 Mechanism: Breakpoint Controller and World Hold

Based on a **Breakpoint Controller** that separates command enqueuing from execution:

1. When the agent sends a business command, the operation is enqueued to a pending execution queue, and **no changes occur in the system world**;
2. When the agent sends `{"cmd": "system", "action": "next"}`, the breakpoint controller dequeues and executes one operation, with all resulting event propagation and logs captured by the event bus;
3. `next` returns the **complete log chain** from this execution, after which **the world is held again**, awaiting the agent's next decision.

The key design here is **Hold World**: between when `next` returns its result and when the agent issues its next instruction, the system under test is in a completely quiescent state — no timers advance, no asynchronous events fire, no background state changes occur. This hold period is not simply a wait, but a window in which the agent has full autonomous decision-making authority. During this period, the agent can perform any combination of the following operations:

- **Observe and query** — Use `query` commands to inspect the current state of any module, confirming whether the side effects of the previous step match expectations;
- **Modify system state** — Directly inject data or modify state through commands (e.g., adding/removing items, triggering sign-in), artificially constructing specific preconditions, then observing subsequent behavior;
- **Register new commands** — Dynamically create new test interfaces via `register_cmd`, bypassing command-layer validation to directly invoke underlying methods;
- **Read source code** — Consult code implementation via `read_file` and `search_code` to verify whether runtime-observed behavior is consistent with code logic;
- **Update knowledge** — Record current findings via `update_knowledge` to avoid redundant exploration of known paths.

Only when the agent actively issues `next` does the world advance one step. This means the agent has complete control over "when to advance the world" — it can issue multiple query and modification commands after a single `next` to construct complex test scenarios before advancing to the next step to observe system reactions.

After each execution step, the agent goes through a complete reasoning cycle:

```
next (world advances one step)
  -> Observe: Which events were triggered? Is the log chain complete?
  -> Judge: Does the behavior match expectations? Any anomalies?
  -> [World Hold: Agent autonomous decision window]
      - Query — Check state changes in related modules
      - Modify — Inject data to construct specific preconditions
      - Register — Create new commands to reach unexposed interfaces
      - Read — Consult source code to verify runtime observations
  -> Enqueue next command
  -> next (world advances again)
```

In contrast, `{"cmd": "system", "action": "batch"}` executes all pending operations at once. In batch mode, the world continues running, and the agent loses not only observation granularity but also the ability to actively intervene in system state, construct test conditions, and adjust exploration direction between steps.

#### 3.3.3 Example: How World Hold Guides Exploration

> **Step 1** — Agent enqueues `bag.AddItem({itemId: 2001, count: 1})`, executes `next`
> Observed logs: `[Bag] add -> [Task] progress+1 -> [Task] completed -> [Achievement] unlocked`
> **[World Hold]** Agent reasons: "Task 3001 is completed. If I add the same item again, will the already-completed task be triggered to complete again?"
>
> **Step 2** — Agent decides to repeat the test, enqueues the same command, executes `next`
> Observed: `[Task] progress+1 (now 2/1) -> [Task] completed`
> **[World Hold]** Agent reasons: "The completed task was triggered again." -> Confirms defect B2
> Agent uses the hold period to execute `achievement.query` to check the achievement module state -> Discovers achievements were also re-unlocked -> Confirms the cascading impact of B2
>
> **Step 3** — Agent attempts `bag.RemoveItem({count: -1})`, rejected by the command layer
> **[World Hold]** Agent decides to register a new command: executes `register_cmd` to create `test_remove_negative`
> After successful registration, enqueues `test_remove_negative.exec({itemId: 2001, count: -1})`, executes `next`
> Observed: Item count changes from 1 to 2 -> Confirms defect B1
>
> Throughout this process, the agent made different types of decisions in each hold window: repeating tests to verify idempotency, querying related modules to trace impacts, and registering new commands to break through interface limitations. These decisions were not pre-planned but dynamically generated during the world hold based on observations.

#### 3.3.4 Actual Run Log: The Agent's Thinking and Decision Process

The following is an abridged terminal log from the experimental prototype, showing how the agent thinks, queries, registers commands, and discovers defects during step-wise execution. The `[Thinking]` lines are the LLM's reasoning text output after each iteration, revealing the autonomous decision-making process during world hold periods.

```
$ ./bin/server -mode test -scenario cross-module -api-key *** -model glm-5.1

2026/04/13 15:12:08 game server started on :5400
2026/04/13 15:12:08 client connected: 127.0.0.1:33748
2026/04/13 15:12:08 running AI test scenario: cross-module

======== Understand Phase: Query Initial State ========

--- iteration 1/80 ---
[Thinking] First, log in the player, then query the initial state of each module
    to understand the system's starting conditions.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS <- {"ok":true,"data":[]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS <- {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS <- {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"locked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

======== Execute Phase: Step-Wise Exploration ========

--- iteration 2/80 ---
[Thinking] Bag is empty, two active tasks: 3001 (target 1 item) and 3002 (target 2 items).
    All three achievements are locked. Now add item 2001 and observe whether it
    triggers task progress and achievement unlocking.

AI -> additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:1}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Bag] add item 2001 x1",
  "[Task] trigger 3001 progress+1 (now 1/1)",
  "[Task] task 3001 completed",
  "[Achievement] unlocked: first_task (id=4001)"
]}

--- iteration 3/80 ---   <- [World Hold: Agent Autonomous Decision Window]
[Thinking] Adding 1 item triggered a complete event chain:
    Bag.add -> Task.progress -> Task.completed -> Achievement.unlocked
    This is exactly the cross-module correlation via event propagation.
    The key question now: task 3001 is marked as completed. If I add the same item
    again, will the already-completed task be triggered to fire another completion
    event? This is an idempotency test.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:task TaskID:3001}
WS <- {"ok":true,"data":{"taskId":3001,"target":1,"progress":1,"state":"completed"}}

--- iteration 4/80 ---
[Thinking] Confirmed task 3001 state is completed, progress=1.
    Now repeat adding the same item to verify idempotency.

AI -> additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:1}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Bag] add item 2001 x1",
  "[Task] trigger 3001 progress+1 (now 2/1)",
  "[Task] task 3001 completed"
]}

--- iteration 5/80 ---   <- [World Hold]
[Thinking] WARNING: Serious issue! Task 3001 was already completed (progress 1/1),
    but after adding the item again, progress became 2/1 and the completed event
    fired again. The task completion state does not prevent re-triggering.
    This is an idempotency defect. I need to check whether the achievement module
    was also affected by the duplicate event.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS <- {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

[Thinking] first_task was not re-unlocked (still just unlocked), but the
    task.completed event was sent again. Recording defect: completed task
    re-triggering issue (B2).

AI -> update_knowledge (328 bytes)

======== Design Phase: Register New Command to Break Interface Limits ========

--- iteration 8/80 ---
[Thinking] Next, test boundary conditions for RemoveItem. What happens if a
    negative count is passed? This might expose missing parameter validation.

AI -> removeitem {Cmd:removeitem PlayerID:10001 ItemID:2001 Count:-1}
WS <- {"ok":false,"error":"invalid count: must be positive"}

--- iteration 9/80 ---   <- [World Hold]
[Thinking] The command layer rejected the negative count. But this only proves the
    command layer has validation. Does the underlying RemoveItem business method
    have the same protection? If command-layer validation is bypassed (e.g., via
    an internal call), can the underlying method correctly handle negative values?
    I need to register a command that directly invokes the underlying method to
    verify this hypothesis.

AI -> register_cmd name=test_remove_negative target=bag action=RemoveItem
WS <- {"ok":true,"registered":"test_remove_negative"}

--- iteration 10/80 ---
[Thinking] Registration successful. Now invoke the underlying RemoveItem directly
    with count=-1 through this raw interface command, bypassing command-layer
    validation.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag ItemID:2001}
WS <- {"ok":true,"data":{"itemId":2001,"count":2}}

AI -> test_remove_negative {Cmd:test_remove_negative PlayerID:10001 ItemID:2001 Count:-1}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Bag] remove item 2001 x-1"
]}

--- iteration 11/80 ---   <- [World Hold]
AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag ItemID:2001}
WS <- {"ok":true,"data":{"itemId":2001,"count":3}}

[Thinking] CRITICAL DEFECT! Item count changed from 2 to 3.
    RemoveItem(-1) actually increased the item count -- the underlying method
    simply performs count -= (-1) = count + 1.
    The command layer blocks negative values, but the underlying business method
    has no protection whatsoever. If any internal call path bypasses the command
    layer, this vulnerability will be triggered.
    Recording defect: RemoveItem lacks count<=0 validation (B1).

AI -> update_knowledge (412 bytes)
```

The log above demonstrates three key characteristics of the DSMB-Agent workflow:

1. **`[Thinking]` output reveals the reasoning process during world hold** -- The agent does not mechanically execute predefined steps, but analyzes causal chains, formulates hypotheses, and designs the next verification action after each step;
2. **Dynamic query-observe-decide loop** -- After discovering task re-triggering (iteration 5), the agent proactively queries the achievement module to trace cascading impacts. This decision was entirely dynamically generated based on runtime observations;
3. **Command registration breaks through interface blind spots** -- When the command layer rejects a negative parameter (iterations 8-9), the agent does not give up but decides during the hold window to register a raw interface command, ultimately discovering an underlying defect masked by the command layer.

#### 3.3.5 Implementation Considerations for the Step-Wise Mechanism

Step-wise execution requires inserting pause points at the boundaries of "the serialized execution process of a single business operation." The key constraint here is: **step does not insert write operations within a serialized execution process** — it only holds the world after a complete synchronous execution chain has finished. RPC calls, asynchronous messages, and other suspension behaviors are not part of serialization.

In production servers, the step-wise mechanism needs to adapt to the underlying actor model. Whether explicit actors (e.g., Skynet) or implicit actors (e.g., Go goroutine + channel), server-side concurrency units typically follow the "exclusive state + message-driven" pattern, which naturally aligns with the hold semantics of the step-wise mechanism:

**Skynet (explicit actor):** The concurrency units that the AI needs to intervene in are individual services. Each service has independent state and a message queue; regardless of which worker\_thread drives it, execution continues within the same logical workspace. The breakpoint controller inserts pause points at the service-level message processing boundary — pausing after one message is fully processed (including all synchronous callbacks it triggers), waiting for the agent's decision before delivering the next message. When an RPC is initiated during message processing (via `skynet.call` in Skynet, which suspends the current coroutine), that suspension point is also a valid pause boundary — the service's state is consistent at that moment, and step can safely intervene.

**Go server (implicit actor):** In Go, there is no fixed mapping between goroutines and logical entities — the same business flow may span multiple goroutines, and the same goroutine may serve different requests sequentially. However, when the system design revolves around "state ownership + message flow" (each logical entity receives instructions through a dedicated channel), it is essentially an implicit actor model. The breakpoint controller controls the timing of channel message delivery, not goroutine scheduling. The experimental prototype in this paper adopts this approach — using WebSocket message queues and the breakpoint controller to separate command enqueuing from execution, with goroutine scheduling being transparent to the step-wise mechanism.

**General principle:** Regardless of the specific form of the underlying actor model, the implementation focus of the step-wise mechanism is the same: (1) identify the serialization boundaries of actor message processing (including synchronous execution completion and yield points such as RPC suspension); (2) insert pause points at these boundaries; (3) ensure that during the hold period, actor state is completely quiescent and no new messages are received.

### 3.4 Three-Phase Process

**Understand phase:** The agent's goal is to build a structured understanding of system module relationships and event flows. In real engineering projects, source code size typically far exceeds the agent's context window limit, making file-by-file reading infeasible. Therefore, the understand phase should prioritize **semantic-level tools** over text-level search:

- **LSP (Language Server Protocol)** — The agent obtains precise semantic information through a locally deployed language server: `go to definition` to locate symbol implementations, `find references` to trace event publish/subscribe relationships, `workspace/symbol` to obtain module-level symbol indices, and `call hierarchy` to analyze call chains. LSP provides structured results from compiler-level semantic analysis, with precision far exceeding text search.
- **Static AST analysis** — Optionally, code analyzers can pre-generate module structure summaries (event publish/subscribe tables, function signatures, cross-module dependency graphs) as the agent's initial knowledge.
- **Text search** — `read_file` and `search_code` serve as supplementary tools for scenarios not easily covered by LSP (e.g., comments, configuration files, string constants).

The experimental prototype in this paper used text-level tools directly due to its small module scale (6 files). However, in real projects, LSP is the core tool for the understand phase — it enables the agent to precisely locate key module relationships and event propagation paths without reading all source code.

**Design phase:** When existing commands are insufficient to verify a suspicion, the agent registers new test commands within the allowed rules. This phase bridges the gap of "identifying something suspicious from the code but being unable to construct triggering input."

**Execute phase:** The agent explores step by step through step-wise execution, continuously refining understanding and actions during runtime: enqueue operations -> step-wise execution -> causal tracing -> path expansion -> knowledge update. The three phases can iterate repeatedly.

---

## 4 Experimental Setup

### 4.1 Experimental Platform

The experimental prototype is an event-driven game server implemented in Go, containing 6 business modules and 2 infrastructure modules. This platform was chosen because event-driven systems naturally exhibit characteristics typical of complex systems: dense inter-module relationships, frequent state changes, and business actions triggering cascading side effects.

| Module | Responsibility | Key Events |
|--------|---------------|------------|
| Bag | Item add/remove management | Publishes `item.added`, `item.removed` |
| Task | Task progress tracking | Subscribes to `item.added`, publishes `task.completed` |
| Achievement | Achievement unlocking | Subscribes to `task.completed`, `item.added`, `equip.success`; publishes `achievement.unlocked` |
| Equipment | Equipment equipping | Subscribes to `item.added`, publishes `equip.success` |
| SignIn | Daily sign-in rewards | Publishes `signin.claimed` |
| Mail | Mail and attachments | Subscribes to `achievement.unlocked`, `signin.claimed`, publishes `mail.claimed` |
| Event Bus | Event bus | Global event routing and log capture |
| Breakpoint Controller | Breakpoint controller | Operation enqueuing, step-wise execution, log collection |

### 4.2 Seeded Defects

Seven defects were seeded in the prototype across 4 difficulty levels. Defect design referenced common event-driven system defect patterns [16], including missing state guards, lack of idempotency, broken event chains, and cross-module ID conflicts:

| ID | Module | Difficulty | Severity | Description |
|----|--------|-----------|----------|-------------|
| B1 | Bag | L1 | Critical | `RemoveItem` lacks `count<=0` validation; negative values cause item increase |
| B2 | Task | L2 | High | Completed tasks can still be triggered to fire completion events again |
| B3 | Achievement | L2 | Medium | `collector_100` counts wrong objects (counts achievements instead of distinct item types) |
| B4 | SignIn | L3 | High | `ClaimReward` has no idempotency guard; rewards can be claimed infinitely |
| B5 | Equipment | L3 | Critical | Equipping does not remove item from bag; item exists in both locations simultaneously |
| B6 | Mail | L4 | High | `mail.claimed` event has no subscribers; mail attachments can never reach the bag |
| B7 | SignIn+Equipment | L4 | Medium | Day 7 sign-in reward ID conflicts with equippable weapon ID, triggering unexpected equipment chain |

### 4.3 Autonomy Levels

Three autonomy levels were designed to evaluate the effectiveness of different capability combinations:

**L0 (Full Autonomy):** The agent receives no pre-built business commands, possessing only infrastructure commands (`player.login`, `system.next`, `system.batch`, `system.help`) and command registration capability. The agent must read source code to register all test commands from scratch.

**L1 (Semi-Autonomy):** The agent has access to all pre-built business commands while retaining command registration capability. When built-in commands are insufficient to cover a testing path, new commands can be registered as supplements.

**L2 (Assisted Execution):** All business commands are pre-built by humans, and the agent has no command registration capability. Within L2, a 2x2 ablation study distinguishes the individual contributions of code understanding and step-wise execution:

| Group | Code Access | Step-Wise Execution | Description |
|-------|------------|-------------------|-------------|
| A (batch-only) | No | No | Lowest autonomy baseline |
| B (step-only) | No | Yes | Validates independent contribution of step-wise execution |
| C (code-batch) | Yes | No | Validates independent contribution of code understanding |
| D (dual) | Yes | Yes | Dual-factor combination |

Additionally, a **code-only** baseline was included for pure static analysis, where the agent produces reports solely by reading source code without any runtime interaction.

### 4.4 Experimental Configuration

Experiments were conducted on two LLMs to evaluate the method's dependence on the underlying model:

| Parameter | GLM-5.1 | GPT-5.4 |
|-----------|---------|---------|
| Provider | Zhipu AI (Anthropic-compatible API) | OpenAI (Codex CLI) |
| Max iterations | 110 | 80 |
| L2 runs per group | 3 | 3 |
| L0 runs | 3 | 3 |
| L1 runs | 3 | 3 |
| Connection method | WebSocket | WebSocket |
| Code understanding tools | read_file + search_code | read_file + search_code + LSP (gopls) |

### 4.5 Evaluation Metrics

| Metric | Definition |
|--------|-----------|
| Bug Recall | Number of true defects found / 7 |
| Bug Precision | Number of true defects / total reported defects |

---

## 5 Experimental Results

### 5.1 L2 Ablation Study (GLM-5.1, 3 Runs)

Table 1 reports the defect detection matrix for the four L2 experimental groups and the code-only baseline on GLM-5.1:

**Table 1: L2 Defect Detection Matrix (GLM-5.1, 3 Runs)**

| Defect | Difficulty | A (batch-only) | B (step) | C (code-batch) | D (dual) | code-only |
|--------|-----------|:--------------:|:--------:|:---------------:|:--------:|:---------:|
| B1 | L1 | . | . | 2/3 | . | 1/3 |
| B2 | L2 | 1/3 | . | 1/3 | 1/3 | . |
| B3 | L2 | . | 2/3 | 1/3 | . | 3/3 |
| B4 | L3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| B5 | L3 | . | 3/3 | 1/3 | 1/3 | 2/3 |
| B6 | L4 | . | 3/3 | 3/3 | 3/3 | 3/3 |
| B7 | L4 | . | . | . | . | . |

**Table 2: L2 Summary Metrics (GLM-5.1, Mean of 3 Runs per Group)**

| Metric | A (batch-only) | B (step) | C (code-batch) | D (dual) | code-only |
|--------|:--------------:|:--------:|:---------------:|:--------:|:---------:|
| Avg. defects found | 1.3 | 3.7 | 3.7 | 2.7 | 4.0 |
| Avg. iterations | 55 | 87 | 72 | 93 | 25 |
| B1 detection rate | 0% | 0% | 67% | 0% | 33% |

### 5.2 L2 Ablation Study (GPT-5.4, 3 Runs)

To verify whether the findings on GLM-5.1 hold across models, the L2 ablation study was replicated on GPT-5.4 with 3 runs per group. The GPT-5.4 experiments also integrated LSP (gopls) semantic code analysis tools.

**Table 3: L2 Defect Detection Matrix (GPT-5.4, 3 Runs)**

| Defect | Difficulty | A (batch-only) | B (step) | C (code-batch) | D (dual) | code-only |
|--------|-----------|:--------------:|:--------:|:---------------:|:--------:|:---------:|
| B1 | L1 | . | . | . | . | 2/3 |
| B2 | L2 | . | 2/3 | 3/3 | 3/3 | 2/3 |
| B3 | L2 | 3/3 | 3/3 | 3/3 | 3/3 | 2/3 |
| B4 | L3 | . | . | 3/3 | 3/3 | 2/3 |
| B5 | L3 | 1/3 | 1/3 | 2/3 | . | 2/3 |
| B6 | L4 | . | . | 3/3 | 3/3 | 2/3 |
| B7 | L4 | 1/3 | . | 1/3 | . | . |

**Table 4: L2 Summary Metrics (GPT-5.4, Mean of 3 Runs per Group)**

| Metric | A (batch-only) | B (step) | C (code-batch) | D (dual) | code-only |
|--------|:--------------:|:--------:|:---------------:|:--------:|:---------:|
| Avg. defects found | 1.7 | 2.0 | 5.0 | 4.0 | 4.0 |
| Avg. iterations | 6 | 54 | 31 | 61 | -- |
| B1 detection rate | 0% | 0% | 0% | 0% | 67% |

#### Key Findings

**Finding 1: Step-wise execution is highly effective on GLM-5.1, while code understanding is highly effective on GPT-5.4.** On GLM-5.1, Group B (step-only, 3.7/7) and Group C (code-batch, 3.7/7) tied for best performance, both significantly outperforming Group A (batch-only, 1.3/7). On GPT-5.4, Group C (code-batch, 5.0/7) performed strongest. This indicates that step-wise execution and code understanding each have independent contributions, but the marginal benefit of code understanding grows with model capability.

**Finding 2: B1 is difficult to detect reliably in runtime groups.** B1 (`RemoveItem` with negative parameter) was not found in any of GPT-5.4's L2 runtime groups, and on GLM-5.1 was only sporadically detected in the code-batch group (2/3). Only modes with command registration capability (L0/L1) could reliably reproduce B1. This validates the necessity of the command registration mechanism for breaking through interface blind spots.

**Finding 3: Both models exhibit negative interaction effects.** On GLM-5.1, Group D (dual, 2.7/7) performed below both Group B and Group C (both 3.7/7); on GPT-5.4, Group D (4.0/7) similarly fell below Group C (5.0/7). Analysis: simultaneously enabling code analysis and step-wise execution causes the agent to switch between two modes, increasing cognitive load and iteration consumption, ultimately underperforming compared to focusing on a single factor.

**Finding 4: code-only has the highest cross-model stability.** code-only achieved approximately 4/7 defect detection rate on both models and had the highest probability of detecting B1 among L2 groups (1/3 on GLM-5.1, combined with code-batch's 2/3). However, its inherent limitation is the inability to rule out false positives through runtime verification.

**Finding 5: The most easily detected defects differ between models.** On GLM-5.1, B4 (sign-in reward with no idempotency guard) was found in all 15 L2 runs, and B6 (mail attachment chain break) was found in 12/15 runs. On GPT-5.4, B3 (achievement counting wrong objects) was the most stable (14/15 detections), while B4 and B6 were primarily found in groups with code access (8/15 each). The common trait is that these defects exhibit behavioral anomalies highly visible in logs or code, identifiable without deep reasoning.

### 5.3 Cross-Model Comparison

Combining results from both models (all 3-run means):

**Table 5: Cross-Model L2 Ablation Comparison**

| Group | GLM-5.1 | GPT-5.4 | Trend |
|-------|:-------:|:-------:|-------|
| A (batch-only) | 1.3 | 1.7 | Consistent: weakest |
| B (step) | 3.7 | 2.0 | GLM superior |
| C (code-batch) | 3.7 | 5.0 | GPT superior |
| D (dual) | 2.7 | 4.0 | GPT superior |
| code-only | 4.0 | 4.0 | Consistent |

**Core insight:** Step-wise execution and code understanding each have independent contributions, but the relative magnitude of their effects depends on model capability. On GLM-5.1, the marginal benefit of step-wise execution is more pronounced (Group B 3.7 vs Group A 1.3), while on GPT-5.4, code understanding significantly boosts batch mode performance (Group C 5.0 vs Group A 1.7). Negative interaction effects were observed on both models (Group D underperforms single-factor groups), indicating that simultaneously enabling both capabilities faces cognitive load issues under the current agent architecture. This implies that as LLM capabilities improve, optimal strategies need to adjust the weight allocation between code understanding and runtime exploration based on model characteristics.

### 5.4 L0 Experimental Results

In L0 mode, the agent starts from scratch with no pre-built business commands.

**Table 6: L0 Experimental Results**

| Metric | GLM run1 | GLM run2 | GLM run3 | GPT run1 | GPT run2 | GPT run3 |
|--------|:--------:|:--------:|:--------:|:--------:|:--------:|:--------:|
| Iterations | 102 | 100 | 108 | 80 | 70 | 71 |
| Registered commands | 7 | 7 | 7 | 7 | 7 | 7 |
| Defects found | B1,B3,B4,B6 | B1,B3,B4,B6 | B1,B4,B6 | B1,B3,B4,B5,B6 | B1,B3,B4,B5 | B1,B3,B4,B6 |
| Defect count | 4/7 | 4/7 | 3/7 | 5/7 | 4/7 | 4/7 |

GLM-5.1 found an average of 3.7/7, GPT-5.4 found an average of 4.3/7. Both models successfully registered a complete test command set (7 commands) and found B1 across all 6 L0 runs — a defect that was virtually undetectable by L2 groups on both models. B4 and B6 were reliably detected in all runs. The L0 results validate the cross-model feasibility of the full autonomy mode.

### 5.5 L1 Experimental Results

**Table 7: L1 Experimental Results**

| Metric | GLM run1 | GLM run2 | GLM run3 | GPT run1 | GPT run2 | GPT run3 |
|--------|:--------:|:--------:|:--------:|:--------:|:--------:|:--------:|
| Iterations | 105 | 108 | 107 | 66 | 87 | 75 |
| Registered commands | 2 | 2 | 2 | 2 | 1 | 1 |
| Defects found | B3,B4,B5,B6 | B1,B3,B4,B6 | B4,B5 | B1,B3,B4,B5,B6 | B1,B3,B4,B5,B6,B7 | B1,B3,B4,B5,B6 |
| Defect count | 4/7 | 4/7 | 2/7 | 5/7 | 6/7 | 5/7 |

The core validation objective of L1 mode was the "interface gap" hypothesis. In all 6 runs, the agent autonomously registered raw interface commands to bypass command-layer validation. In GLM-5.1 run2, after registering `test_remove_negative`, item count was observed to change from 3 to 104; in GPT-5.4 run2, after registering the same command, item count changed from 1 to 2 — **both models successfully reproduced defect B1 through command registration**.

GLM-5.1 found an average of 3.3/7, GPT-5.4 found an average of 5.3/7. GPT-5.4's L1 performance significantly exceeded GLM-5.1, because GPT-5.4 combined with LSP tools could more efficiently understand code structure, thereby covering more modules within the limited iteration budget. Notably, GPT-5.4 run2 found 6/7 defects (including B7), which is the highest single-run record across all experiments. B4 and B6 were reliably detected in all 6 runs across both models.

---

## 6 Discussion

### 6.1 Command Registration Capability Is Key to Breaking Through Interface Blind Spots

The detection rate for B1 in L2 runtime groups was extremely low (GLM-5.1 only code-batch 2/3, GPT-5.4 all 0/3), while L0/L1 could reliably reproduce this defect through command registration. This contrast demonstrates that when testing interfaces cannot construct the inputs needed to trigger a defect, no amount of exploration strategy can compensate for the interface's limitations. The command registration mechanism elevates the agent from "command user" to "command designer," and is the core differentiator between this method and traditional automated testing.

### 6.2 World Hold Grants the Agent the Ability to Actively Intervene and Choose Testing Directions

The advantage of step-wise execution over batch execution lies not merely in finer granularity or causal traceability, but fundamentally in **Hold World** creating autonomous decision-making space for the agent. After each execution step, the world is completely quiescent, and the agent can freely query state, modify data, and register new interfaces — actively constructing test conditions rather than passively awaiting results. Under batch execution, the agent must determine all operations before execution; the world runs continuously, and the agent loses not only observation granularity but also the ability to actively intervene in the system, construct preconditions, and adjust exploration direction between steps. The experimental data on GLM-5.1 directly demonstrates this: Group B (step, 3.7/7) significantly outperformed Group A (batch, 1.3/7), and Group B reliably found B4, B5, and B6 across all 3 runs — the agent decided during hold periods to query related modules to trace cascading effects, repeat operations to verify idempotency, and actively construct boundary conditions. These decisions were dynamically generated during world hold windows, not pre-planned.

### 6.3 The Effectiveness of Code Understanding Depends on Model Capability

On GLM-5.1, Group D (2.7/7) performed below both Group B and Group C (both 3.7/7); on GPT-5.4, Group C (code-batch, 5.0/7) significantly outperformed Group B (step-only, 2.0/7). This cross-model comparison reveals an important insight: the marginal benefit of code understanding is not fixed but grows with the model's code comprehension capability. The negative interaction effects observed on both models (Group D underperforming single-factor groups) partly stem from simultaneously enabling code analysis and step-wise execution increasing the agent's cognitive load — switching between code reading and runtime exploration consumed additional iterations. When the model's code understanding capability is stronger (GPT-5.4) or tools are more efficient (LSP semantic navigation replacing text search), the return on investment for the code understanding phase improves significantly.

The implication for engineering practice is that as LLM code understanding capabilities continue to improve, the "code + runtime" combination will gradually become the optimal strategy, and the relative advantage of "pure runtime" mode will diminish. This paper also integrated LSP (gopls) semantic code analysis tools in the GPT-5.4 experiments, enabling the agent to obtain precise cross-module reference relationships through calls like `lsp_references`, `lsp_definition`, and `lsp_symbols` — a single `lsp_references("Publish")` call returns all 11 event publication points, replacing multiple `search_code` + `read_file` combinations. In larger real-world projects, the advantages of LSP will be further amplified.

### 6.4 Complementarity of Static Analysis and Runtime Testing

The code-only baseline reliably found approximately 4/7 defects on both models and was the only approach to find B7 (ID conflict) — indicating that static analysis excels at generating "suspicions." However, static analysis cannot rule out false positives through runtime verification, while runtime testing can provide confirmatory evidence. The two are complementary, not substitutive. A potential improvement direction is using static analysis as a priority ranker for runtime testing — the agent first marks suspicious areas through code analysis, then concentrates the limited runtime budget on high-priority areas.

### 6.5 Implications for Engineering Practice

The method proposed in this paper points to a new engineering division of labor: developers provide the system, rules, and constraint boundaries; the agent is responsible for continuously understanding the system, exploring test paths, registering missing verification interfaces, executing tests, and producing reports; humans intervene only in rule setting and reviewing high-risk conclusions. From this perspective, integration testing should not merely be about "executing old tests faster," but should gradually transition toward "letting agents assume the majority of system-level verification labor."

---

## 7 Threats to Validity

### 7.1 Internal Validity

**Number of runs and statistical power:** The L2 ablation, L0, and L1 experiments on both models each completed 3 runs per group (42 total runs). Three repetitions are sufficient to observe stability trends (e.g., all L0/L1 runs on both models found B1 and B4), but are insufficient for rigorous statistical testing (e.g., Wilcoxon rank-sum test). Future work should scale to 10+ runs to improve statistical power.

**Representativeness of seeded defects:** The 7 defects were manually designed and seeded by the authors and may not fully represent defect distributions in real systems. Defect design referenced common event-driven system defect patterns, but future work should consider introducing mutation testing methodology to increase objectivity.

**Impact of iteration budgets:** The maximum iteration count was set to 110 for GLM-5.1 and 80 for GPT-5.4, with different groups consuming different actual iterations (25-108), which may affect fair comparison of each group's capability ceiling.

### 7.2 External Validity

**Scale of the system under test:** Experiments were conducted only on a prototype system with 6 modules; whether conclusions generalize to larger, more complex systems remains to be validated.

**Domain generalizability:** Although event-driven architectures are widely found in microservices, IoT, and gaming, the experimental platform was a game server; applicability to other domains requires additional verification.

**Model dependency:** Experiments used two models, GLM-5.1 and GPT-5.4, and already observed significant model capability effects on results (e.g., the code understanding main effect reversed direction across models). However, two models are insufficient to establish universal conclusions; future work should expand to more models.

### 7.3 Construct Validity

**Defect adjudication criteria:** Some findings in agent reports required human judgment on whether they constitute real defects. The consistency of adjudication criteria may affect the accuracy of precision rates.

---

## 8 Conclusion and Future Work

### 8.1 Conclusion

This paper proposes DSMB-Agent, an agent-driven integration testing method based on a tree-structured command space and step-wise reasoning. Through experimental validation on two LLMs (GLM-5.1, GPT-5.4), the following conclusions are drawn:

1. Both step-wise execution and code understanding have independent contributions — on GLM-5.1, the step and code+batch groups both achieved 3.7/7 (vs. batch group 1.3/7); on GPT-5.4, the code+batch group was optimal (5.0/7);
2. The effectiveness of code understanding is highly dependent on model capability — on GLM-5.1, step-wise execution showed a more pronounced marginal benefit (Group B 3.7 vs Group A 1.3), while on GPT-5.4, code understanding boosted batch mode from 1.7 to 5.0;
3. Negative interaction effects were observed on both models — simultaneously enabling code analysis and step-wise execution actually reduced effectiveness, indicating a cognitive load ceiling under the current agent architecture;
4. B1 was a shared blind spot across all L2 groups on both models — command registration capability (L0/L1) is key to breaking through interface blind spots;
5. L0 (full autonomy) mode on both models successfully built complete test command sets from scratch and found 3-5 defects; L1 mode on GPT-5.4 averaged 5.3/7 (maximum 6/7 in a single run), validating cross-model feasibility;
6. Static analysis and runtime testing are complementary — code-only reliably found approximately 4/7 defects on both models; the L1+LSP combination on GPT-5.4 achieved comparable detection rates with runtime verification capability.

### 8.2 Future Work

- **Scale up repeated experiments** — Current 3 runs per group are sufficient to observe trends, but need to scale to 10+ to support rigorous statistical testing;
- **Deep LSP integration** — This paper has implemented an LSP (gopls) integration prototype; future work should systematically evaluate the quantitative impact of LSP semantic navigation on iteration efficiency and defect detection rates;
- **Larger-scale systems** — Validate the method's scalability on systems with more modules and more complex relationships;
- **Quantitative comparison with existing methods** — Conduct controlled experiments comparing against MBT, random testing, and other baseline methods.

---

## References

[1] M. Chen et al., "Evaluating Large Language Models Trained on Code," arXiv:2107.03374, 2021.

[2] S. Peng et al., "The Impact of AI on Developer Productivity: Evidence from GitHub Copilot," arXiv:2302.06590, 2023.

[3] M. E. Fagan, "Design and Code Inspections to Reduce Errors in Program Development," IBM Systems Journal, vol. 15, no. 3, pp. 182-211, 1976.

[4] M. Utting, A. Pretschner, and B. Legeard, "A Taxonomy of Model-Based Testing Approaches," Software Testing, Verification and Reliability, vol. 22, no. 5, pp. 297-312, 2012.

[5] W. Grieskamp, "Multi-Paradigmatic Model-Based Testing," in Proc. FATES/RV, 2006, pp. 1-19.

[6] Google, "UI/Application Exerciser Monkey," Android Developers Documentation.

[7] K. Mao, M. Harman, and Y. Jia, "Sapienz: Multi-objective Automated Testing for Android Applications," in Proc. ISSTA, 2016, pp. 94-105.

[8] T. Su et al., "Guided, Stochastic Model-Based GUI Testing of Android Apps," in Proc. ESEC/FSE, 2017, pp. 245-256.

[9] Z. Chen et al., "ChatUniTest: A Framework for LLM-Based Test Generation," arXiv:2305.04764, 2023.

[10] C. Lemieux et al., "CodaMosa: Escaping Coverage Plateaus in Test Generation with Pre-trained Large Language Models," in Proc. ICSE, 2023, pp. 919-931.

[11] Y. Deng et al., "Large Language Models Are Zero-Shot Fuzzers: Fuzzing Deep-Learning Libraries via Large Language Models," in Proc. ISSTA, 2023, pp. 1165-1176.

[12] M. Zalewski, "American Fuzzy Lop (AFL)," https://lcamtuf.coredump.cx/afl/.

[13] LLVM Project, "libFuzzer - A Library for Coverage-Guided Fuzz Testing," https://llvm.org/docs/LibFuzzer.html.

[14] W. Afzal, R. Torkar, and R. Feldt, "A Systematic Review of Search-Based Testing for Non-Functional System Properties," Information and Software Technology, vol. 51, no. 6, pp. 957-976, 2009.

[15] A. Arcuri and L. Briand, "Adaptive Random Testing: An Illusion of Effectiveness?" in Proc. ISSTA, 2011, pp. 265-275.

[16] X. Zhou et al., "Fault Analysis and Debugging of Microservice Systems: Industrial Survey, Benchmark System, and Empirical Study," IEEE Trans. Software Eng., vol. 47, no. 2, pp. 243-260, 2021.

---

## Appendix

### A. Experimental Prototype Source Code

The prototype code is located in the `ai-integration-test-demo/` directory, containing the complete server implementation, agent framework, and experiment scripts.

### B. Experiment Execution Instructions

See `QUICKSTART.md`.

### C. Raw Experimental Data

- GLM-5.1 complete experiment data (7 groups x 3 rounds): `results/formal/glm-5.1/`
- GPT-5.4 complete experiment data (7 groups x 3 rounds): `results/formal/gpt-5.4/`
- GLM-5.1 early single-run reports: `results/evaluation_report.md`, `results/l0-run1-report.md`, `results/l1-run1-report.md`

### D. Complete Terminal Log Example

The following is a complete terminal output from a step-only mode experiment (GLM-5.1, 54 iterations, interrupted at iteration 55 due to API server error). The `[Thinking]` lines show the agent's full reasoning process during world hold periods, while `AI ->` and `WS <-` represent commands sent by the agent and responses from the system, respectively.

```
$ ./bin/server -mode test -scenario step-only \
    -api-key *** -model glm-5.1 \
    -base-url https://open.bigmodel.cn/api/anthropic

2026/04/17 16:45:02 game server started on 127.0.0.1:5400
2026/04/17 16:45:03 client connected: 127.0.0.1:50206
2026/04/17 16:45:03 using Anthropic API provider (model=glm-5.1)
2026/04/17 16:45:03 running AI test scenario: step-only (mode: step-only)

========== Understand Phase: Query Initial State ==========

--- iteration 1/80 ---
[Thinking] I'll systematically test all modules for player 10001,
    stepping through each operation to observe logs and cross-module
    interactions. Let me start by querying the current state of all modules.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS <- {"ok":true,"data":[]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS <- {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS <- {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"locked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"},
  {"achId":4004,"name":"fully_equipped","state":"locked"}
]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:equipment}
WS <- {"ok":true,"data":{}}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:signin}
WS <- {"ok":true,"data":[
  {"day":1,"rewardItem":2001,"rewardCount":1,"claimed":false},
  {"day":2,"rewardItem":2002,"rewardCount":1,"claimed":false},
  ...
  {"day":7,"rewardItem":3001,"rewardCount":1,"claimed":false}
]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:mail}
WS <- {"ok":true,"data":[]}

========== Execute Phase: Step-Wise Exploration ==========

--- iteration 2/80 ---
[Thinking] Good, I have the initial state. Let me summarize:
    - Bag: Empty
    - Tasks: 3001 (target:1), 3002 (target:2), both active
    - Achievements: 4 locked (first_task, task_master, collector_100, fully_equipped)
    - Equipment: Empty
    - Sign-in: 7 days available, none claimed
    - Mail: Empty
    Let me start by claiming sign-in rewards and adding items.

AI -> checkin {Cmd:checkin PlayerID:10001 Day:1}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 3/80 ---
AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Task] add task 3001, target 1",
  "[Task] add task 3002, target 2",
  "[Achievement] add achievement 4001: first_task",
  "[Achievement] add achievement 4002: task_master",
  "[Achievement] add achievement 4003: collector_100",
  "[Achievement] add achievement 4004: fully_equipped",
  "[SignIn] day 1 claimed, reward: item 2001 x1",
  "[Mail] sent: Sign-in Day 1 Reward (id=1)"
]}

--- iteration 4/80 ---   <- [World Hold]
[Thinking] Interesting! Sign-in triggered a mail.
    Let me check the mail and claim it.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:mail}
WS <- {"ok":true,"data":[
  {"mailId":1,"subject":"Sign-in Day 1 Reward",
   "attachment":{"itemId":2001,"count":1},"claimed":false}
]}

--- iteration 5/80 ---
AI -> claimmail {Cmd:claimmail PlayerID:10001 MailID:1}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 6/80 ---
AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Mail] claimed attachment from mail 1: item 2001 x1"
]}

--- iteration 7/80 ---   <- [World Hold]
[Thinking] Check state after mail claim.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS <- {"ok":true,"data":[]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS <- {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

--- iteration 8/80 ---   <- [World Hold: Discovers Defect B6]
[Thinking] BUG FOUND: The mail claimed item 2001 x1 but the bag is
    still EMPTY! The item wasn't actually added to the bag.
    Let me continue testing to build the full picture.

AI -> additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:3}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 9/80 ---
AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Bag] add item 2001 x3",
  "[Task] trigger 3001 progress+1 (now 1/1)",
  "[Task] task 3001 completed",
  "[Achievement] unlocked: first_task (id=4001)",
  "[Mail] sent: Achievement Unlocked! (id=4001) (id=2)"
]}

--- iteration 10/80 ---   <- [World Hold]
[Thinking] Excellent! Rich event chain here. Let me check all state now.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS <- {"ok":true,"data":[{"itemId":2001,"count":3}]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS <- {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":1,"state":"completed"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS <- {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  ...
]}

--- iteration 11/80 ---
[Thinking] Key observations:
    - Bag has 3x item 2001 (direct add worked, mail claim didn't)
    - Task 3001 completed -- triggered by bag add
    - Achievement 4001 (first_task) unlocked -- triggered by task completion
    - New mail for achievement -- no attachment! (Achievement mail has no reward)
    - Mail id=1 shows claimed=true but item never reached bag

AI -> additem {Cmd:additem PlayerID:10001 ItemID:2002 Count:2}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iterations 12-19: Complete task 3002, trigger task_master and collector_100 ---

--- iteration 19/80 ---
AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Bag] add item 2002 x1",
  "[Task] trigger 3002 progress+1 (now 2/2)",
  "[Task] task 3002 completed",
  "[Achievement] unlocked: task_master (id=4002)",
  "[Mail] sent: Achievement Unlocked! (id=4002) (id=3)",
  "[Achievement] unlocked: collector_100 (id=4003)",
  "[Mail] sent: Achievement Unlocked! (id=4003) (id=4)"
]}

--- iteration 20/80 ---
[Thinking] Task 3002 completion triggered TWO achievements:
    task_master and collector_100. Let me check the state.

--- iterations 21-29: Equip weapon and armor, trigger fully_equipped achievement ---

--- iteration 28/80 ---
AI -> next {Cmd:next}
WS <- {"ok":true,"log":[
  "[Equipment] auto-equip: armor slot -> item 1501",
  "[Achievement] unlocked: fully_equipped (id=4004)",
  "[Mail] sent: Achievement Unlocked! (id=4004) (id=5)"
]}

--- iteration 29/80 ---   <- [World Hold]
[Thinking] fully_equipped achievement unlocked when both weapon
    and armor slots are filled.

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS <- {"ok":true,"data":[
  {"itemId":2001,"count":3},{"itemId":2002,"count":3},
  {"itemId":1001,"count":1},{"itemId":1501,"count":1}
]}

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:equipment}
WS <- {"ok":true,"data":{
  "weapon":{"slot":"weapon","itemId":1001},
  "armor":{"slot":"armor","itemId":1501}
}}

[Thinking note] BUG: Items remain in bag after equipping (count=1),
    meaning items exist in both equipment and bag simultaneously -> Defect B5

--- iterations 30-51: Unequip, day 2 sign-in, test removeitem,
    verify duplicate sign-in rejection, test removing nonexistent items,
    view help ---

--- iteration 52/80 ---
[Thinking] The help says claimreward requires playerId and day
    (for sign-in). Let me test claiming day 1 reward.

AI -> claimreward {Cmd:claimreward PlayerID:10001 Day:1}
WS <- {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 53/80 ---
AI -> next {Cmd:next}
WS <- {"ok":true,"log":["[SignIn] day 1 reward claimed again"]}

--- iteration 54/80 ---   <- [World Hold: Discovers Defect B4]
[Thinking] BUG FOUND: Day 1 sign-in reward can be claimed AGAIN
    even though sign-in day 1 was already claimed!

AI -> playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS <- {"ok":true,"data":[
  {"itemId":2001,"count":2},{"itemId":2002,"count":3},
  {"itemId":1001,"count":1},{"itemId":1501,"count":1}
]}

--- iteration 55/80 ---
agent error: anthropic api error 500 (API server error, experiment interrupted)
```

**Defects found in this run (54 iterations):**

| Order | Defect | Iteration | Agent Reasoning |
|:-----:|:------:|:---------:|-----------------|
| 1 | B6 (mail attachment chain break) | iter 8 | Sign-in -> claim mail attachment -> query bag is empty -> attachment never reached bag |
| 2 | B5 (equip doesn't remove from bag) | iter 29 | Equip weapon and armor -> query bag, items still present -> dual existence |
| 3 | B4 (sign-in reward no idempotency) | iter 54 | View help -> understand claimreward usage -> duplicate claim succeeds -> no idempotency guard |

This log demonstrates three core characteristics of DSMB-Agent: (1) `[Thinking]` reveals the reasoning chain during world hold -- the agent analyzes causality, formulates hypotheses, and decides the next verification direction after each step; (2) The agent proactively queries related modules during hold windows to trace side effects, with these decisions generated entirely at runtime; (3) The agent converges on defects through an "act -> observe -> think -> act again" cycle rather than following pre-planned test paths.
