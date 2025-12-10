-- QuietHire ClickHouse Schema
-- Job storage with deduplication and efficient querying

-- Main jobs table using ReplacingMergeTree for deduplication
CREATE TABLE IF NOT EXISTS jobs (
    -- Primary identifiers
    id String,
    job_hash String,  -- Hash of (url + title + company) for deduplication
    
    -- Job details
    title String,
    company String,
    description String,
    location String,
    remote UInt8,  -- 0 = no, 1 = yes
    
    -- Salary information
    salary_min Nullable(Int32),
    salary_max Nullable(Int32),
    currency Nullable(String),
    
    -- Job metadata
    job_type String,  -- full-time, part-time, contract, etc.
    experience_level Nullable(String),  -- entry, mid, senior, etc.
    
    -- Authenticity scoring
    real_score Int32,  -- 0-100 authenticity score
    
    -- Hiring manager information
    hiring_manager_name Nullable(String),
    hiring_manager_email Nullable(String),
    
    -- Source information
    source_url String,
    source_platform String,  -- indeed, linkedin, greenhouse, etc.
    
    -- Skills/tags
    tags Array(String),
    
    -- Timestamps
    posted_at DateTime,
    updated_at DateTime,
    crawled_at DateTime DEFAULT now(),
    
    -- Version for deduplication (higher version = newer)
    version UInt64 DEFAULT 1
    
) ENGINE = ReplacingMergeTree(version)
PARTITION BY toYYYYMM(posted_at)
ORDER BY (company, location, posted_at, id)
SETTINGS index_granularity = 8192;

-- Table for raw HTML storage (for reprocessing if needed)
CREATE TABLE IF NOT EXISTS jobs_raw_html (
    id String,
    url String,
    html String,  -- Compressed HTML content
    crawled_at DateTime DEFAULT now(),
    status String,  -- success, failed, pending
    error_message Nullable(String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(crawled_at)
ORDER BY (crawled_at, id)
SETTINGS index_granularity = 8192;

-- Table for tracking crawl history
CREATE TABLE IF NOT EXISTS crawl_history (
    crawl_id String,
    source_platform String,
    start_time DateTime,
    end_time Nullable(DateTime),
    total_jobs_found Int32,
    jobs_inserted Int32,
    jobs_updated Int32,
    jobs_failed Int32,
    status String,  -- running, completed, failed
    error_message Nullable(String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (start_time, crawl_id)
SETTINGS index_granularity = 8192;

-- Materialized view for active jobs (real_score >= 70, posted within last 90 days)
CREATE MATERIALIZED VIEW IF NOT EXISTS jobs_active
ENGINE = MergeTree()
PARTITION BY toYYYYMM(posted_at)
ORDER BY (company, location, posted_at)
AS SELECT 
    id,
    job_hash,
    title,
    company,
    description,
    location,
    remote,
    salary_min,
    salary_max,
    currency,
    job_type,
    experience_level,
    real_score,
    hiring_manager_name,
    hiring_manager_email,
    source_url,
    source_platform,
    tags,
    posted_at,
    updated_at,
    crawled_at
FROM jobs
WHERE real_score >= 70 
  AND posted_at >= now() - INTERVAL 90 DAY;

-- Table for deduplication tracking
CREATE TABLE IF NOT EXISTS job_duplicates (
    job_hash String,
    duplicate_urls Array(String),
    first_seen DateTime,
    last_seen DateTime,
    occurrence_count Int32
) ENGINE = SummingMergeTree(occurrence_count)
ORDER BY (job_hash, first_seen)
SETTINGS index_granularity = 8192;

-- Statistics table for monitoring
CREATE TABLE IF NOT EXISTS job_stats (
    stat_date Date,
    source_platform String,
    total_jobs Int64,
    active_jobs Int64,
    avg_real_score Float32,
    jobs_with_manager Int64
) ENGINE = SummingMergeTree()
ORDER BY (stat_date, source_platform)
SETTINGS index_granularity = 8192;

-- Create indexes for faster querying
CREATE INDEX IF NOT EXISTS idx_job_title ON jobs (title) TYPE tokenbf_v1(10240, 3, 0);
CREATE INDEX IF NOT EXISTS idx_job_company ON jobs (company) TYPE tokenbf_v1(10240, 3, 0);
CREATE INDEX IF NOT EXISTS idx_job_location ON jobs (location) TYPE tokenbf_v1(10240, 3, 0);
CREATE INDEX IF NOT EXISTS idx_job_real_score ON jobs (real_score) TYPE minmax;
