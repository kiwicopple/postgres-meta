-- List foreign tables
-- Parameters: $1 = schema names array (text[])
SELECT
  c.oid::int8 AS id,
  n.nspname AS schema,
  c.relname AS name,
  obj_description(c.oid) AS comment
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = ANY($1)
  AND c.relkind = 'f'
ORDER BY n.nspname, c.relname
