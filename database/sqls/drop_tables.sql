\c ubik;

BEGIN;
DROP TABLE IF EXISTS
    action_logs,
    script_flow_mounts,
    script_flow_steps,
    script_flows,
    script_versions,
    script_definitions,
    admin_roles,
    admins,
    role_permissions,
    roles,
    permissions,
    review_results,
    reviews,
    review_event_judges,
    review_events,
    judges,
    works,
    authors,
    tracks,
    contests,
    global_config
    CASCADE;
COMMIT;