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

DROP INDEX idx_scheduled_job_id;
DROP TABLE executions;

