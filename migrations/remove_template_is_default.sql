ALTER TABLE templates
    DROP COLUMN IF EXISTS is_default;

DROP INDEX IF EXISTS idx_templates_default;

COMMENT ON COLUMN config_policies.template_name IS '使用的规则模板，为空则使用内置模板';
