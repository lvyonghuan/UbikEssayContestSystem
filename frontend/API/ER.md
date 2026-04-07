erDiagram

    contests {
        int contest_id PK
        string contest_name
        date contest_start_date
        date contest_end_date
        text contest_introduction
    }

    tracks {
        int track_id PK
        string track_name
        int contest_id FK
        text track_description
        jsonb track_settings
    }

    authors {
        int author_id PK
        string author_name
        string password
        string author_email
        jsonb author_infos
    }

    works {
        int work_id PK
        string work_title
        int track_id FK
        int author_id FK
        jsonb work_infos
    }

    judges {
        int judge_id PK
        string judge_name
        string password
    }

    review_events {
        int event_id PK
        int track_id FK
        string event_name
        timestamp start_time
        timestamp end_time
        int[] judge_ids "MULTI-VALUED"
    }

    reviews {
        int review_id PK
        int work_id FK
        int review_event_id FK
        int judge_id FK
        jsonb work_reviews
    }

    review_results {
        int result_id PK
        int work_id FK
        int review_event_id FK
        jsonb reviews
    }

    permissions {
        int permission_id PK
        string name
        string resource
        string action
        jsonb meta
    }

    roles {
        int role_id PK
        string role_name
        text description
        boolean is_default
        boolean is_super
        jsonb meta
    }

    role_permissions {
        int role_id FK
        int permission_id FK
    }

    admins {
        int admin_id PK
        string admin_name
        string password
        string admin_email
        boolean is_active
    }

    admin_roles {
        int admin_id FK
        int role_id FK
    }

    global_config {
        int id PK
        boolean is_init
        string site_name
        string email_address
        string email_app_password
        string email_smtp_server
        int email_smtp_port
    }

    action_logs {
        int log_id PK
        int admin_id FK
        string resource
        string action
        timestamp created_at
        jsonb details
    }

    %% 关系

    contests ||--o{ tracks : has
    tracks ||--o{ works : contains
    authors ||--o{ works : creates

    tracks ||--o{ review_events : hosts
    review_events ||--o{ reviews : includes
    judges ||--o{ reviews : writes
    works ||--o{ reviews : receives

    works ||--o{ review_results : summarized_in
    review_events ||--o{ review_results : generates

    roles ||--o{ role_permissions : has
    permissions ||--o{ role_permissions : assigned

    admins ||--o{ admin_roles : has
    roles ||--o{ admin_roles : assigned

    admins ||--o{ action_logs : performs