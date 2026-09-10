INSERT INTO accounts (code, name, parent_code, is_active)
VALUES ('33311', '- Thuế GTGT đầu ra phải nộp', '3331', TRUE)
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, parent_code = EXCLUDED.parent_code, is_active = TRUE;
