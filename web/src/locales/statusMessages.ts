export const stateLabels: Record<string, string> = {
  empty: "暂无内容",
  loading: "正在加载",
  stale: "数据可能已过期",
  reconnecting: "正在重新连接",
  unsupported: "当前环境不支持",
  permission: "权限不足",
  quota: "资源配额不足",
  conflict: "状态冲突",
  "partial-failure": "部分操作失败",
  cleanup: "正在清理",
  "terminal-error": "操作失败",
  pending: "等待中",
  queued: "已排队",
  running: "运行中",
  stopped: "已停止",
  completed: "已完成",
  failed: "失败",
  cancelled: "已取消",
  starting: "正在启动",
  stopping: "正在停止",
};

export function localizeState(value: string): string {
  return stateLabels[value.toLowerCase()] ?? value;
}

export function problemContext(code?: string): string {
  if (!code) return "操作未成功，请检查详细信息后重试。";
  if (/timeout/i.test(code)) return "操作超时，请检查运行状态和网络连接后重试。";
  if (/conflict|revision/i.test(code)) return "资源已被其他操作更新，请刷新状态后重试。";
  if (/not.?found/i.test(code)) return "目标资源不存在或已被删除，请刷新列表。";
  return "操作未成功；下方保留原始错误，便于诊断。";
}

