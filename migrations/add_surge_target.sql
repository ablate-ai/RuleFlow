-- 放开 config_policies.target 约束，增加 surge 支持
ALTER TABLE config_policies
  DROP CONSTRAINT config_policies_target_check;

ALTER TABLE config_policies
  ADD CONSTRAINT config_policies_target_check
    CHECK (target IN ('clash', 'stash', 'surge'));
