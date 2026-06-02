ALTER TABLE billing_admission_controls
DROP CONSTRAINT billing_admission_controls_use_class_check;

ALTER TABLE billing_admission_controls
ADD CONSTRAINT billing_admission_controls_use_class_check
CHECK (use_class IN ('chat', 'web_search', 'tool_call', 'batch', 'unknown'));
