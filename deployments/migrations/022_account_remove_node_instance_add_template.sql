-- 022: 账号移除 node_id，实例添加 template_id
-- 账号不再绑定节点，Volume 归档存储在 MinIO 共享存储中
-- 实例支持关联模板创建

ALTER TABLE instances ADD COLUMN template_id VARCHAR(100);
