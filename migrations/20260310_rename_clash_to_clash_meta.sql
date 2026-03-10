-- 将 config_policies 表中 target 字段的 clash 类型重命名为 clash-meta

-- 更新现有数据
UPDATE config_policies SET target = 'clash-meta' WHERE target = 'clash';

-- 重建 CHECK 约束
ALTER TABLE config_policies DROP CONSTRAINT config_policies_target_check;
ALTER TABLE config_policies ADD CONSTRAINT config_policies_target_check
    CHECK (target IN ('clash-meta', 'stash', 'surge'));
