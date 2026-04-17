# DSMB-Agent 实验补完 TODO 备忘录

> 这份备忘录只记录**后续需要补完的实验工作**。当前论文主文档见 [`README.md`](./README.md)。

## 当前已完成

- `register_cmd` 最小 MVP 已实现
- L0 / L1 场景脚手架已接入
- 离线测试通过
- L0 live run 已完成
- L1 live run 已完成，并通过 `register_cmd` 复现 B1
- fresh L2 已拿到部分新证据

## 当前未完成

### 1. fresh L2 完整对照
- [ ] 补完 `dual` 的 final report
- [ ] 补完 `step-only` 的 final report
- [ ] 如资源允许，补跑 `batch-only`
- [ ] 如资源允许，补跑 `code-only`

### 2. 正式比较表
- [ ] 生成 L0 / L1 / L2 对比表
- [ ] 汇总 defect discovery 对比
- [ ] 汇总 correlation recovery 对比
- [ ] 汇总 CMD design evidence（L0 / L1）

### 3. 论文回填
- [ ] 把最终实验表写回 `README.md`
- [ ] 更新摘要里的实验结论
- [ ] 更新结论部分的 strongest claim
- [ ] 增加明确的 limitation / validity note（资源限制、单次运行等）

### 4. 如 API 资源恢复
- [ ] 每组至少再跑 2 次
- [ ] 检查稳定性 / 一致性
- [ ] 形成更像正式论文的结果节

## 当前阻塞

- 继续补跑 fresh L2 时，外部 GLM-5.1 API 返回 `429 Too Many Requests / 余额不足或无可用资源包`
- 因此目前最合理的停点是：
  - 保留已经完成的 L0 / L1 live evidence
  - 保留 fresh L2 partial evidence
  - 等 API 资源恢复后再补完整实验
