\c ubik;

-- 赛事表，组织结构如下：
-- contest_id：赛事ID，主键，自增
-- contest_name：赛事名称，字符串，不可为空
-- contest_start_date：赛事开始日期，日期类型，不可为空
-- contest_end_date：赛事结束日期，日期类型，不可为空
-- contest_introduction：赛事简介，字符串

CREATE TABLE contests (
    contest_id SERIAL PRIMARY KEY,
    contest_name VARCHAR(255) NOT NULL,
    contest_start_date DATE NOT NULL,
    contest_end_date DATE NOT NULL,
    contest_introduction TEXT
);

-- 赛道表，组织结构如下：
-- track_id：赛道ID，主键，自增
-- track_name：赛道名称，字符串，不可为空
-- track_description：赛道描述，字符串
-- track_settings：赛道设置，JSONB，保留灵活度
-- contest_id：赛事ID，外键，关联contest表的contest_id

CREATE TABLE tracks (
    track_id SERIAL PRIMARY KEY,
    track_name VARCHAR(255) NOT NULL,
    contest_id INT,
    track_description TEXT,
    track_settings JSONB,
    FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE
);

-- 作者表，组织结构如下：
-- author_id：作者ID，主键，自增
-- author_name：作者名称，字符串，不可为空
-- author_email：作者邮箱，字符串，不可为空，唯一
-- author_infos：其他要求的作者信息，JSONB，保留灵活度。需要预先设置。

CREATE TABLE authors (
    author_id SERIAL PRIMARY KEY,
    author_name VARCHAR(255) NOT NULL,
    pen_name VARCHAR(255)  NULL, -- 笔名，可以为空
    password VARCHAR(255) NOT NULL,
    author_email VARCHAR(255) NOT NULL UNIQUE,
    author_infos JSONB
);

-- 作品表，组织结构如下：
-- work_id：作品ID，主键，自增
-- work_title：作品标题，字符串，不可为空
-- work_infos：其他要求的作品信息，JSONB，保留灵活度。通过处理脚本等产生。
-- track_id：赛道ID，外键，关联track表的track_id
-- author_id：作者ID，外键，关联author表的author_id

CREATE TABLE works (
    work_id SERIAL PRIMARY KEY,
    work_title VARCHAR(255) NOT NULL,
    track_id INT,
    author_id INT,
    work_infos JSONB,
    FOREIGN KEY (track_id) REFERENCES tracks(track_id) ON DELETE SET NULL,
    FOREIGN KEY (author_id) REFERENCES authors(author_id) ON DELETE CASCADE
);

-- 评委表，组织结构如下：
-- judge_id：评委ID，主键，自增
-- judge_name：评委名称，字符串，不可为空
-- password：评委密码，字符串，不可为空
-- judge_email：评委邮箱，字符串，不可为空，唯一

CREATE TABLE judges (
    judge_id SERIAL PRIMARY KEY,
    judge_name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
);

-- 评审事件表，用于确定一次评审事件，和赛道绑定，同时规范哪些评委参与评审，评审的开始与截止时间等等信息

CREATE TABLE review_events
(
    event_id   SERIAL PRIMARY KEY,
    track_id   INT,
    event_name VARCHAR(255) NOT NULL,
    start_time TIMESTAMP    NOT NULL,
    end_time   TIMESTAMP    NOT NULL,
    judge_ids  INT[], -- 参与评审的评委ID列表
    FOREIGN KEY (track_id) REFERENCES tracks (track_id) ON DELETE SET NULL
);


-- 评审表，组织结构如下：
-- review_id：评审ID，主键，自增
-- judge_id：评委ID，外键，关联judge表的judge_id
-- work_reviews：评审信息，JSONB，保留灵活度。
-- review_event_id：评审事件ID，外键，关联review_events表的event_id，用于区分不同评审事件的评审信息

CREATE TABLE reviews (
    review_id SERIAL PRIMARY KEY,
    work_id INT ,
    review_event_id INT,
    judge_id INT,
    work_reviews JSONB,
    FOREIGN KEY (review_event_id) REFERENCES review_events(event_id) ON DELETE CASCADE,
    FOREIGN KEY (judge_id) REFERENCES judges(judge_id) ON DELETE SET NULL ,
    FOREIGN KEY (work_id) REFERENCES works(work_id) ON DELETE CASCADE
);

-- 评审结果表，组织结构如下：
-- result_id：评审结果ID，主键，自增
-- work_id：作品ID，外键，关联work表的work_id
-- reviews：评审结果信息，JSONB，保留灵活度。
-- review_event_id：评审事件ID，外键，关联review_events表的event_id，用于区分不同评审事件的结果

CREATE TABLE review_results (
    result_id SERIAL PRIMARY KEY,
    work_id INT,
    review_event_id INT,
    reviews JSONB,
    FOREIGN KEY (work_id) REFERENCES works(work_id) ON DELETE CASCADE ,
    FOREIGN KEY (review_event_id) REFERENCES review_events(event_id) ON DELETE CASCADE
);

CREATE TABLE permissions (
    permission_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE, -- 如: "work.create", "review.submit"
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

-- 全局表
CREATE TABLE global_config (
    id INT PRIMARY KEY CHECK (id = 1), -- 强制只能插入一行
    is_init BOOLEAN DEFAULT FALSE,
    site_name VARCHAR(255) DEFAULT 'Ubik',
    -- 邮箱设置(SMTP协议)
    email_address VARCHAR(255),
    email_app_password VARCHAR(255), --SMTP授权码
    email_smtp_server VARCHAR(255),
    email_smtp_port int DEFAULT 587
);
-- 初始化全局配置表，插入默认值
INSERT INTO global_config (id, is_init) VALUES (1, FALSE);

-- 设置super权限规则，初始化超级管理员角色和用户
INSERT INTO permissions (name, resource, action) VALUES ('super', '*', '*');
INSERT INTO roles (role_name, description, is_super) VALUES ('superadmin', '拥有所有权限的超级管理员角色', TRUE);
INSERT INTO role_permissions (role_id, permission_id) VALUES ((SELECT role_id FROM roles WHERE role_name = 'superadmin'), (SELECT permission_id FROM permissions WHERE name = 'super'));
INSERT INTO admins (admin_name, password) VALUES ('superadmin', 'superpassword');
INSERT INTO admin_roles (admin_id, role_id) VALUES ((SELECT admin_id FROM admins WHERE admin_name = 'superadmin'), (SELECT role_id FROM roles WHERE role_name = 'superadmin'));

-- 动作记录表，记录管理员的操作日志
CREATE TABLE action_logs (
    log_id SERIAL PRIMARY KEY,
    admin_id INT REFERENCES admins(admin_id) ON DELETE SET NULL,
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details JSONB
);
