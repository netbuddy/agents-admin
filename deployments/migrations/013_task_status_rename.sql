-- 013: 将 Task.status 从 'running' 重命名为 'in_progress'
--
-- 背景：
--   Task（任务）和 Run（执行）都有"正在进行"的状态：
--   - Task: running（改为 in_progress）
--   - Run: running（保持不变）
--   为避免混淆，将 Task 的 running 改为 in_progress。
--
-- 术语规范：
--   - Task = 任务（定义"做什么"）
--   - Run = 执行（记录"一次尝试"）

-- 更新现有数据
UPDATE tasks SET status = 'in_progress' WHERE status = 'running';

-- 注意：如果使用 PostgreSQL ENUM 类型，需要额外步骤：
-- 1. ALTER TYPE task_status ADD VALUE 'in_progress';
-- 2. UPDATE tasks SET status = 'in_progress' WHERE status = 'running';
-- 3. 需要创建新类型并替换旧类型才能移除 'running' 值
-- 本项目使用 VARCHAR 存储状态，无需上述操作
