-- 020: 节点主机信息字段
-- 为节点卡片显示主机名和 IP 地址，将这些信息从 capacity JSON 提升为一级字段

ALTER TABLE nodes ADD COLUMN IF NOT EXISTS hostname VARCHAR(255) DEFAULT '';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS ips TEXT DEFAULT '';
