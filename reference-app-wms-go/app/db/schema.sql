CREATE TABLE job_sites (
  id UUID PRIMARY KEY,
  name VARCHAR(255),
  location VARCHAR(255),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE schedules (
  id UUID PRIMARY KEY,
  job_site_id UUID NOT NULL REFERENCES job_sites(id),
  start_date TIMESTAMP NOT NULL,
  end_date TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);


CREATE TABLE tasks (
  id UUID PRIMARY KEY,
  schedule_id UUID NOT NULL REFERENCES schedules(id),
  worker_id UUID,  -- assigned worker if applicable
  name VARCHAR(255),
  description TEXT,
  planned_start_time TIMESTAMP,
  planned_end_time TIMESTAMP,
  status VARCHAR(50) NOT NULL DEFAULT 'PENDING', 
    -- possible values: 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'BLOCKED'
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);


CREATE TABLE workers (
  id UUID PRIMARY KEY,
  full_name VARCHAR(255),
  role VARCHAR(100), 
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE worker_assignments (
  id UUID PRIMARY KEY,
  worker_id UUID NOT NULL REFERENCES workers(id),
  schedule_id UUID NOT NULL REFERENCES schedules(id),
  assigned_date DATE,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE worker_attendance (
  worker_id UUID,
  job_site_id UUID NOT NULL REFERENCES job_sites(id),
  date DATE NOT NULL,
  check_in_time TIMESTAMP,
  check_out_time TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (worker_id, date)
);

-- Index for querying attendance by job site and date range
CREATE INDEX idx_worker_attendance_job_site_date ON worker_attendance(job_site_id, date);
