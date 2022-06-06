-- Write your migrate up statements here

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE scheduled_jobs (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL UNIQUE,
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
    updated_at TIMESTAMP WITH TIME ZONE 
);

---- create above / drop below ----

DROP TABLE scheduled_jobs;
