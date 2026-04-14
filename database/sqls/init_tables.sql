\c ubik;

CREATE TABLE contests (
    contest_id SERIAL PRIMARY KEY,
    contest_name VARCHAR(255) NOT NULL,
    contest_start_date TIMESTAMP NOT NULL,
    contest_end_date TIMESTAMP NOT NULL,
    contest_introduction TEXT
);

CREATE TABLE tracks (
    track_id SERIAL PRIMARY KEY,
    track_name VARCHAR(255) NOT NULL,
    contest_id INT,
    track_description TEXT,
    track_settings JSONB,
    contest_end_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    contest_end_attempt_count INT NOT NULL DEFAULT 0,
    contest_end_last_error TEXT,
    contest_end_last_started_at TIMESTAMP NULL,
    contest_end_last_finished_at TIMESTAMP NULL,
    contest_end_trigger_source VARCHAR(32) NOT NULL DEFAULT 'system',
    contest_end_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE
);

CREATE TABLE authors (
    author_id SERIAL PRIMARY KEY,
    author_name VARCHAR(255) NOT NULL,
    pen_name VARCHAR(255) NULL,
    password VARCHAR(255) NOT NULL,
    author_email VARCHAR(255) NOT NULL UNIQUE,
    author_infos JSONB
);

CREATE TABLE works (
    work_id SERIAL PRIMARY KEY,
    work_title VARCHAR(255) NOT NULL,
    track_id INT,
    author_id INT,
    work_status VARCHAR(128) NOT NULL DEFAULT 'submission_success',
    work_infos JSONB,
    FOREIGN KEY (track_id) REFERENCES tracks(track_id) ON DELETE SET NULL,
    FOREIGN KEY (author_id) REFERENCES authors(author_id) ON DELETE CASCADE
);

CREATE TABLE judges (
    judge_id SERIAL PRIMARY KEY,
    judge_name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE review_events (
    event_id SERIAL PRIMARY KEY,
    track_id INT,
    event_name VARCHAR(255) NOT NULL,
    work_status VARCHAR(128) NOT NULL DEFAULT 'submission_success',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    FOREIGN KEY (track_id) REFERENCES tracks(track_id) ON DELETE SET NULL
);

CREATE TABLE review_event_judges (
    event_id INT NOT NULL,
    judge_id INT NOT NULL,
    deadline_at TIMESTAMP NULL,
    PRIMARY KEY (event_id, judge_id),
    FOREIGN KEY (event_id) REFERENCES review_events(event_id) ON DELETE CASCADE,
    FOREIGN KEY (judge_id) REFERENCES judges(judge_id) ON DELETE CASCADE
);

CREATE TABLE reviews (
    review_id SERIAL PRIMARY KEY,
    work_id INT,
    review_event_id INT,
    judge_id INT,
    work_reviews JSONB,
    FOREIGN KEY (review_event_id) REFERENCES review_events(event_id) ON DELETE CASCADE,
    FOREIGN KEY (judge_id) REFERENCES judges(judge_id) ON DELETE SET NULL,
    FOREIGN KEY (work_id) REFERENCES works(work_id) ON DELETE CASCADE
);

CREATE TABLE review_results (
    result_id SERIAL PRIMARY KEY,
    work_id INT,
    review_event_id INT,
    reviews JSONB,
    FOREIGN KEY (work_id) REFERENCES works(work_id) ON DELETE CASCADE,
    FOREIGN KEY (review_event_id) REFERENCES review_events(event_id) ON DELETE CASCADE
);

CREATE TABLE permissions (
    permission_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    resource VARCHAR(255),
    action VARCHAR(255),
    meta JSONB
);

CREATE TABLE roles (
    role_id SERIAL PRIMARY KEY,
    role_name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    is_super BOOLEAN DEFAULT FALSE,
    meta JSONB
);

CREATE TABLE role_permissions (
    role_id INT NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
    permission_id INT NOT NULL REFERENCES permissions(permission_id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE admins (
    admin_id SERIAL PRIMARY KEY,
    admin_name VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    admin_email VARCHAR(255) UNIQUE,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE admin_roles (
    admin_id INT NOT NULL REFERENCES admins(admin_id) ON DELETE CASCADE,
    role_id INT NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
    PRIMARY KEY (admin_id, role_id)
);

CREATE TABLE global_config (
    id INT PRIMARY KEY CHECK (id = 1),
    is_init BOOLEAN DEFAULT FALSE,
    site_name VARCHAR(255) DEFAULT 'Ubik',
    email_address VARCHAR(255),
    email_app_password VARCHAR(255),
    email_smtp_server VARCHAR(255),
    email_smtp_port INT DEFAULT 587
);

INSERT INTO global_config (id, is_init) VALUES (1, FALSE);

INSERT INTO permissions (name, resource, action) VALUES
    ('contest.create', 'contest', 'create'),
    ('contest.update', 'contest', 'update'),
    ('contest.delete', 'contest', 'delete'),
    ('author.read', 'author', 'read'),
    ('author.update', 'author', 'update'),
    ('author.delete', 'author', 'delete'),
    ('track.create', 'track', 'create'),
    ('track.update', 'track', 'update'),
    ('track.delete', 'track', 'delete'),
    ('works.read', 'works', 'read'),
    ('works.delete', 'works', 'delete'),
    ('super', '*', '*')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (role_name, description, is_super)
VALUES ('superadmin', 'super admin role', TRUE)
ON CONFLICT (role_name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
VALUES (
    (SELECT role_id FROM roles WHERE role_name = 'superadmin'),
    (SELECT permission_id FROM permissions WHERE name = 'super')
)
ON CONFLICT DO NOTHING;

INSERT INTO admins (admin_name, password)
VALUES ('superadmin', 'superpassword')
ON CONFLICT (admin_name) DO NOTHING;

INSERT INTO admin_roles (admin_id, role_id)
VALUES (
    (SELECT admin_id FROM admins WHERE admin_name = 'superadmin'),
    (SELECT role_id FROM roles WHERE role_name = 'superadmin')
)
ON CONFLICT DO NOTHING;

CREATE TABLE script_definitions (
    script_id SERIAL PRIMARY KEY,
    script_key VARCHAR(255) NOT NULL UNIQUE,
    script_name VARCHAR(255) NOT NULL,
    interpreter VARCHAR(64) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT TRUE,
    meta JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE script_versions (
    version_id SERIAL PRIMARY KEY,
    script_id INT NOT NULL REFERENCES script_definitions(script_id) ON DELETE CASCADE,
    version_num INT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    relative_path TEXT NOT NULL,
    checksum VARCHAR(128),
    is_active BOOLEAN DEFAULT FALSE,
    created_by INT REFERENCES admins(admin_id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(script_id, version_num)
);

CREATE TABLE script_flows (
    flow_id SERIAL PRIMARY KEY,
    flow_key VARCHAR(255) NOT NULL UNIQUE,
    flow_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT TRUE,
    meta JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE script_flow_steps (
    step_id SERIAL PRIMARY KEY,
    flow_id INT NOT NULL REFERENCES script_flows(flow_id) ON DELETE CASCADE,
    step_order INT NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    script_id INT NOT NULL REFERENCES script_definitions(script_id) ON DELETE CASCADE,
    script_version_id INT REFERENCES script_versions(version_id) ON DELETE SET NULL,
    timeout_ms INT DEFAULT 5000,
    failure_strategy VARCHAR(32) DEFAULT 'fail_close',
    input_template JSONB,
    is_enabled BOOLEAN DEFAULT TRUE,
    UNIQUE(flow_id, step_order)
);

CREATE TABLE script_flow_mounts (
    mount_id SERIAL PRIMARY KEY,
    flow_id INT NOT NULL REFERENCES script_flows(flow_id) ON DELETE CASCADE,
    scope VARCHAR(64) NOT NULL,
    event_key VARCHAR(128) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id INT NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scope, event_key, target_type, target_id)
);

CREATE TABLE action_logs (
    log_id SERIAL PRIMARY KEY,
    admin_id INT REFERENCES admins(admin_id) ON DELETE SET NULL,
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details JSONB
);
