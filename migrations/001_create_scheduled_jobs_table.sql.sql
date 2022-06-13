-- Write your migrate up statements here

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    api_key VARCHAR(44) DEFAULT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE scheduled_jobs (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
    user_id INT NOT NULL,
    name VARCHAR(32) NOT NULL,
    description VARCHAR(100) NOT NULL,
    url VARCHAR(256) NOT NULL,
    payload TEXT NOT NULL,
    content_type VARCHAR(32) NOT NULL,
    signature VARCHAR(64) NOT NULL,
    run_at TIMESTAMP WITH TIME ZONE NOT NULL,
    retries INTEGER NOT NULL,
    status INT NOT NULL,
    partition INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_users
      FOREIGN KEY(user_id) 
	  REFERENCES users(id)
);

CREATE INDEX idx_user_id ON scheduled_jobs(user_id);

CREATE TABLE executions (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scheduled_job_id INT NOT NULL,
    status_code INT NOT NULL,
    msg VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT fk_scheduled_job
      FOREIGN KEY(scheduled_job_id) 
	  REFERENCES scheduled_jobs(id)
);

CREATE INDEX idx_scheduled_job_id ON executions(scheduled_job_id);

---- create above / drop below ----

DROP INDEX idx_user_id;
DROP INDEX idx_scheduled_job_id;

DROP TABLE scheduled_jobs;
DROP TABLE users;
