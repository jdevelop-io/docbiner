-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS templates;
DROP TABLE IF EXISTS usage_monthly;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS plans;

-- Drop extensions
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
