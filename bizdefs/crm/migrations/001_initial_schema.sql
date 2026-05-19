CREATE TABLE IF NOT EXISTS crm_industry (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    industry VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS crm_product (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    product_code VARCHAR(255),
    product_name VARCHAR(255),
    disabled BOOLEAN DEFAULT FALSE,
    image VARCHAR(1024),
    description TEXT,
    standard_rate NUMERIC(15,2)
);

CREATE TABLE IF NOT EXISTS crm_products (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    parent_id VARCHAR(255),
    parent_type VARCHAR(255),
    product_code VARCHAR(255),
    product_name VARCHAR(255),
    qty NUMERIC(10,2),
    rate NUMERIC(15,2),
    amount NUMERIC(15,2),
    discount_percentage NUMERIC(5,2),
    discount_amount NUMERIC(15,2),
    net_amount NUMERIC(15,2)
);

CREATE TABLE IF NOT EXISTS crm_organization (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    organization_name VARCHAR(255),
    website VARCHAR(255),
    organization_logo VARCHAR(1024),
    no_of_employees VARCHAR(100),
    annual_revenue NUMERIC(20,2),
    industry VARCHAR(255),
    currency VARCHAR(10),
    exchange_rate NUMERIC(10,4)
);

CREATE TABLE IF NOT EXISTS crm_deal_status (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    deal_status VARCHAR(255),
    color VARCHAR(100),
    position INT,
    probability NUMERIC(5,2),
    type VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS crm_lead_status (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    lead_status VARCHAR(255),
    color VARCHAR(100),
    position INT,
    type VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS crm_lead (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    salutation VARCHAR(50),
    first_name VARCHAR(100),
    middle_name VARCHAR(100),
    last_name VARCHAR(100),
    lead_name VARCHAR(255),
    gender VARCHAR(50),
    status VARCHAR(255),
    email VARCHAR(255),
    website VARCHAR(255),
    mobile_no VARCHAR(100),
    phone VARCHAR(100),
    no_of_employees VARCHAR(100),
    annual_revenue NUMERIC(20,2),
    industry VARCHAR(255),
    image VARCHAR(1024),
    job_title VARCHAR(255),
    organization VARCHAR(255),
    converted BOOLEAN DEFAULT FALSE,
    total NUMERIC(15,2),
    net_total NUMERIC(15,2)
);

CREATE TABLE IF NOT EXISTS crm_deal (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    organization VARCHAR(255),
    probability NUMERIC(5,2),
    annual_revenue NUMERIC(20,2),
    website VARCHAR(255),
    next_step VARCHAR(255),
    email VARCHAR(255),
    mobile_no VARCHAR(100),
    status VARCHAR(255),
    organization_name VARCHAR(255),
    industry VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    total NUMERIC(15,2),
    net_total NUMERIC(15,2),
    deal_value NUMERIC(15,2),
    expected_deal_value NUMERIC(15,2),
    expected_closure_date TIMESTAMP,
    closed_date TIMESTAMP
);

CREATE TABLE IF NOT EXISTS crm_task (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    title VARCHAR(255),
    priority VARCHAR(50),
    start_date TIMESTAMP,
    status VARCHAR(100),
    due_date TIMESTAMP,
    description TEXT
);
