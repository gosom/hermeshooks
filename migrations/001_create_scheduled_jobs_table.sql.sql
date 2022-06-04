-- Write your migrate up statements here

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE scheduled_jobs (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(32) NOT NULL,
    description VARCHAR(100) NOT NULL,
    url VARCHAR(256) NOT NULL,
    payload TEXT NOT NULL,
    signature VARCHAR(44) NOT NULL,
    run_at TIMESTAMP WITH TIME ZONE NOT NULL,
    retries INTEGER NOT NULL,
    status VARCHAR(10) NOT NULL,
    partition INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

---- create above / drop below ----

DROP TABLE scheduled_jobs;
