/**
 * Agent 类型 → 支持的模型列表映射
 *
 * 每种 Agent 类型有一组可选模型，创建/编辑模板时根据选定的类型动态渲染模型下拉列表。
 * 新增类型或模型时只需修改此文件。
 */

export interface ModelOption {
  /** 模型标识（存入数据库） */
  value: string
  /** 显示名称 */
  label: string
  /** 简要说明 */
  description?: string
}

export interface AgentTypeConfig {
  /** 类型标识 */
  id: string
  /** 显示名称 */
  label: string
  /** 支持的模型列表 */
  models: ModelOption[]
  /** 默认模型 */
  defaultModel: string
  /** 默认温度 */
  defaultTemperature: number
}

export const AGENT_TYPE_CONFIGS: Record<string, AgentTypeConfig> = {
  claude: {
    id: 'claude',
    label: 'Claude',
    models: [
      { value: 'claude-sonnet-4-20250514', label: 'Claude Sonnet 4', description: '平衡性能与速度' },
      { value: 'claude-opus-4-20250514', label: 'Claude Opus 4', description: '最强推理能力' },
      { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet', description: '高性价比' },
      { value: 'claude-3-5-haiku-20241022', label: 'Claude 3.5 Haiku', description: '快速响应' },
    ],
    defaultModel: 'claude-sonnet-4-20250514',
    defaultTemperature: 0.3,
  },
  gemini: {
    id: 'gemini',
    label: 'Gemini',
    models: [
      { value: 'gemini-2.5-pro', label: 'Gemini 2.5 Pro', description: '最新旗舰模型' },
      { value: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash', description: '快速且高效' },
      { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash', description: '上一代快速模型' },
      { value: 'gemini-pro', label: 'Gemini Pro', description: '经典版本' },
    ],
    defaultModel: 'gemini-2.5-pro',
    defaultTemperature: 0.5,
  },
  qwen: {
    id: 'qwen',
    label: 'Qwen',
    models: [
      { value: 'qwen3-235b-a22b', label: 'Qwen3 235B', description: 'MoE 旗舰模型' },
      { value: 'qwen3-32b', label: 'Qwen3 32B', description: '高性能密集模型' },
      { value: 'qwen3-14b', label: 'Qwen3 14B', description: '中等规模' },
      { value: 'qwen3-8b', label: 'Qwen3 8B', description: '轻量高效' },
      { value: 'qwen-coder-plus', label: 'Qwen Coder Plus', description: '代码专用模型' },
    ],
    defaultModel: 'qwen3-32b',
    defaultTemperature: 0.7,
  },
  codex: {
    id: 'codex',
    label: 'Codex',
    models: [
      { value: 'codex-mini', label: 'Codex Mini', description: 'OpenAI 编程代理' },
      { value: 'o4-mini', label: 'o4-mini', description: '推理型编程模型' },
      { value: 'gpt-4.1', label: 'GPT-4.1', description: '通用编程模型' },
    ],
    defaultModel: 'codex-mini',
    defaultTemperature: 0.3,
  },
  custom: {
    id: 'custom',
    label: '自定义',
    models: [],
    defaultModel: '',
    defaultTemperature: 0.7,
  },
}

/** 获取类型配置，不存在则返回 custom */
export function getAgentTypeConfig(type: string): AgentTypeConfig {
  return AGENT_TYPE_CONFIGS[type] || AGENT_TYPE_CONFIGS.custom
}

/** 获取所有类型的选项列表 */
export function getAgentTypeOptions(): { value: string; label: string }[] {
  return Object.values(AGENT_TYPE_CONFIGS).map(c => ({ value: c.id, label: c.label }))
}
