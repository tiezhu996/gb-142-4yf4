-- Schema is migrated by GORM at application startup. This file verifies database initialization.
CREATE TABLE IF NOT EXISTS schema_bootstrap (
  id INT PRIMARY KEY AUTO_INCREMENT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
