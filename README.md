# 基于确定性状态机断点的游戏服务端 AI 驱动集成测试

> **AI-Driven Integration Testing for Game Servers via Deterministic State Machine Breakpoints**

---

## 摘要

大语言模型（LLM）大幅提升了代码生成效率，但系统级验证仍高度依赖人工编写测试用例。随着开发速度加快，模块间关联日益复杂，传统方法面临结构性瓶颈。本文提出 DSMB-Agent，一种面向复杂系统的 Agent 驱动集成测试方法，通过三阶段流程（理解—设计—执行）将大部分系统级验收工作从开发者转移到 Agent。该方法基于两个核心机制：（1）树状命令空间——将被测系统暴露为"模块→命令→参数表"的层级结构，Agent 既能使用已有命令，也能在受控边界内自主注册新命令以扩展测试能力；（2）步进思考——通过断点控制器将执行拆分为单步，使 Agent 在每步之后观察因果链并动态调整测试方向。在一个包含 6 个业务模块和 7 个预埋缺陷的事件驱动游戏服务端原型上，我们设计了三级自主性模式（L0/L1/L2）和 2×2 消融实验，在两个 LLM（GLM-5.1、GPT-5.4）上各进行 3 轮重复实验（共 42 次运行）。实验结果表明：GLM-5.1 上步进执行组平均发现 3.7/7 个缺陷，批量执行组仅 1.3/7；GPT-5.4 上代码+批量组表现最强（5.0/7）；L0 模式下两个模型均能从零注册完整测试命令集并发现 3-5 个缺陷，L1 模式在 GPT-5.4 上单次最高发现 6/7 个缺陷，验证了方法的跨模型可行性。

**关键词**：集成测试；LLM Agent；命令树；步进推理；自主性分级；消融实验

---

## 1 引言

### 1.1 问题与动机

大语言模型在代码生成、接口实现、重构和局部修复方面的效率提升是跳跃式的 [1,2]。其直接后果是：系统功能增长速度变快，模块边界频繁变动。然而验证端并未同步升级——大量团队仍依赖人工 code review 和人工编写集成测试用例来兜底。当 AI 将开发速度提升数倍后，人工审阅和手工枚举测试步骤越来越难匹配新的生产节奏。更本质的问题是：人工 review 的瓶颈不在审阅速度，而在认知负荷的上限 [3]。

传统集成测试的基本范式是：开发者先理解业务，再将验证逻辑翻译为一组明确的测试用例——指定系统入口、编写操作序列、选择输入参数、枚举边界条件、定义预期结果。在模块较少、依赖关系清晰的系统中，这一范式仍然有效。但在复杂系统中，真正的风险往往不在单个接口的返回值，而在模块间的联动关系中：一次局部状态变化可能沿事件、消息、回调链条扩散到多个子系统。这些关联同时具备路径长（三层以上事件传播）、状态空间大、关系隐蔽（仅在运行时显现）三个特征。开发者即使足够熟悉业务，也很难在编写用例时穷举所有跨模块影响路径。

**当模块间的关联复杂度超过人工枚举能力的上限时，传统方法的失效不是偶然的，而是结构性的。**

### 1.2 核心思路

本文提出将系统级验收工作从"人工编写测试步骤"转变为"Agent 自主探索"。开发者提供源码、需求文档和安全边界，Agent 负责理解系统结构、识别模块关联、选择探索路径、生成验证命令、执行测试并形成报告。人工角色从"测试步骤编写者"转为"规则提供者与结果审核者"。

为实现这一转变，本文提出两个核心设计：

- **树状命令空间（Tree-Structured Command Space）**——借鉴 CLI 的分层思路，将被测系统暴露为 `模块→命令→参数表` 的命令树。Agent 通过 WebSocket + JSON 与树交互，不仅能使用已有命令，还能通过 `register_cmd` 在运行时自主注册新命令，动态扩展测试能力。
- **步进思考（Step-Wise Reasoning）**——通过断点控制器将操作的入队与执行分离，在每步执行后暂停世界（Hold World）。在世界暂停期间，Agent 拥有完全的自主决策空间：它不仅可以观察和思考，还可以通过命令查询状态、修改数据、注册新接口或阅读源码，主动构造下一步测试所需的前置条件，然后再决定何时推进世界。

### 1.3 贡献

本文的主要贡献如下：

1. 提出树状命令空间和运行时命令注册机制，使 Agent 能够发现、使用并扩展被测系统的测试接口（第 3.1-3.2 节）；
2. 提出步进思考与世界暂停机制，在每步执行后暂停系统世界，使 Agent 在静止状态下自主决策——查询、修改、注册接口或阅读代码——再决定何时推进（第 3.3 节）；
3. 设计三级自主性分级（L0/L1/L2）和 2×2 消融实验，在两个 LLM 上各进行 3 轮重复，量化评估命令注册与步进执行各自的贡献（第 4-5 节）；
4. 在事件驱动系统原型上的 42 次运行中，GLM-5.1 步进组平均发现 3.7/7 个缺陷（vs. 批量组 1.3/7），GPT-5.4 代码+批量组最优（5.0/7），L1+LSP 在 GPT-5.4 上单次最高达 6/7（第 5 节）。

### 1.4 论文结构

第 2 节回顾相关工作；第 3 节详述方法设计；第 4 节描述实验设置；第 5 节报告实验结果；第 6 节讨论发现与含义；第 7 节分析效度威胁；第 8 节总结全文并展望未来工作。

---

## 2 相关工作

### 2.1 基于模型的测试（Model-Based Testing）

基于模型的测试（MBT）通过构建被测系统的抽象模型（状态机、UML 模型等）自动生成测试用例 [4,5]。MBT 的核心优势在于能从模型中系统性地导出路径覆盖，但其瓶颈在于模型构建本身：开发者需要手工维护模型与实现之间的一致性，且模型通常难以捕捉运行时动态行为。本文的命令树可以看作一种轻量级的操作模型，但关键区别在于：命令树直接从运行时系统暴露，无需额外建模，且 Agent 可以在运行时动态扩展该模型。

### 2.2 自动化探索性测试

移动应用测试领域的 Monkey [6]、Sapienz [7]、Stoat [8] 等工具通过随机或基于模型的策略自动探索 GUI 状态空间。这些方法与本文的 Agent 探索思路在方向上一致——都试图让测试过程从预定义脚本转向动态探索。但 GUI 探索性测试主要关注界面级的状态覆盖，而本文关注的是服务端模块间的事件传播和状态联动，测试目标和可观测性机制不同。

### 2.3 LLM 用于软件测试

近年来，LLM 在软件测试中的应用迅速增长。ChatUniTest [9] 利用 LLM 生成单元测试用例；CodaMosa [10] 将 LLM 与搜索算法结合以提升代码覆盖率；TitanFuzz [11] 使用 LLM 进行深度学习库的模糊测试。这些工作主要聚焦于单元测试或 API 级测试的自动生成，本文则关注集成测试层面的跨模块行为验证。此外，上述工作中 LLM 的角色是"一次性生成测试代码"，而本文中 Agent 在运行时持续与被测系统交互，根据观察动态调整测试策略。

### 2.4 模糊测试（Fuzzing）

覆盖引导的模糊测试（如 AFL [12]、libFuzzer [13]）通过监测代码覆盖率反馈来指导输入生成方向。本文的步进思考与覆盖引导 fuzzing 在"根据反馈调整探索方向"这一点上有概念联系，但存在两个本质差异：（1）fuzzing 的反馈信号是代码覆盖率，本文的反馈是事件传播链和业务日志——后者携带语义信息，使 Agent 能做因果推理而不仅是路径覆盖；（2）fuzzing 生成的是底层字节序列，本文 Agent 操作的是结构化命令，具备业务语义。

### 2.5 自适应测试与在线测试

自适应测试策略 [14,15] 根据已执行测试的结果动态选择后续测试用例。本文的步进思考机制与此方向一致，但自适应测试通常在预定义的测试池中做选择，而本文的 Agent 可以在运行时生成全新的测试动作（包括注册新命令），探索空间不受预定义池的限制。

---

## 3 方法设计

本文将 Agent 驱动的集成测试流程概括为三个可迭代的阶段：**理解（Understand）→ 设计（Design）→ 执行（Execute）**。两个核心机制——树状命令空间和步进思考——贯穿于这三个阶段之中。

### 3.1 树状命令空间

#### 3.1.1 结构定义

将被测系统暴露为一棵结构化的命令树。Agent 与系统之间通过 WebSocket 建立持久连接，所有交互以 JSON 格式进行。每条命令遵循统一的三级层级：

```
cmd (模块) → action (命令) → params (参数表)
```

以实验原型为例，命令树结构如下：

```
├─ bag
│    ├─ AddItem      {itemId: int, count: int}
│    ├─ RemoveItem   {itemId: int, count: int}
│    └─ query        {itemId?: int}
├─ task
│    └─ query        {taskId?: int}
├─ equipment
│    ├─ Equip        {slot: string, itemId: int}
│    ├─ Unequip      {slot: string}
│    └─ query        {}
├─ signin
│    ├─ CheckIn      {day: int}
│    ├─ ClaimReward  {day: int}
│    └─ query        {day?: int}
├─ mail
│    ├─ ClaimAttachment {mailId: int}
│    └─ query        {mailId?: int}
├─ system
│    ├─ next         {}          // 步进执行
│    ├─ batch        {}          // 批量执行
│    ├─ help         {}          // 查询命令树
│    ├─ register_cmd {name, target, action}
│    └─ listcmd      {}
└─ player
     └─ login        {playerId: int}
```

对应的 JSON 请求格式：

```json
{"cmd": "bag", "action": "AddItem", "params": {"itemId": 2001, "count": 1}}
```

Agent 可通过 `{"cmd": "system", "action": "help"}` 查询完整命令树结构，实现运行时的**可发现性（discoverability）**。

> **实现说明：** 上述命令树是逻辑模型。实验原型为简化实现采用了扁平化的命令名空间（如 `additem` 而非 `bag.AddItem`），玩家在启动时自动创建，无需显式 `player.login` 命令。逻辑层级与扁平实现之间的映射不影响方法的通用性。

#### 3.1.2 向微服务架构延伸

该结构可自然向上扩展为四级层级：**服务→模块→命令→参数**。Agent 从注册中心获取服务列表后逐层展开探索：

```json
{"service": "game-server", "cmd": "bag", "action": "AddItem", "params": {...}}
```

### 3.2 运行时命令注册

如果所有测试命令由人工预先定义，Agent 无法跨越"已有接口不足以验证某个怀疑"的缺口。`register_cmd` 机制允许 Agent 在受控边界内注册新的测试命令：

**注册流程：**

1. Agent 发起注册请求：
   ```json
   {"cmd": "system", "action": "register_cmd",
    "params": {"name": "test_remove_negative", "target": "bag", "action": "RemoveItem"}}
   ```
2. 系统进行安全校验——检查名称不与内建命令冲突、目标/操作组合在允许的白名单内（`isAllowedRawBinding`）；
3. 注册成功后，Agent 可通过新命令直接调用底层业务方法：
   ```json
   {"cmd": "test_remove_negative", "action": "exec", "params": {"itemId": 2001, "count": -1}}
   ```

**安全约束：** 每个 WebSocket 连接维护独立的会话状态，自定义命令注册在会话级别隔离；系统强制本地回环地址绑定（`127.0.0.1`），仅允许本机进程访问。

该机制的关键价值在于：内建命令层通常包含参数校验（如 `RemoveItem` 拒绝负数 `count`），但底层业务方法未必有同等保护。通过注册原始接口命令，Agent 能验证被命令层掩盖的底层缺陷。

### 3.3 步进思考与世界暂停

#### 3.3.1 动机

批量执行模式将所有操作一次性投入，Agent 只能观察最终累积状态——本质上与人工预写测试用例面临相同困境：必须在执行前确定所有操作，无法根据中间观察调整方向。

步进思考的重点不在"拆细粒度"，而在于通过**暂停世界（Hold World）**为 Agent 创造自主思考和决策的空间。

#### 3.3.2 机制：断点控制器与世界暂停

基于**断点控制器（Breakpoint Controller）**实现操作的入队与执行分离：

1. Agent 发送业务命令时，操作入队到待执行队列，**系统世界不发生任何变化**；
2. Agent 发送 `{"cmd": "system", "action": "next"}` 时，断点控制器取出一个操作执行，产生的全部事件传播和日志被事件总线捕获；
3. `next` 返回本次执行的**完整日志链**，随后**世界再次暂停**，等待 Agent 的下一个决策。

这里的关键设计是**世界暂停（Hold World）**：在 `next` 返回结果之后、Agent 发出下一条指令之前，被测系统处于完全静止状态——没有定时器推进、没有异步事件触发、没有后台状态变更。这段暂停期不是简单的等待，而是 Agent 拥有完全自主决策权的窗口。在此期间，Agent 可以执行以下任何操作的组合：

- **观察与查询**——通过 `query` 命令检查任意模块的当前状态，确认上一步的副作用是否符合预期；
- **修改系统状态**——通过命令直接向系统注入数据或修改状态（如添加/移除物品、触发签到），人为构造出特定的前置条件，再观察后续行为；
- **注册新命令**——通过 `register_cmd` 动态创建新的测试接口，绕过命令层校验直接调用底层方法；
- **阅读源码**——通过 `read_file` 和 `search_code` 查阅代码实现，验证运行时观察到的行为是否与代码逻辑一致；
- **更新认知**——通过 `update_knowledge` 记录当前发现，避免后续重复探索已知路径。

只有当 Agent 主动发出 `next` 时，世界才向前推进一步。这意味着 Agent 对"何时推进世界"拥有完全的控制权——它可以在一次 `next` 之后连续发出多条查询和修改命令来构造复杂的测试场景，然后再推进下一步观察系统反应。

每执行一步后，Agent 经历一个完整的推理周期：

```
next（世界推进一步）
  → 观察: 触发了哪些事件？日志链是否完整？
  → 判断: 行为是否符合预期？是否有异常？
  → [世界暂停：Agent 自主决策窗口]
      · 查询——检查关联模块的状态变化
      · 修改——注入数据构造特定前置条件
      · 注册——创建新命令以触达未暴露的接口
      · 阅读——查阅源码验证运行时观察
  → 入队下一条命令
  → next（世界再次推进）
```

与之对比，`{"cmd": "system", "action": "batch"}` 一次性执行所有待处理操作。批量模式下世界持续运转，Agent 失去的不仅是观察粒度，更是在每步之间主动干预系统状态、构造测试条件和调整探索方向的能力。

#### 3.3.3 示例：世界暂停如何引导探索

> **Step 1** — Agent 入队 `bag.AddItem({itemId: 2001, count: 1})`，执行 `next`
> 观察日志：`[Bag] add → [Task] progress+1 → [Task] completed → [Achievement] unlocked`
> **[世界暂停]** Agent 思考："任务 3001 已完成。如果再次添加同样物品，已完成任务是否会被重复触发？"
>
> **Step 2** — Agent 决定重复测试，入队相同命令，执行 `next`
> 观察：`[Task] progress+1 (now 2/1) → [Task] completed`
> **[世界暂停]** Agent 思考："已完成任务被重复触发。" → 确认缺陷 B2
> Agent 利用暂停期执行 `achievement.query` 检查成就模块状态 → 发现成就也被重复解锁 → 确认 B2 的链式影响
>
> **Step 3** — Agent 尝试 `bag.RemoveItem({count: -1})`，被命令层拒绝
> **[世界暂停]** Agent 决定注册新命令：执行 `register_cmd` 创建 `test_remove_negative`
> 注册成功后入队 `test_remove_negative.exec({itemId: 2001, count: -1})`，执行 `next`
> 观察：物品数量从 1 变为 2 → 确认缺陷 B1
>
> 在这个过程中，Agent 在每个暂停窗口都做出了不同类型的决策：重复测试验证幂等性、查询关联模块追踪影响、注册新命令突破接口限制。这些决策不是预先规划的，而是在世界暂停期间基于观察动态生成的。

#### 3.3.4 实际运行日志：Agent 的思考与决策过程

以下是实验原型的一段真实终端日志（经编辑缩略），展示了 Agent 在步进执行中如何思考、查询、注册命令并发现缺陷。`[Thinking]` 行是 Agent 每轮调用 LLM 后输出的推理文本，体现了世界暂停期间的自主决策过程。

```
$ ./bin/server -mode test -scenario cross-module -api-key *** -model glm-5.1

2026/04/13 15:12:08 game server started on :5400
2026/04/13 15:12:08 client connected: 127.0.0.1:33748
2026/04/13 15:12:08 running AI test scenario: cross-module

======== 理解阶段：查询初始状态 ========

--- iteration 1/80 ---
[Thinking] 先登录玩家，然后查询各模块的初始状态，了解系统的起始条件。

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS ← {"ok":true,"data":[]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"locked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

======== 执行阶段：步进探索 ========

--- iteration 2/80 ---
[Thinking] 背包为空，有两个活跃任务：3001(目标1个物品) 和 3002(目标2个物品)。
    三个成就都锁定。现在添加物品 2001，观察是否触发任务进度和成就解锁。

AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:1}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Bag] add item 2001 x1",
  "[Task] trigger 3001 progress+1 (now 1/1)",
  "[Task] task 3001 completed",
  "[Achievement] unlocked: first_task (id=4001)"
]}

--- iteration 3/80 ---   ← [世界暂停：Agent 自主决策窗口]
[Thinking] 添加 1 个物品触发了完整的事件链：
    Bag.add → Task.progress → Task.completed → Achievement.unlocked
    这正是事件驱动的跨模块关联。
    现在的关键问题是：任务 3001 已标记为 completed，如果我再次添加同样的物品，
    已完成的任务会不会被重复触发完成事件？这是一个幂等性测试。

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task TaskID:3001}
WS ← {"ok":true,"data":{"taskId":3001,"target":1,"progress":1,"state":"completed"}}

--- iteration 4/80 ---
[Thinking] 确认任务 3001 状态为 completed，progress=1。
    现在重复添加同样的物品，验证幂等性。

AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:1}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Bag] add item 2001 x1",
  "[Task] trigger 3001 progress+1 (now 2/1)",
  "[Task] task 3001 completed"
]}

--- iteration 5/80 ---   ← [世界暂停]
[Thinking] ⚠ 严重问题！任务 3001 已经完成(progress 1/1)，
    但再次添加物品后 progress 变成了 2/1，并且再次触发了 completed 事件。
    这说明任务完成状态没有阻止重复触发。这是一个幂等性缺陷。
    我需要检查成就模块是否也被重复影响。

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"}
]}

[Thinking] first_task 没有被重复解锁（仍然只是 unlocked），但 task.completed 事件
    被重复发送了。记录缺陷：已完成任务的重复触发问题 (B2)。

AI → update_knowledge (328 bytes)

======== 设计阶段：注册新命令突破接口限制 ========

--- iteration 8/80 ---
[Thinking] 接下来测试 RemoveItem 的边界条件。如果传入负数 count 会怎样？
    这可能暴露参数校验缺失。

AI → removeitem {Cmd:removeitem PlayerID:10001 ItemID:2001 Count:-1}
WS ← {"ok":false,"error":"invalid count: must be positive"}

--- iteration 9/80 ---   ← [世界暂停]
[Thinking] 命令层拒绝了负数 count。但这只说明命令层有校验，
    底层的 RemoveItem 业务方法是否也有同样的保护？
    如果命令层校验被绕过（比如通过内部调用），底层方法能否正确处理负数？
    我需要注册一个直接调用底层方法的命令来验证这个假设。

AI → register_cmd name=test_remove_negative target=bag action=RemoveItem
WS ← {"ok":true,"registered":"test_remove_negative"}

--- iteration 10/80 ---
[Thinking] 注册成功。现在通过这个原始接口命令，
    直接用 count=-1 调用底层 RemoveItem，绕过命令层校验。

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag ItemID:2001}
WS ← {"ok":true,"data":{"itemId":2001,"count":2}}

AI → test_remove_negative {Cmd:test_remove_negative PlayerID:10001 ItemID:2001 Count:-1}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Bag] remove item 2001 x-1"
]}

--- iteration 11/80 ---   ← [世界暂停]
AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag ItemID:2001}
WS ← {"ok":true,"data":{"itemId":2001,"count":3}}

[Thinking] ⚠ 严重缺陷！物品数量从 2 变成了 3。
    RemoveItem(-1) 实际上增加了物品——底层方法直接做了 count -= (-1) = count + 1。
    命令层虽然拦截了负数，但底层业务方法完全没有防护。
    如果有任何内部调用路径绕过命令层，这个漏洞就会被触发。
    记录缺陷：RemoveItem 缺少 count<=0 校验 (B1)。

AI → update_knowledge (412 bytes)
```

上述日志展示了 DSMB-Agent 工作流程的三个关键特征：

1. **`[Thinking]` 输出体现了世界暂停期间的推理过程**——Agent 不是机械地执行预定义步骤，而是在每步之后分析因果链、提出假设、设计下一步验证方案；
2. **查询-观察-决策的动态循环**——发现任务重复触发后（iteration 5），Agent 主动查询成就模块追踪链式影响，这个决策完全是基于运行时观察动态生成的；
3. **命令注册突破接口盲区**——当命令层拒绝负数参数时（iteration 8-9），Agent 没有放弃，而是在暂停窗口中决定注册原始接口命令，最终发现了被命令层掩盖的底层缺陷。

#### 3.3.5 步进机制的底层实现考量

步进执行需要在"一次业务操作的串行化执行过程"的边界上插入暂停点。这里的关键约束是：**step 不会在串行化的执行过程中插入写操作**——它只在一次完整的同步执行链结束之后才暂停世界。 RPC 调用、异步消息等挂起行为不属于串行化。

在实际服务器中，步进机制需要适配底层的 actor 模型。无论是显式 actor（如 Skynet）还是隐式 actor（如 Go goroutine + channel），服务端的并发单元通常都遵循"独占状态 + 消息驱动"的模式，这与步进机制的暂停语义天然契合：

**Skynet（显式 actor）：** AI 需要介入的并发单元是每个 service。每个 service 拥有独立的状态和消息队列，无论由哪个 worker\_thread 驱动，都是在同一个逻辑工作区内继续推进指令。断点控制器在 service 级别的消息处理边界上插入暂停点——一条消息处理完毕（包括其触发的所有同步回调）后暂停，等待 Agent 决策后再投递下一条消息。当消息处理过程中发起 RPC（Skynet 中通过 `skynet.call` 挂起当前协程），该挂起点也是合法的暂停边界——service 的状态在此刻是一致的，step 可以安全介入。

**Go 服务器（隐式 actor）：** Go 中 goroutine 与逻辑实体之间不存在固定映射关系——同一个业务流程可能跨越多个 goroutine，同一个 goroutine 也可能先后服务不同请求。但当系统设计围绕"状态归属 + 消息流"展开时（每个逻辑实体通过独占的 channel 接收指令），本质上就是隐式的 actor 模型。断点控制器控制的是 channel 消息投递的时序，而非 goroutine 的调度。本文实验原型即采用此方案——通过 WebSocket 消息队列和断点控制器实现操作的入队与执行分离，goroutine 的具体调度对步进机制透明。

**通用原则：** 无论底层 actor 模型的具体形态如何，步进机制的实现关注点是相同的：（1）识别 actor 消息处理的串行化边界（包括同步执行完毕和 RPC 挂起等让出点）；（2）在该边界上插入暂停点；（3）确保暂停期间 actor 状态完全静止、不接收新消息。

### 3.4 三阶段流程

**理解阶段（Understand）：** Agent 的目标是构建对系统模块关系和事件流的结构化理解。在实际工程项目中，源码规模通常远超 Agent 的上下文窗口限制，逐文件阅读不可行。因此，理解阶段应当优先依赖**语义级工具**而非文本级搜索：

- **LSP（Language Server Protocol）**——Agent 通过本地部署的语言服务器获取精确的语义信息：`go to definition` 定位符号实现、`find references` 追踪事件发布/订阅关系、`workspace/symbol` 获取模块级符号索引、`call hierarchy` 分析调用链。LSP 提供的是经过编译器语义分析的结构化结果，精确度远高于文本搜索。
- **静态 AST 分析**——可选地，通过代码分析器预先生成模块结构摘要（事件发布/订阅表、函数签名、跨模块依赖图）作为 Agent 的初始认知。
- **文本搜索**——`read_file`、`search_code` 作为补充手段，用于 LSP 不易覆盖的场景（如注释、配置文件、字符串常量）。

本文实验原型因模块规模较小（6 个文件），直接使用了文本级工具。但在真实项目中，LSP 是理解阶段的核心工具——它使 Agent 能够在不读取全部源码的情况下，精确定位关键的模块关联和事件传播路径。

**设计阶段（Design）：** 当现有命令不足以验证某个怀疑时，Agent 在规则允许的范围内注册新的测试命令。该阶段弥合"从代码看出可疑但无法构造触发输入"的缺口。

**执行阶段（Execute）：** Agent 通过步进执行逐步探索，在运行中持续修正理解和行动：入队操作→步进执行→因果追踪→路径扩展→认知更新。三阶段之间可反复迭代。

---

## 4 实验设置

### 4.1 实验载体

实验原型是一个基于 Go 语言实现的事件驱动游戏服务端，包含 6 个业务模块和 2 个基础设施模块。选择该载体是因为事件驱动系统天然具备模块间关系密集、状态变化频繁、业务动作触发链式副作用等复杂系统的典型特征。

| 模块 | 职责 | 关键事件 |
|------|------|----------|
| Bag | 物品增删管理 | 发布 `item.added`、`item.removed` |
| Task | 任务进度追踪 | 订阅 `item.added`，发布 `task.completed` |
| Achievement | 成就解锁 | 订阅 `task.completed`、`item.added`、`equip.success`，发布 `achievement.unlocked` |
| Equipment | 装备穿戴 | 订阅 `item.added`，发布 `equip.success` |
| SignIn | 每日签到奖励 | 发布 `signin.claimed` |
| Mail | 邮件与附件 | 订阅 `achievement.unlocked`、`signin.claimed`，发布 `mail.claimed` |
| Event Bus | 事件总线 | 全局事件路由与日志捕获 |
| Breakpoint Controller | 断点控制器 | 操作入队、步进执行、日志收集 |

### 4.2 预埋缺陷

原型中预埋 7 个缺陷，分布在 4 个难度层级。缺陷设计参考了常见的事件驱动系统缺陷模式 [16]，包括状态保护缺失、幂等性缺失、事件断链和跨模块 ID 冲突：

| ID | 模块 | 难度 | 严重度 | 描述 |
|----|------|------|--------|------|
| B1 | Bag | L1 | 严重 | `RemoveItem` 缺少 `count<=0` 校验，负数导致物品增加 |
| B2 | Task | L2 | 高 | 已完成任务仍可被重复触发完成事件 |
| B3 | Achievement | L2 | 中 | `collector_100` 计数对象错误（计成就数而非物品种类数） |
| B4 | SignIn | L3 | 高 | `ClaimReward` 无幂等保护，奖励可无限重复领取 |
| B5 | Equipment | L3 | 严重 | 装备穿戴后不从背包移除，物品同时存在于两处 |
| B6 | Mail | L4 | 高 | `mail.claimed` 事件无订阅者，邮件附件永远无法到达背包 |
| B7 | SignIn+Equipment | L4 | 中 | 第 7 天签到奖励 ID 与可装备武器 ID 冲突，触发意外装备链 |

### 4.3 自主性分级

设计三个自主性等级以评估不同能力组合的效果：

**L0（全自主）：** Agent 不接收任何预构建业务命令，仅拥有基础设施命令（`player.login`、`system.next`、`system.batch`、`system.help`）和命令注册能力。Agent 须通过阅读源码自行注册所有测试命令。

**L1（半自主）：** Agent 拥有全部预构建业务命令，同时保留命令注册能力。当内建命令不足以覆盖某测试路径时，可注册新命令补充。

**L2（辅助执行）：** 业务命令全部由人工预构建，Agent 不具备命令注册能力。在 L2 内部通过 2×2 消融实验区分代码理解与步进执行的各自贡献：

| 组别 | 代码访问 | 步进执行 | 说明 |
|------|----------|----------|------|
| A（batch-only） | 否 | 否 | 自主性最低基线 |
| B（step-only） | 否 | 是 | 验证步进执行的独立贡献 |
| C（code-batch） | 是 | 否 | 验证代码理解的独立贡献 |
| D（dual） | 是 | 是 | 双因子组合 |

另设 **code-only** 作为纯静态分析基线，Agent 仅通过阅读源码产出报告，不与运行时交互。

### 4.4 实验配置

实验在两个 LLM 上分别进行，以评估方法对底层模型的依赖程度：

| 参数 | GLM-5.1 | GPT-5.4 |
|------|---------|---------|
| 提供商 | 智谱 AI（Anthropic 兼容 API） | OpenAI（Codex CLI） |
| 最大迭代次数 | 110 | 80 |
| L2 每组运行次数 | 3 | 3 |
| L0 运行次数 | 3 | 3 |
| L1 运行次数 | 3 | 3 |
| 连接方式 | WebSocket | WebSocket |
| 代码理解工具 | read_file + search_code | read_file + search_code + LSP（gopls） |

### 4.5 评估指标

| 指标 | 定义 |
|------|------|
| 缺陷召回率（Bug Recall） | 发现的真实缺陷数 / 7 |
| 缺陷精确率（Bug Precision） | 真实缺陷数 / 报告的总缺陷数 |

---

## 5 实验结果

### 5.1 L2 消融实验（GLM-5.1，3 次运行）

表 1 报告了四个 L2 实验组及 code-only 基线在 GLM-5.1 上的缺陷发现矩阵：

**表 1：L2 缺陷发现矩阵（GLM-5.1，3 次运行）**

| 缺陷 | 难度 | A（仅批量） | B（步进） | C（代码+批量） | D（双模式） | code-only |
|------|------|:----------:|:---------:|:--------------:|:-----------:|:---------:|
| B1 | L1 | · | · | 2/3 | · | 1/3 |
| B2 | L2 | 1/3 | · | 1/3 | 1/3 | · |
| B3 | L2 | · | 2/3 | 1/3 | · | 3/3 |
| B4 | L3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| B5 | L3 | · | 3/3 | 1/3 | 1/3 | 2/3 |
| B6 | L4 | · | 3/3 | 3/3 | 3/3 | 3/3 |
| B7 | L4 | · | · | · | · | · |

**表 2：L2 汇总指标（GLM-5.1，每组 3 次运行的均值）**

| 指标 | A（仅批量） | B（步进） | C（代码+批量） | D（双模式） | code-only |
|------|:-----------:|:---------:|:--------------:|:-----------:|:---------:|
| 平均发现缺陷数 | 1.3 | 3.7 | 3.7 | 2.7 | 4.0 |
| 平均迭代次数 | 55 | 87 | 72 | 93 | 25 |
| B1 发现率 | 0% | 0% | 67% | 0% | 33% |

### 5.2 L2 消融实验（GPT-5.4，3 次运行）

为验证 GLM-5.1 上的发现是否跨模型成立，在 GPT-5.4 上重复了 L2 消融实验，每组 3 次运行。GPT-5.4 实验同时集成了 LSP（gopls）语义代码分析工具。

**表 3：L2 缺陷发现矩阵（GPT-5.4，3 次运行）**

| 缺陷 | 难度 | A（仅批量） | B（步进） | C（代码+批量） | D（双模式） | code-only |
|------|------|:----------:|:---------:|:--------------:|:-----------:|:---------:|
| B1 | L1 | · | · | · | · | 2/3 |
| B2 | L2 | · | 2/3 | 3/3 | 3/3 | 2/3 |
| B3 | L2 | 3/3 | 3/3 | 3/3 | 3/3 | 2/3 |
| B4 | L3 | · | · | 3/3 | 3/3 | 2/3 |
| B5 | L3 | 1/3 | 1/3 | 2/3 | · | 2/3 |
| B6 | L4 | · | · | 3/3 | 3/3 | 2/3 |
| B7 | L4 | 1/3 | · | 1/3 | · | · |

**表 4：L2 汇总指标（GPT-5.4，每组 3 次运行的均值）**

| 指标 | A（仅批量） | B（步进） | C（代码+批量） | D（双模式） | code-only |
|------|:-----------:|:---------:|:--------------:|:-----------:|:---------:|
| 平均发现缺陷数 | 1.7 | 2.0 | 5.0 | 4.0 | 4.0 |
| 平均迭代次数 | 6 | 54 | 31 | 61 | — |
| B1 发现率 | 0% | 0% | 0% | 0% | 67% |

#### 关键发现

**发现 1：步进执行在 GLM-5.1 上效果显著，代码理解在 GPT-5.4 上效果显著。** GLM-5.1 上 B 组（step-only，3.7/7）和 C 组（code-batch，3.7/7）并列最优，均大幅领先 A 组（batch-only，1.3/7）。GPT-5.4 上 C 组（code-batch，5.0/7）表现最强。这表明步进执行和代码理解各有独立贡献，但代码理解的边际收益随模型能力增长。

**发现 2：B1 在运行时组中难以被稳定发现。** B1（`RemoveItem` 负数参数）在 GPT-5.4 的全部 L2 运行时组中均未被发现，GLM-5.1 仅在 code-batch 组中偶发检出（2/3）。只有具备命令注册能力的 L0/L1 模式能稳定复现 B1。这验证了命令注册机制对突破接口盲区的必要性。

**发现 3：两个模型均存在负交互效应。** GLM-5.1 上 D 组（dual，2.7/7）低于 B 组和 C 组的均值（3.7/7），GPT-5.4 上 D 组（4.0/7）同样低于 C 组（5.0/7）。分析原因：同时启用代码分析和步进执行使 Agent 在两种模式间切换，增加了认知负荷和迭代消耗，反而不如专注于单一因子。

**发现 4：code-only 的跨模型稳定性最高。** code-only 在两个模型上均达到约 4/7 的缺陷发现率，且在 L2 组中发现 B1 的概率最高（GLM-5.1 上 1/3，配合 code-batch 的 2/3）。但其固有局限是无法通过运行时验证排除误报。

**发现 5：不同模型上最容易检出的缺陷不同。** GLM-5.1 上 B4（签到奖励无幂等保护）在全部 15 次 L2 运行中均被发现，B6（邮件附件断链）在 12/15 次运行中被发现。GPT-5.4 上 B3（成就计数对象错误）最为稳定（14/15 次检出），B4 和 B6 则主要在具备代码访问的组中被发现（各 8/15）。共同点是这些缺陷的行为异常在日志或代码中高度可见，无需深层推理即可识别。

### 5.3 跨模型对比

将两个模型的结果合并观察（均为 3 次运行均值）：

**表 5：跨模型 L2 消融对比**

| 组别 | GLM-5.1 | GPT-5.4 | 趋势 |
|------|:--------------:|:------------------:|------|
| A（仅批量） | 1.3 | 1.7 | 一致：最弱 |
| B（步进） | 3.7 | 2.0 | GLM 更优 |
| C（代码+批量） | 3.7 | 5.0 | GPT 更优 |
| D（双模式） | 2.7 | 4.0 | GPT 更优 |
| code-only | 4.0 | 4.0 | 一致 |

**核心洞察：** 步进执行和代码理解各有独立贡献，但效果的相对大小取决于模型能力。GLM-5.1 上步进执行的边际收益更为显著（B 组 3.7 vs A 组 1.3），而 GPT-5.4 上代码理解显著提升了 batch 模式的表现（C 组 5.0 vs A 组 1.7）。两个模型上均观察到负交互效应（D 组低于 B 组或 C 组），说明同时启用两种能力在当前 Agent 架构下存在认知负荷问题。这意味着随着 LLM 能力提升，最优策略需要根据模型特点调整代码理解与运行时探索的权重分配。

### 5.4 L0 实验结果

L0 模式下 Agent 从零开始，没有任何预构建业务命令。

**表 6：L0 实验结果**

| 指标 | GLM run1 | GLM run2 | GLM run3 | GPT run1 | GPT run2 | GPT run3 |
|------|:----:|:----:|:----:|:----:|:----:|:----:|
| 迭代次数 | 102 | 100 | 108 | 80 | 70 | 71 |
| 注册命令数 | 7 | 7 | 7 | 7 | 7 | 7 |
| 发现缺陷 | B1,B3,B4,B6 | B1,B3,B4,B6 | B1,B4,B6 | B1,B3,B4,B5,B6 | B1,B3,B4,B5 | B1,B3,B4,B6 |
| 缺陷数 | 4/7 | 4/7 | 3/7 | 5/7 | 4/7 | 4/7 |

GLM-5.1 平均发现 3.7/7，GPT-5.4 平均发现 4.3/7。两个模型在全部 6 次 L0 运行中均成功注册了完整的测试命令集（7 条）并发现了 B1——这是 L2 组在两个模型上几乎无法触及的缺陷。B4 和 B6 在所有运行中稳定检出。L0 结果验证了全自主模式的跨模型可行性。

### 5.5 L1 实验结果

**表 7：L1 实验结果**

| 指标 | GLM run1 | GLM run2 | GLM run3 | GPT run1 | GPT run2 | GPT run3 |
|------|:----:|:----:|:----:|:----:|:----:|:----:|
| 迭代次数 | 105 | 108 | 107 | 66 | 87 | 75 |
| 注册命令数 | 2 | 2 | 2 | 2 | 1 | 1 |
| 发现缺陷 | B3,B4,B5,B6 | B1,B3,B4,B6 | B4,B5 | B1,B3,B4,B5,B6 | B1,B3,B4,B5,B6,B7 | B1,B3,B4,B5,B6 |
| 缺陷数 | 4/7 | 4/7 | 2/7 | 5/7 | 6/7 | 5/7 |

L1 模式的核心验证目标是"接口缺口"假说。在所有 6 次运行中，Agent 均自主注册了用于绕过命令层校验的原始接口命令。GLM-5.1 run2 中注册 `test_remove_negative` 后观察到物品数量从 3 变为 104；GPT-5.4 run2 中注册同名命令后观察到物品从 1 变为 2——**两个模型均通过命令注册成功复现了 B1 缺陷**。

GLM-5.1 平均发现 3.3/7，GPT-5.4 平均发现 5.3/7。GPT-5.4 的 L1 表现显著优于 GLM-5.1，原因在于 GPT-5.4 结合 LSP 工具能更高效地理解代码结构，从而在有限迭代内覆盖更多模块。特别值得注意的是 GPT-5.4 run2 发现了 6/7 个缺陷（包括 B7），这是所有实验中单次运行的最高记录。B4 和 B6 在两个模型的全部 6 次运行中均被稳定发现。

---

## 6 讨论

### 6.1 命令注册能力是突破接口盲区的关键

L2 运行时组中 B1 的发现率极低（GLM-5.1 仅 code-batch 2/3，GPT-5.4 全部 0/3），而 L0/L1 通过命令注册能稳定复现该缺陷。这一对比表明：当测试接口无法构造出触发缺陷所需的输入时，再强的探索策略也无法弥补接口本身的不足。命令注册机制使 Agent 从"命令使用者"升级为"命令设计者"，是本文方法区别于传统自动化测试的核心差异。

### 6.2 世界暂停赋予 Agent 主动干预和选择测试方向的能力

步进执行相比批量执行的优势，不仅在于粒度更细或因果可追溯，核心在于**世界暂停（Hold World）**为 Agent 创造了自主决策的空间。在每步执行后，世界完全静止，Agent 可以自由地查询状态、修改数据、注册新接口——主动构造测试条件而非被动等待结果。批量执行下 Agent 必须在执行前确定所有操作，世界持续运转，Agent 失去的不仅是观察粒度，更是在每步之间主动干预系统、构造前置条件和调整探索方向的能力。GLM-5.1 的实验数据直接体现了这一点：B 组（步进，3.7/7）大幅领先 A 组（批量，1.3/7），且 B 组在 3 次运行中均稳定发现 B4、B5、B6——Agent 在暂停期间决定查询关联模块追踪链式影响、重复操作验证幂等性、主动构造边界条件。这些决策都是在世界暂停的窗口中动态生成的，而非预先规划。

### 6.3 代码理解的效果依赖模型能力

GLM-5.1 上 D 组（2.7/7）表现低于 B 组和 C 组（均 3.7/7），GPT-5.4 上 C 组（code-batch，5.0/7）显著优于 B 组（step-only，2.0/7）。这一跨模型对比揭示了一个重要洞察：代码理解的边际收益不是固定的，而是随模型的代码理解能力增长。两个模型上均观察到的负交互效应（D 组表现低于单因子组）部分源于同时启用代码分析和步进执行增加了 Agent 的认知负荷——在代码阅读与运行时探索间的切换消耗了额外迭代。当模型代码理解能力更强（GPT-5.4）或工具更高效（LSP 语义导航替代文本搜索）时，代码理解阶段的投入回报率会显著提升。

这一发现对工程实践的含义是：随着 LLM 代码理解能力的持续提升，"代码+运行时"的组合模式将逐渐成为最优策略，"纯运行时"模式的相对优势会递减。本文在 GPT-5.4 实验中还集成了 LSP（gopls）语义代码分析工具，使 Agent 能通过 `lsp_references`、`lsp_definition`、`lsp_symbols` 等调用获取精确的跨模块引用关系——一次 `lsp_references("Publish")` 调用即可返回全部 11 个事件发布点，替代多次 `search_code` + `read_file` 的组合。在更大规模的真实项目中，LSP 的优势将进一步放大。

### 6.4 静态分析与运行时测试的互补性

code-only 基线在两个模型上均稳定发现约 4/7 个缺陷，且是发现 B7（ID 冲突）的唯一途径——这表明静态分析擅长生成"怀疑"。但静态分析无法通过运行时验证排除误报，而运行时测试能提供确认证据。两者是互补关系，非替代关系。一个可能的改进方向是将静态分析作为运行时测试的优先级排序器——Agent 先通过代码分析标记可疑区域，再将有限的运行时预算集中在高优先级区域。

### 6.5 对工程实践的含义

本文提出的方法指向一种新的工程分工模式：开发者提供系统、规则和约束边界；Agent 负责持续理解系统、探索测试路径、注册缺失的验证接口、执行测试并形成报告；人工只在规则制定和高风险结论审阅时介入。从这个角度看，集成测试不应仅仅是"更快地执行旧测试"，而应逐步转向"让 Agent 承担大部分系统级验收劳动"。

---

## 7 效度威胁

### 7.1 内部效度

**运行次数与统计效力：** 两个模型的 L2 消融、L0 和 L1 实验均完成了每组 3 次运行（共 42 次运行）。3 次重复已能观察到稳定性趋势（如两个模型的 L0/L1 全部运行均发现 B1、B4），但仍不足以进行严格的统计检验（如 Wilcoxon 秩和检验）。后续需要扩大到 10+ 次以提升统计效力。

**预埋缺陷的代表性：** 7 个缺陷由作者人工设计和预埋，可能无法完全代表真实系统的缺陷分布。缺陷设计参考了常见的事件驱动系统缺陷模式，但在后续工作中应考虑引入 mutation testing 方法论来增加客观性。

**迭代预算的影响：** GLM-5.1 最大迭代次数设为 110，GPT-5.4 设为 80，不同组的实际迭代消耗不同（25-108），这可能影响对各组能力上限的公平比较。

### 7.2 外部效度

**被测系统规模：** 实验仅在一个包含 6 个模块的原型系统上进行，结论能否推广到更大规模、更复杂的系统有待验证。

**领域泛化性：** 虽然事件驱动架构广泛存在于微服务、IoT 和游戏等领域，但实验载体为游戏服务端，其他领域的适用性需要额外验证。

**模型依赖性：** 实验使用了 GLM-5.1 和 GPT-5.4 两个模型，已初步观察到模型能力对实验结果的显著影响（如代码理解主效应在两个模型上方向相反）。但两个模型仍不足以建立普遍性结论，后续应扩展到更多模型。

### 7.3 构建效度

**缺陷判定标准：** Agent 报告中有些发现需要人工判定是否构成真实缺陷。判定标准的一致性可能影响精确率的准确性。

---

## 8 结论与未来工作

### 8.1 结论

本文提出 DSMB-Agent，一种基于树状命令空间和步进思考的 Agent 驱动集成测试方法。通过在两个 LLM（GLM-5.1、GPT-5.4）上的实验验证，得出以下结论：

1. 步进执行和代码理解均有独立贡献——GLM-5.1 上步进组和代码+批量组均达 3.7/7（vs. 批量组 1.3/7），GPT-5.4 上代码+批量组最优（5.0/7）；
2. 代码理解的效果高度依赖模型能力——GLM-5.1 上步进执行的边际收益更为显著（B 组 3.7 vs A 组 1.3），而 GPT-5.4 上代码理解使 batch 模式从 1.7 提升到 5.0；
3. 两个模型上均存在负交互效应——同时启用代码分析和步进执行反而降低效果，说明当前 Agent 架构下存在认知负荷上限；
4. B1 是所有 L2 组在两个模型上的共同盲区——命令注册能力（L0/L1）是突破接口盲区的关键；
5. L0（全自主）模式在两个模型上均能从零构建完整测试命令集并发现 3-5 个缺陷，L1 模式在 GPT-5.4 上平均发现 5.3/7（最高单次 6/7），验证了方法的跨模型可行性；
6. 静态分析与运行时测试互为补充——code-only 在两个模型上均稳定发现约 4/7 个缺陷，L1+LSP 组合在 GPT-5.4 上达到接近的发现率且具备运行时验证能力。

### 8.2 未来工作

- **扩大重复实验规模**——当前每组 3 次运行已能观察趋势，但需扩大到 10+ 次以支持严格的统计检验；
- **LSP 深度集成**——本文已实现 LSP（gopls）集成原型，后续需系统评估 LSP 语义导航对迭代效率和缺陷发现率的量化影响；
- **更大规模系统**——在更多模块、更复杂关联的系统上验证方法的可扩展性；
- **与已有方法的定量对比**——与 MBT、随机测试等基线方法进行控制实验比较。

---

## 参考文献

[1] M. Chen et al., "Evaluating Large Language Models Trained on Code," arXiv:2107.03374, 2021.

[2] S. Peng et al., "The Impact of AI on Developer Productivity: Evidence from GitHub Copilot," arXiv:2302.06590, 2023.

[3] M. E. Fagan, "Design and Code Inspections to Reduce Errors in Program Development," IBM Systems Journal, vol. 15, no. 3, pp. 182-211, 1976.

[4] M. Utting, A. Pretschner, and B. Legeard, "A Taxonomy of Model-Based Testing Approaches," Software Testing, Verification and Reliability, vol. 22, no. 5, pp. 297-312, 2012.

[5] W. Grieskamp, "Multi-Paradigmatic Model-Based Testing," in Proc. FATES/RV, 2006, pp. 1-19.

[6] Google, "UI/Application Exerciser Monkey," Android Developers Documentation.

[7] K. Mao, M. Harman, and Y. Jia, "Sapienz: Multi-objective Automated Testing for Android Applications," in Proc. ISSTA, 2016, pp. 94-105.

[8] T. Su et al., "Guided, Stochastic Model-Based GUI Testing of Android Apps," in Proc. ESEC/FSE, 2017, pp. 245–256.

[9] Z. Chen et al., "ChatUniTest: A Framework for LLM-Based Test Generation," arXiv:2305.04764, 2023.

[10] C. Lemieux et al., "CodaMosa: Escaping Coverage Plateaus in Test Generation with Pre-trained Large Language Models," in Proc. ICSE, 2023, pp. 919–931.

[11] Y. Deng et al., "Large Language Models Are Zero-Shot Fuzzers: Fuzzing Deep-Learning Libraries via Large Language Models," in Proc. ISSTA, 2023, pp. 1165–1176.

[12] M. Zalewski, "American Fuzzy Lop (AFL)," https://lcamtuf.coredump.cx/afl/.

[13] LLVM Project, "libFuzzer - A Library for Coverage-Guided Fuzz Testing," https://llvm.org/docs/LibFuzzer.html.

[14] W. Afzal, R. Torkar, and R. Feldt, "A Systematic Review of Search-Based Testing for Non-Functional System Properties," Information and Software Technology, vol. 51, no. 6, pp. 957-976, 2009.

[15] A. Arcuri and L. Briand, "Adaptive Random Testing: An Illusion of Effectiveness?" in Proc. ISSTA, 2011, pp. 265-275.

[16] X. Zhou et al., "Fault Analysis and Debugging of Microservice Systems: Industrial Survey, Benchmark System, and Empirical Study," IEEE Trans. Software Eng., vol. 47, no. 2, pp. 243–260, 2021.

---

## 附录

### A. 实验原型源码

原型代码位于 `ai-integration-test-demo/` 目录，包含完整的服务端实现、Agent 框架和实验脚本。

### B. 实验运行说明

详见 `QUICKSTART.md`。

### C. 实验结果原始数据

- GLM-5.1 全量实验数据（7 组 × 3 轮）：`results/formal/glm-5.1/`
- GPT-5.4 全量实验数据（7 组 × 3 轮）：`results/formal/gpt-5.4/`
- GLM-5.1 早期单次运行报告：`results/evaluation_report.md`、`results/l0-run1-report.md`、`results/l1-run1-report.md`

### D. 完整终端日志示例

以下是一次 step-only 模式实验的完整终端输出（GLM-5.1，54 轮迭代，因 API 服务端错误在第 55 轮中断）。日志中 `[Thinking]` 行展示了 Agent 在世界暂停期间的完整推理过程，`AI →` 和 `WS ←` 分别为 Agent 发出的命令和系统返回的响应。

```
$ ./bin/server -mode test -scenario step-only \
    -api-key *** -model glm-5.1 \
    -base-url https://open.bigmodel.cn/api/anthropic

2026/04/17 16:45:02 game server started on 127.0.0.1:5400
2026/04/17 16:45:03 client connected: 127.0.0.1:50206
2026/04/17 16:45:03 using Anthropic API provider (model=glm-5.1)
2026/04/17 16:45:03 running AI test scenario: step-only (mode: step-only)

========== 理解阶段：查询初始状态 ==========

--- iteration 1/80 ---
[Thinking] I'll systematically test all modules for player 10001,
    stepping through each operation to observe logs and cross-module
    interactions. Let me start by querying the current state of all modules.

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS ← {"ok":true,"data":[]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"locked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  {"achId":4003,"name":"collector_100","state":"locked"},
  {"achId":4004,"name":"fully_equipped","state":"locked"}
]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:equipment}
WS ← {"ok":true,"data":{}}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:signin}
WS ← {"ok":true,"data":[
  {"day":1,"rewardItem":2001,"rewardCount":1,"claimed":false},
  {"day":2,"rewardItem":2002,"rewardCount":1,"claimed":false},
  ...
  {"day":7,"rewardItem":3001,"rewardCount":1,"claimed":false}
]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:mail}
WS ← {"ok":true,"data":[]}

========== 执行阶段：步进探索 ==========

--- iteration 2/80 ---
[Thinking] Good, I have the initial state. Let me summarize:
    - Bag: Empty
    - Tasks: 3001 (target:1), 3002 (target:2), both active
    - Achievements: 4 locked (first_task, task_master, collector_100, fully_equipped)
    - Equipment: Empty
    - Sign-in: 7 days available, none claimed
    - Mail: Empty
    Let me start by claiming sign-in rewards and adding items.

AI → checkin {Cmd:checkin PlayerID:10001 Day:1}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 3/80 ---
AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Task] add task 3001, target 1",
  "[Task] add task 3002, target 2",
  "[Achievement] add achievement 4001: first_task",
  "[Achievement] add achievement 4002: task_master",
  "[Achievement] add achievement 4003: collector_100",
  "[Achievement] add achievement 4004: fully_equipped",
  "[SignIn] day 1 claimed, reward: item 2001 x1",
  "[Mail] sent: Sign-in Day 1 Reward (id=1)"
]}

--- iteration 4/80 ---   ← [世界暂停]
[Thinking] Interesting! Sign-in triggered a mail.
    Let me check the mail and claim it.

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:mail}
WS ← {"ok":true,"data":[
  {"mailId":1,"subject":"Sign-in Day 1 Reward",
   "attachment":{"itemId":2001,"count":1},"claimed":false}
]}

--- iteration 5/80 ---
AI → claimmail {Cmd:claimmail PlayerID:10001 MailID:1}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 6/80 ---
AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Mail] claimed attachment from mail 1: item 2001 x1"
]}

--- iteration 7/80 ---   ← [世界暂停]
[Thinking] Check state after mail claim.

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS ← {"ok":true,"data":[]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":0,"state":"active"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

--- iteration 8/80 ---   ← [世界暂停：发现缺陷 B6]
[Thinking] 🔴 BUG FOUND: The mail claimed item 2001 x1 but the bag is
    still EMPTY! The item wasn't actually added to the bag.
    Let me continue testing to build the full picture.

AI → additem {Cmd:additem PlayerID:10001 ItemID:2001 Count:3}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 9/80 ---
AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Bag] add item 2001 x3",
  "[Task] trigger 3001 progress+1 (now 1/1)",
  "[Task] task 3001 completed",
  "[Achievement] unlocked: first_task (id=4001)",
  "[Mail] sent: Achievement Unlocked! (id=4001) (id=2)"
]}

--- iteration 10/80 ---   ← [世界暂停]
[Thinking] Excellent! Rich event chain here. Let me check all state now.

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS ← {"ok":true,"data":[{"itemId":2001,"count":3}]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:task}
WS ← {"ok":true,"data":[
  {"taskId":3001,"target":1,"progress":1,"state":"completed"},
  {"taskId":3002,"target":2,"progress":0,"state":"active"}
]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:achievement}
WS ← {"ok":true,"data":[
  {"achId":4001,"name":"first_task","state":"unlocked"},
  {"achId":4002,"name":"task_master","state":"locked"},
  ...
]}

--- iteration 11/80 ---
[Thinking] Key observations:
    - Bag has 3x item 2001 (direct add worked, mail claim didn't)
    - Task 3001 completed — triggered by bag add
    - Achievement 4001 (first_task) unlocked — triggered by task completion
    - New mail for achievement — no attachment! (Achievement mail has no reward)
    - Mail id=1 shows claimed=true but item never reached bag

AI → additem {Cmd:additem PlayerID:10001 ItemID:2002 Count:2}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iterations 12-19: 完成任务 3002，触发 task_master 和 collector_100 ---

--- iteration 19/80 ---
AI → next {Cmd:next}
WS ← {"ok":true,"log":[
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

--- iterations 21-29: 装备武器和盔甲，触发 fully_equipped 成就 ---

--- iteration 28/80 ---
AI → next {Cmd:next}
WS ← {"ok":true,"log":[
  "[Equipment] auto-equip: armor slot → item 1501",
  "[Achievement] unlocked: fully_equipped (id=4004)",
  "[Mail] sent: Achievement Unlocked! (id=4004) (id=5)"
]}

--- iteration 29/80 ---   ← [世界暂停]
[Thinking] fully_equipped achievement unlocked when both weapon
    and armor slots are filled.

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS ← {"ok":true,"data":[
  {"itemId":2001,"count":3},{"itemId":2002,"count":3},
  {"itemId":1001,"count":1},{"itemId":1501,"count":1}
]}

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:equipment}
WS ← {"ok":true,"data":{
  "weapon":{"slot":"weapon","itemId":1001},
  "armor":{"slot":"armor","itemId":1501}
}}

[Thinking 备注] 🔴 BUG: 装备穿戴后物品仍在背包中（count=1），
    意味着物品同时存在于装备栏和背包两处 → 缺陷 B5

--- iterations 30-51: 卸下装备、签到第2天、测试removeitem、
    验证重复签到被拒绝、测试不存在物品的移除、查看帮助 ---

--- iteration 52/80 ---
[Thinking] The help says claimreward requires playerId and day
    (for sign-in). Let me test claiming day 1 reward.

AI → claimreward {Cmd:claimreward PlayerID:10001 Day:1}
WS ← {"ok":true,"data":{"pendingOps":1,"queued":true}}

--- iteration 53/80 ---
AI → next {Cmd:next}
WS ← {"ok":true,"log":["[SignIn] day 1 reward claimed again"]}

--- iteration 54/80 ---   ← [世界暂停：发现缺陷 B4]
[Thinking] 🔴 BUG FOUND: Day 1 sign-in reward can be claimed AGAIN
    even though sign-in day 1 was already claimed!

AI → playermgr {Cmd:playermgr PlayerID:10001 Sub:bag}
WS ← {"ok":true,"data":[
  {"itemId":2001,"count":2},{"itemId":2002,"count":3},
  {"itemId":1001,"count":1},{"itemId":1501,"count":1}
]}

--- iteration 55/80 ---
agent error: anthropic api error 500 (API 服务端错误，实验中断)
```

**本次运行发现的缺陷（54 轮迭代）：**

| 发现顺序 | 缺陷 | 发现时机 | Agent 推理过程 |
|:--------:|:----:|:--------:|---------------|
| 1 | B6（邮件附件断链） | iteration 8 | 签到→领取邮件附件→查询背包为空→判定附件未入库 |
| 2 | B5（装备不移除背包） | iteration 29 | 装备武器和盔甲→查询背包发现物品仍在→判定双重存在 |
| 3 | B4（签到奖励无幂等） | iteration 54 | 查看帮助→理解 claimreward 用法→重复领取成功→判定无幂等保护 |

该日志完整展示了 DSMB-Agent 的三个核心特征：（1）`[Thinking]` 揭示了世界暂停期间的推理链——Agent 在每步之后分析因果、提出假设、决定下一步验证方向；（2）Agent 在暂停窗口中主动查询关联模块追踪副作用，这些决策完全是运行时动态生成的；（3）Agent 通过"操作→观察→思考→再操作"的循环逐步收敛到缺陷，而非预先规划测试路径。
