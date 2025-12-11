-- PostgreSQL schema for OSINT Discovery Service
-- Stores discovered companies, career pages, and crawl queue

-- Companies table
CREATE TABLE IF NOT EXISTS companies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    description TEXT,
    industry VARCHAR(100),
    source VARCHAR(50) NOT NULL,  -- github, crunchbase, manual, etc.
    metadata JSONB DEFAULT '{}',
    discovered_at TIMESTAMP DEFAULT NOW(),
    last_crawled_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(domain)
);

CREATE INDEX idx_companies_domain ON companies(domain);
CREATE INDEX idx_companies_source ON companies(source);
CREATE INDEX idx_companies_discovered_at ON companies(discovered_at DESC);

-- Career pages / URLs discovered
CREATE TABLE IF NOT EXISTS discovered_urls (
    id SERIAL PRIMARY KEY,
    company_id INTEGER REFERENCES companies(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    url_hash VARCHAR(64) NOT NULL,  -- SHA256 hash for deduplication
    page_type VARCHAR(50),  -- careers, jobs, opportunities, etc.
    confidence FLOAT DEFAULT 0.0,  -- 0.0 to 1.0
    ats_platform VARCHAR(50),  -- greenhouse, ashby, lever, etc.
    discovered_via VARCHAR(50),  -- subdomain, google, sitemap, manual, etc.
    discovered_at TIMESTAMP DEFAULT NOW(),
    last_crawled_at TIMESTAMP,
    crawl_status VARCHAR(20) DEFAULT 'pending',  -- pending, crawling, completed, failed
    crawl_error TEXT,
    priority INTEGER DEFAULT 50,  -- 0-100, higher = more important
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    UNIQUE(url_hash)
);

CREATE INDEX idx_discovered_urls_company_id ON discovered_urls(company_id);
CREATE INDEX idx_discovered_urls_url_hash ON discovered_urls(url_hash);
CREATE INDEX idx_discovered_urls_ats_platform ON discovered_urls(ats_platform);
CREATE INDEX idx_discovered_urls_crawl_status ON discovered_urls(crawl_status);
CREATE INDEX idx_discovered_urls_priority ON discovered_urls(priority DESC);
CREATE INDEX idx_discovered_urls_discovered_at ON discovered_urls(discovered_at DESC);

-- Subdomains discovered
CREATE TABLE IF NOT EXISTS discovered_subdomains (
    id SERIAL PRIMARY KEY,
    company_id INTEGER REFERENCES companies(id) ON DELETE CASCADE,
    subdomain VARCHAR(255) NOT NULL,
    method VARCHAR(20) NOT NULL,  -- dns, crt, manual
    is_job_related BOOLEAN DEFAULT FALSE,
    is_accessible BOOLEAN DEFAULT FALSE,
    discovered_at TIMESTAMP DEFAULT NOW(),
    last_checked_at TIMESTAMP,
    UNIQUE(subdomain)
);

CREATE INDEX idx_discovered_subdomains_company_id ON discovered_subdomains(company_id);
CREATE INDEX idx_discovered_subdomains_is_job_related ON discovered_subdomains(is_job_related);

-- Google dork results
CREATE TABLE IF NOT EXISTS dork_results (
    id SERIAL PRIMARY KEY,
    query TEXT NOT NULL,
    url TEXT NOT NULL,
    title TEXT,
    snippet TEXT,
    rank INTEGER,
    discovered_at TIMESTAMP DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,
    url_hash VARCHAR(64) NOT NULL,
    UNIQUE(url_hash)
);

CREATE INDEX idx_dork_results_query ON dork_results(query);
CREATE INDEX idx_dork_results_processed ON dork_results(processed);
CREATE INDEX idx_dork_results_discovered_at ON dork_results(discovered_at DESC);

-- Crawl queue (optimized for job processing)
CREATE TABLE IF NOT EXISTS crawl_queue (
    id SERIAL PRIMARY KEY,
    url_id INTEGER REFERENCES discovered_urls(id) ON DELETE CASCADE,
    priority INTEGER DEFAULT 50,
    status VARCHAR(20) DEFAULT 'pending',  -- pending, processing, completed, failed
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    assigned_worker VARCHAR(100),  -- Which worker is processing this
    UNIQUE(url_id)
);

CREATE INDEX idx_crawl_queue_status ON crawl_queue(status);
CREATE INDEX idx_crawl_queue_priority ON crawl_queue(priority DESC, created_at ASC);
CREATE INDEX idx_crawl_queue_worker ON crawl_queue(assigned_worker);

-- Discovery campaigns (track bulk discovery operations)
CREATE TABLE IF NOT EXISTS discovery_campaigns (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- company_search, dork_query, subdomain_enum, etc.
    query TEXT,
    status VARCHAR(20) DEFAULT 'running',  -- running, completed, failed
    total_results INTEGER DEFAULT 0,
    companies_found INTEGER DEFAULT 0,
    urls_found INTEGER DEFAULT 0,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_discovery_campaigns_status ON discovery_campaigns(status);
CREATE INDEX idx_discovery_campaigns_started_at ON discovery_campaigns(started_at DESC);

-- ATS platform statistics
CREATE TABLE IF NOT EXISTS ats_statistics (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL,
    total_urls INTEGER DEFAULT 0,
    active_urls INTEGER DEFAULT 0,
    last_updated TIMESTAMP DEFAULT NOW(),
    UNIQUE(platform)
);

-- Function to auto-update URL hash
CREATE OR REPLACE FUNCTION update_url_hash()
RETURNS TRIGGER AS $$
BEGIN
    NEW.url_hash = encode(sha256(NEW.url::bytea), 'hex');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_url_hash
BEFORE INSERT OR UPDATE ON discovered_urls
FOR EACH ROW
EXECUTE FUNCTION update_url_hash();

CREATE TRIGGER trigger_update_dork_url_hash
BEFORE INSERT OR UPDATE ON dork_results
FOR EACH ROW
EXECUTE FUNCTION update_url_hash();

-- View for high-priority crawl targets
CREATE OR REPLACE VIEW high_priority_targets AS
SELECT 
    du.id,
    du.url,
    du.ats_platform,
    du.page_type,
    du.confidence,
    du.priority,
    c.name as company_name,
    c.domain as company_domain,
    cq.status as crawl_status,
    cq.retry_count
FROM discovered_urls du
LEFT JOIN companies c ON du.company_id = c.id
LEFT JOIN crawl_queue cq ON cq.url_id = du.id
WHERE du.is_active = TRUE 
  AND du.crawl_status = 'pending'
  AND (cq.status IS NULL OR cq.status = 'pending')
ORDER BY du.priority DESC, du.discovered_at DESC;

-- View for ATS platform distribution
CREATE OR REPLACE VIEW ats_platform_distribution AS
SELECT 
    ats_platform,
    COUNT(*) as total_urls,
    COUNT(CASE WHEN crawl_status = 'completed' THEN 1 END) as crawled_urls,
    COUNT(CASE WHEN is_active = TRUE THEN 1 END) as active_urls,
    AVG(confidence) as avg_confidence
FROM discovered_urls
WHERE ats_platform IS NOT NULL
GROUP BY ats_platform
ORDER BY total_urls DESC;
