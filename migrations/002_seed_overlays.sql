-- Seed data for overlay system
-- Inserts basic domain, capability, and locale overlays

-- Domain overlays
INSERT INTO overlays (name, category, version, config, requires) VALUES
('fintech.pci', 'domain', '1.0.0',
 '{"security_level": "high", "compliance": ["PCI-DSS"], "encryption": "required"}',
 '["capability.audit", "capability.rbac"]'),

('healthcare.hipaa', 'domain', '1.0.0',
 '{"security_level": "high", "compliance": ["HIPAA"], "data_retention": "7_years"}',
 '["capability.audit", "capability.encryption"]'),

('ecommerce.basic', 'domain', '1.0.0',
 '{"features": ["cart", "checkout", "inventory"], "payment_gateways": ["stripe"]}',
 '["capability.auth"]');

-- Capability overlays
INSERT INTO overlays (name, category, version, config, requires) VALUES
('capability.auth', 'capability', '1.0.0',
 '{"methods": ["jwt", "oauth2"], "session_timeout": "24h", "password_policy": "medium"}',
 '[]'),

('capability.rbac', 'capability', '1.0.0',
 '{"roles": ["admin", "user", "viewer"], "permissions": "resource_based"}',
 '["capability.auth"]'),

('capability.audit', 'capability', '1.0.0',
 '{"log_level": "info", "retention": "90_days", "fields": ["user_id", "action", "resource"]}',
 '[]'),

('capability.encryption', 'capability', '1.0.0',
 '{"algorithms": ["AES-256", "RSA-2048"], "key_rotation": "quarterly"}',
 '[]'),

('capability.rate_limiting', 'capability', '1.0.0',
 '{"default_rate": "100/minute", "burst_allowance": 50}',
 '[]');

-- Locale overlays
INSERT INTO overlays (name, category, version, config, requires) VALUES
('locale.us', 'locale', '1.0.0',
 '{"timezone": "America/New_York", "currency": "USD", "date_format": "MM/DD/YYYY"}',
 '[]'),

('locale.eu', 'locale', '1.0.0',
 '{"timezone": "Europe/London", "currency": "EUR", "date_format": "DD/MM/YYYY", "gdpr": true}',
 '["capability.audit"]'),

('locale.uk', 'locale', '1.0.0',
 '{"timezone": "Europe/London", "currency": "GBP", "date_format": "DD/MM/YYYY"}',
 '[]');