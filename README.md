# Integration Testing Guide

大型游戏项目集成测试技术指南

[English](./README_EN.md)

## 目录

- [问题](#问题)
- [解决方案：让 AI 像技术专家一样分析运行时环境和数据，做全面的回归测试](#解决方案让-ai-像技术专家一样分析运行时环境和数据做全面的回归测试)
  - [1. 看数据](#1-看数据)
  - [2. 断点](#2-断点)
  - [3. 看日志](#3-看日志)
  - [4. 工程实践](#4-工程实践)
- [核心思想](#核心思想)
- [项目示例实战验证](#项目示例实战验证)
  - [basic — 基础流程验证](#basic--基础流程验证)
  - [cross-module — 跨模块关联](#cross-module-跨模块关联)
  - [edge-case — 边界测试](#edge-case-边界测试)
- [测试结果汇总](#测试结果汇总)

---

## 问题

现在的游戏集成测试环境需要依赖开发人员事先写好测试用例，针对业务用指定或者随机的参数简单校验一下业务逻辑。现在 AI 开发时代，应该有个更好的方案，让集成测试真正体现出"集成测试"。

传统的集成测试仅仅是对每个测试用例单独构建一个环境跑事先编写好的测试逻辑。游戏项目是复杂的：

1. **战斗系统**。属性繁多、技能效果和状态多样，问题的组合是一个庞大的规模。
2. **场景**。对拥有场景的项目来说，数量众多的 entity 相互之间产生的影响也是难以用测试用例覆盖全面。
3. **跨模块关联**。常规业务模块之间的关联测试用例只能管单业务的逻辑，产生关联影响的情况实际会很多，例如物品进入背包，是否正确触发任务、成就等等。

当出现这些复杂问题，工程师是怎么解决的？

> 调试、断点、查日志、看各个模块的日志输出顺序和数据。

## 解决方案：让 AI 像技术专家一样分析运行时环境和数据，做全面的回归测试

### 1. 看数据

项目暴露 CLI 接口，让项目符合现主流的渐进式披露。例如常规的个人模块：

```bash
cli -playermgr (xxxx:playerid) -bag itemid:xxxx
```

可以根据需要多暴露几个接口：

```bash
cli -playermgr xxxx -bag type:xx    # 按类型查看
cli -playermgr xxxx -task xxxx      # 查看任务
```

这样 AI 可以自己决定看哪些模块。

### 2. 断点

项目要支持 AI 断点接口，等价于保留了运行时环境。不论底层是 update 还是 actor 模式，都预留 AI 驱动的接口：

```bash
cli -next    # 执行一次 update 或处理一次 message
```

对应程序就执行一次 update，或从 message_queue 中取出一条消息完成分发处理。

### 3. 看日志

配合 AI 自主断点可以有效地逐步产生日志，而不会像瀑布一样的涌入上下文。

### 4. 工程实践

在实际开发中，应该是以通信的方式来替代 CLI（本质上等价）。例如使用 Telnet 或者 TCP + JSON Lines 等等，后者是可以结构化的方式输出，直接上 WebSocket + JSON 也是不错的。

AI 可以自己维护这套 CLI，因为主要就是给 AI 用的。让大模型自己推理增加修改整个游戏项目的 CLI，使其更全面。

```bash
nc 127.0.0.1 5400
# 请求
> {"cmd": "playermgr", "playerId": 10001, "sub": "bag", "itemId": 2001}
# 响应
< {"ok": true, "data": {"itemId": 2001, "count": 5, "source": "drop"}}
# 断点
> {"cmd": "next"}
< {"ok": true, "log": ["[Bag] add item 2001 x5", "[Task] trigger 3001 progress+1"]}
```

在实践中，需要考虑底层实现做对应调整。例如在 Skynet 的 actor 模型中，AI 需要介入的单元是每个 service（工作区固定，不管是哪个 worker_thread 来驱动 service 都是继续推进指令）。在 Golang 服务器中，应将并发控制的关注点从执行单元（goroutine）提升到通信语义（channel）。由于 goroutine 与逻辑实体之间不存在固定映射关系，系统设计应围绕"状态归属 + 消息流"展开，而不是围绕 goroutine 的创建与生命周期管理。goroutine 仅作为运行时执行载体存在。

---

## 核心思想

### 规则驱动 + 关联推理

将每个模块的业务条件以规则形式维护，模块间的关联关系维护在 wiki 中。Agent 根据业务逻辑和规则，分析所影响模块的日志，而非依赖预设的测试用例。模块间的关联可以通过分析代码中的事件注册与调用链，以及观察操作后多模块的日志输出来发现，逐步沉淀到 wiki 中。

例如背包（bag）与任务（task）、成就（achievement）有事件关联：每次背包 additem，事件应只触发一次。Agent 根据任务规则和进入背包的物品，推断哪些任务应受影响，成就同理。

### 战斗系统的延伸

战斗系统同理——场景中分布 N 个 entity 在不同位置，Agent 通过指令获取 entity 信息、施放技能，再获取当前区域 entity 状态，判断命中目标是否符合设定，逐条校验属性计算和战斗公式。

### 日志增强与知识沉淀

在开发阶段，日志加上 debug 信息，格式如 `[2026-04-13 12:00:00] bag.go additem:136`（时间戳 + 文件名 + 函数名 + 行号）。Agent 通过扫描日志文件发现模块间的隐含关系，最终写入 wiki 供人工审核，逐步完善。

### Agent 自主推理测试场景

传统的人为编写测试用例，需要开发人员自行考虑正常输入、异常输入和边界问题，让 Agent 自主推理这些场景的覆盖更合适。

## 项目示例实战验证

基于上述架构模式，我们实现了完整的项目示例 [ai-integration-test-demo](./ai-integration-test-demo/)，使用 Go + WebSocket + GLM-5.1 进行 AI 驱动集成测试。以下为三个测试场景的完整执行日志（API Key 已脱敏）。

### basic — 基础流程验证

```terminal
$ ./bin/server -mode test -scenario basic -api-key *** -model glm-5.1 -base-url https://open.bigmodel.cn/api/paas/v4

2026/04/13 15:07:46 game server started on :5400
2026/04/13 15:07:47 client connected: 127.0.0.1:54408
2026/04/13 15:07:47 running AI test scenario: basic

2026/04/13 15:08:05 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:08:05 WS ← {"ok":true,"data":[]}

2026/04/13 15:08:05 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
2026/04/13 15:08:05 WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

2026/04/13 15:08:05 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
2026/04/13 15:08:05 WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"locked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

2026/04/13 15:08:21 AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:5}
2026/04/13 15:08:21 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:08:37 AI → next {Cmd:next}
2026/04/13 15:08:37 WS ← {"ok":true,"log":[
  "[Task] add task 3001, target 1",
  "[Task] add task 3002, target 2",
  "[Achievement] add achievement 4001: first_task",
  "[Achievement] add achievement 4002: task_master",
  "[Achievement] add achievement 4003: collector_100",
  "[Bag] add item 2001 x5",
  "[Task] trigger 3001 progress+1 (now 1/1)",
  "[Task] task 3001 completed",
  "[Achievement] unlocked: first_task (id=4001)"
]}

2026/04/13 15:08:53 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:08:53 WS ← {"ok":true,"data":[{"itemId":2001,"count":5}]}

2026/04/13 15:08:53 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
2026/04/13 15:08:53 WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":1,"state":"completed"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

2026/04/13 15:08:53 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
2026/04/13 15:08:53 WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

========== TEST REPORT ==========
All three modules verified. Let me also check if the collector_100 achievement (4003)
should have triggered — it requires 2+ unlocked achievements, and we only have 1 so
far, so it correctly remains locked.

---

## Integration Test Summary — Player 10001

### PASS ✅

| Check | Expected | Actual | Result |
|-------|----------|--------|--------|
| Bag — item 2001 added | count = 5 | count = 5 | ✅ PASS |
| Task 3001 — progress updated | progress = 1 | progress = 1 | ✅ PASS |
| Task 3001 — state completed | state = "completed" | state = "completed" | ✅ PASS |
| Achievement 4001 — unlocked | state = "unlocked" | state = "unlocked" | ✅ PASS |
| Task 3002 — unaffected | progress = 0, state = "active" | progress = 0, state = "active" | ✅ PASS |
| Achievement 4002 — still locked | state = "locked" | state = "locked" | ✅ PASS |
| Achievement 4003 — still locked | state = "locked" (needs 2+) | state = "locked" | ✅ PASS |
| Cross-module chain | Item→Task→Achievement | Logs confirm full chain | ✅ PASS |

### FAIL ❌
None found.

### WARN ⚠️
1. **Task progress increments by 1, not by item count**: Adding 5x item 2001 only
   incremented task 3001 progress by 1 (to 1/1). This appears to be by design
   (presence-based, not quantity-based), but worth confirming the intended behavior.
2. **Achievement 4003 (collector_100) threshold unclear**: With only 1 of 3 achievements
   unlocked, 4003 stays locked. Its name suggests it might require 100% collection.
3. **No idempotency guard visible**: The logs don't show whether re-triggering task 3001
   completion or achievement 4001 unlock would be safely ignored.
=================================
```

### cross-module — 跨模块关联

```terminal
$ ./bin/server -mode test -scenario cross-module -api-key *** -model glm-5.1 -base-url https://open.bigmodel.cn/api/paas/v4

2026/04/13 15:12:08 game server started on :5400
2026/04/13 15:12:08 client connected: 127.0.0.1:33748
2026/04/13 15:12:08 running AI test scenario: cross-module

2026/04/13 15:12:27 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:12:27 WS ← {"ok":true,"data":[]}

2026/04/13 15:12:27 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
2026/04/13 15:12:27 WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

2026/04/13 15:12:27 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
2026/04/13 15:12:27 WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"locked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

--- Step 1: additem 2001 x1 ---
2026/04/13 15:12:47 AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:1}
2026/04/13 15:12:47 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:13:04 AI → next {Cmd:next}
2026/04/13 15:13:04 WS ← {"ok":true,"log":[
  "[Task] add task 3001, target 1",
  "[Task] add task 3002, target 2",
  "[Achievement] add achievement 4001: first_task",
  "[Achievement] add achievement 4002: task_master",
  "[Achievement] add achievement 4003: collector_100",
  "[Bag] add item 2001 x1",
  "[Task] trigger 3001 progress+1 (now 1/1)",
  "[Task] task 3001 completed",
  "[Achievement] unlocked: first_task (id=4001)"
]}

2026/04/13 15:13:17 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task TaskID:3001}
2026/04/13 15:13:17 WS ← {"ok":true,"data":{"taskId":3001,"target":1,"progress":1,"state":"completed"}}

2026/04/13 15:13:17 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
2026/04/13 15:13:17 WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

--- Step 2: additem 2002 x2 ---
2026/04/13 15:13:34 AI → additem {Cmd:additem PlayerID:10001 ItemID:2002 Count:2}
2026/04/13 15:13:34 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:13:42 AI → next {Cmd:next}
2026/04/13 15:13:42 WS ← {"ok":true,"log":[
  "[Bag] add item 2002 x2",
  "[Task] trigger 3002 progress+1 (now 1/2)"
]}

2026/04/13 15:14:14 AI → next {Cmd:next}
2026/04/13 15:14:14 WS ← {"ok":true}

2026/04/13 15:14:14 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task TaskID:3002}
2026/04/13 15:14:14 WS ← {"ok":true,"data":{"taskId":3002,"target":2,"progress":1,"state":"active"}}

2026/04/13 15:14:14 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag ItemID:2002}
2026/04/13 15:14:14 WS ← {"ok":true,"data":{"itemId":2002,"count":2}}

--- Step 3: additem 2002 x1 (to complete task 3002) ---
2026/04/13 15:14:34 AI → additem {Cmd:additem PlayerID:10001 ItemID:2002 Count:1}
2026/04/13 15:14:34 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:14:41 AI → next {Cmd:next}
2026/04/13 15:14:41 WS ← {"ok":true,"log":[
  "[Bag] add item 2002 x1",
  "[Task] trigger 3002 progress+1 (now 2/2)",
  "[Task] task 3002 completed",
  "[Achievement] unlocked: task_master (id=4002)",
  "[Achievement] unlocked: collector_100 (id=4003)"
]}

--- Final state ---
2026/04/13 15:14:55 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:14:55 WS ← {"ok":true,"data":[{"itemId":2001,"count":1},{"itemId":2002,"count":3}]}

2026/04/13 15:14:55 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
2026/04/13 15:14:55 WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":1,"state":"completed"},
  {"taskId":3002,"target":2,"progress":2,"state":"completed"}
]}

2026/04/13 15:14:55 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
2026/04/13 15:14:55 WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"unlocked"},
  {"achId":4003,"name":"collector_100","state":"unlocked"}
]}

========== TEST REPORT ==========
## Final State Summary

| Module | Details |
|--------|---------|
| Bag | Item 2001 x1, Item 2002 x3 |
| Tasks | 3001: completed (1/1), 3002: completed (2/2) |
| Achievements | 4001 (first_task): unlocked, 4002 (task_master): unlocked, 4003 (collector_100): unlocked |

---

### ✅ PASS — Correct Behaviors

1. **Item Addition to Bag**: Items correctly added with proper counts (2001 x1, 2002 x2 + x1 = x3).
2. **Task 3001 Chain**: Adding 1 item 2001 correctly increments task 3001 progress by 1,
   completes it (1/1), and unlocks achievement 4001. Full chain works perfectly.
3. **Task Completion → Achievement Unlock**: Both tasks correctly trigger their associated
   achievements upon completion.
4. **Achievement 4003 (collector_100) Meta-Achievement**: When achievement 4002 was unlocked
   (making it the 2nd unlocked achievement), achievement 4003 correctly auto-unlocked.
5. **Task State Transitions**: Tasks correctly transition from "active" to "completed" when
   progress meets the target.
6. **No Duplicate Unlocks**: Achievements that were already unlocked were not re-triggered.
7. **Initial State Clean**: All modules started in expected clean/zero state.

### ❌ FAIL — Bug Found

**Bug: Task progress only increments by 1 regardless of item count when adding items in bulk**

- **Evidence**: When `additem itemId=2002 count=2` was executed, the bag correctly received
  2 items, but the task progress only incremented by 1 (1/2 instead of 2/2).
- **Log output**: `[Bag] add item 2002 x2` followed by `[Task] trigger 3002 progress+1
  (now 1/2)` — progress increment is hardcoded to +1 instead of matching the item count.
- **Expected behavior**: `[Task] trigger 3002 progress+2 (now 2/2)` — adding 2 items
  should advance progress by 2.
- **Root cause hypothesis**: The task trigger system sends a single notification when items
  are added, and the task progress handler always increments by 1 regardless of the
  quantity. It should either pass the item count to the task progress handler, or fire one
  progress event per item in the batch.

### ⚠️ WARN — Potential Issues

1. **No idempotency guard on task progress**: If a completed task receives additional item
   triggers, it's unclear whether they are silently ignored or could cause issues.
2. **Bulk add is atomic in bag but not in task triggers**: The bag treats `count=2` as a
   single atomic addition, but the task system doesn't respect the count.
3. **Achievement 4003 trigger timing**: Achievement 4003 unlocked in the same step as
   achievement 4002, which is correct but means it's tightly coupled.
=================================
```

### edge-case — 边界测试

```terminal
$ ./bin/server -mode test -scenario edge-case -api-key *** -model glm-5.1 -base-url https://open.bigmodel.cn/api/paas/v4

2026/04/13 15:15:52 game server started on :5401
2026/04/13 15:15:52 client connected: 127.0.0.1:53996
2026/04/13 15:15:52 running AI test scenario: edge-case

--- Initial state check ---
2026/04/13 15:16:08 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:16:08 WS ← {"ok":true,"data":[]}

2026/04/13 15:16:08 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
2026/04/13 15:16:08 WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

--- Test 1: additem count=0 ---
2026/04/13 15:16:19 AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:0}
2026/04/13 15:16:19 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:16:30 AI → next {Cmd:next}
2026/04/13 15:16:30 WS ← {"ok":true,"log":[
  "[Bag] reject add item 2001: invalid count 0"
]}

2026/04/13 15:16:49 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:16:49 WS ← {"ok":true,"data":[]}

--- Test 2: removeitem non-existent item ---
2026/04/13 15:17:02 AI → removeitem {Cmd:removeitem PlayerID:10001 ItemID:2001 Count:1}
2026/04/13 15:17:02 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:17:20 AI → next {Cmd:next}
2026/04/13 15:17:20 WS ← {"ok":true,"log":[
  "[Bag] remove item 2001 x1 failed: not enough"
]}

2026/04/13 15:17:39 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:17:39 WS ← {"ok":true,"data":[]}

--- Test 3: additem normal, then remove more than available ---
2026/04/13 15:17:54 AI → additem {Cmd:additem PlayerID:10001 ItemID:2002 Count:2}
2026/04/13 15:17:54 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:18:07 AI → next {Cmd:next}
2026/04/13 15:18:07 WS ← {"ok":true,"log":[
  "[Bag] add item 2002 x2",
  "[Task] trigger 3002 progress+1 (now 1/2)"
]}

2026/04/13 15:18:26 AI → removeitem {Cmd:removeitem PlayerID:10001 ItemID:2002 Count:5}
2026/04/13 15:18:26 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:18:33 AI → next {Cmd:next}
2026/04/13 15:18:33 WS ← {"ok":true,"log":[
  "[Bag] remove item 2002 x5 failed: not enough"
]}

2026/04/13 15:18:39 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:18:39 WS ← {"ok":true,"data":[{"itemId":2002,"count":2}]}

--- Test 4: additem negative count ---
2026/04/13 15:18:48 AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:-3}
2026/04/13 15:18:48 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:19:01 AI → next {Cmd:next}
2026/04/13 15:19:01 WS ← {"ok":true,"log":[
  "[Bag] reject add item 2001: invalid count -3"
]}

2026/04/13 15:19:18 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:19:18 WS ← {"ok":true,"data":[{"itemId":2002,"count":2}]}

--- Test 5: removeitem negative count (CRITICAL BUG) ---
2026/04/13 15:19:32 AI → removeitem {Cmd:removeitem PlayerID:10001 ItemID:2002 Count:-1}
2026/04/13 15:19:32 WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

2026/04/13 15:19:43 AI → next {Cmd:next}
2026/04/13 15:19:43 WS ← {"ok":true,"log":[
  "[Bag] remove item 2002 x-1"
]}

2026/04/13 15:19:58 AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
2026/04/13 15:19:58 WS ← {"ok":true,"data":[{"itemId":2002,"count":3}]}
                                         ^^^^^^^^^^^ count went from 2→3!

========== TEST REPORT ==========
Confirmed — 3 items in bag but task progress only at 1/2. The removeitem-with-negative-count
added items silently without cross-module task triggers.

---

## Test Summary

### PASS ✅

| Test | Expected | Actual | Result |
|------|----------|--------|--------|
| Add item count=0 | Rejected | `[Bag] reject add item 2001: invalid count 0` — bag unchanged | ✅ PASS |
| Remove non-existent item | Fail gracefully | `[Bag] remove item 2001 x1 failed: not enough` — bag unchanged | ✅ PASS |
| Remove more than available | Fail | `[Bag] remove item 2002 x5 failed: not enough` — bag unchanged | ✅ PASS |
| Add item negative count | Rejected | `[Bag] reject add item 2001: invalid count -3` — bag unchanged | ✅ PASS |
| State consistency after rejected ops | No corruption | Bag, tasks, achievements all unchanged after each rejection | ✅ PASS |

### FAIL ❌

| Test | Expected | Actual | Severity |
|------|----------|--------|----------|
| Remove item with negative count (-1) | Rejected with error | Accepted! `[Bag] remove item 2002 x-1` — count **increased** from 2→3 | 🔴 CRITICAL |

### BUG DETAILS: `removeitem` with negative count is an exploitable validation gap

**Evidence:**
1. `removeitem` with `count=-1` was **not rejected** — it executed as `remove item 2002 x-1`
2. The item count **increased** from 2 to 3 (removing -1 = adding +1)
3. The `additem` path correctly validates: both `count=0` and `count=-3` were rejected
4. The `removeitem` path does **not** validate for negative counts
5. **Cross-module bypass**: Items added via this exploit did **not** trigger task progress.
   Task 3002 remained at 1/2 despite bag having 3x item 2002.

**Impact:**
- Players can use `removeitem` with negative counts as an **unrestricted item duplication exploit**
- The exploit **bypasses cross-module triggers** (no task progress, no achievement advancement),
  allowing silent inventory inflation
- This is an asymmetric validation bug: `additem` validates negative/zero counts,
  but `removeitem` does not

**Recommended Fix:** Add the same count validation to `removeitem` that exists in `additem` —
reject any `count <= 0` before processing.

### ⚠️ WARN — Potential Issues

1. **Task progress granularity**: Adding `item 2002 x2` in a single `additem` call only
   triggered task 3002 progress by +1 (not +2).
2. **No rollback mechanism**: If a remove operation partially succeeds before failing,
   there's no evidence of transactional safety.
=================================
```

---

## 测试结果汇总

三个场景全部跑完，GLM-5.1 自主发现了真实 bug：

| 场景 | 结果 |
|------|------|
| basic | 8/8 通过，跨模块链正常 |
| cross-module | 7/7 通过，但发现 Bug: 批量添加物品时任务进度只+1而非按数量累加 |
| edge-case | 4/5 通过，发现 🔴 严重Bug: removeitem 负数 count 未校验，可被利用无限刷物品且绕过跨模块触发 |

AI 自主发现的两个 bug 是项目中真实存在的代码缺陷（`task.go` 硬编码 `Progress(tid, 1)`；`bag.go` 的 `RemoveItem` 缺少 `count <= 0` 校验），说明这个 AI 驱动集成测试方案是有效的。
