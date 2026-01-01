-- List materialized views
-- Parameters: $1 = schema names array (text[])
SELECT
  c.oid::int8 AS id,
  n.nspname AS schema,
  c.relname AS name,
  pg_get_viewdef(c.oid, true) AS definition,
  obj_description(c.oid) AS comment,
  c.relispopulated AS is_populated
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = ANY($1)
  AND c.relkind = 'm'
ORDER BY n.nspname, c.relname
