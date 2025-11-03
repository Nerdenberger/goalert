

ALTER TABLE user_notification_rules
ADD COLUMN conditions JSONB;

CREATE INDEX idx_notification_rule_conditions ON user_notification_rules USING gin (conditions);

COMMENT ON COLUMN user_notification_rules.conditions IS 'Optional JSONB field for filtering notifications based on alert metadata. Example: {"metadata": {"priority": {"min": 1, "max": 3}}}. NULL means match all alerts.';


DROP INDEX IF EXISTS idx_notification_rule_conditions;
ALTER TABLE user_notification_rules DROP COLUMN IF EXISTS conditions;
