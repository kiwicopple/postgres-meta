-- List schemas
-- Parameters: $1 = schema names array (text[]) - if empty, returns all non-system schemas
SELECT
  n.oid::int8 AS id,
  n.nspname AS name,
  u.rolname AS owner
FROM
  pg_namespace n
  JOIN pg_roles u ON n.nspowner = u.oid
WHERE
  (cardinality($1::text[]) = 0 OR n.nspname = ANY($1))
  AND n.nspname !~ '^pg_'
  AND n.nspname <> 'information_schema'
  AND NOT pg_catalog.starts_with(n.nspname, 'pg_temp_')
  AND NOT pg_catalog.starts_with(n.nspname, 'pg_toast_temp_')
  AND (
    pg_has_role(n.nspowner, 'USAGE')
    OR has_schema_privilege(n.oid, 'CREATE, USAGE')
  )
ORDER BY n.nspname
