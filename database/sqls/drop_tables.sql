\c ubik;

BEGIN;
DROP TABLE IF EXISTS
    action_logs,
    admin_roles,
    admins,
    role_permissions,
    roles,
    permissions,
    review_results,
    reviews,
    review_events,
    judges,
    works,
    authors,
    tracks,
    contests,
    global_config
    CASCADE;
COMMIT;