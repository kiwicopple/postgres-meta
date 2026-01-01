-- List primary keys
-- Parameters: $1 = schema names array (text[])
SELECT
  c.oid::int8 AS table_id,
  n.nspname AS schema,
  c.relname AS table_name,
  a.attname AS name
FROM
  pg_index i
  JOIN pg_class c ON i.indrelid = c.oid
  JOIN pg_namespace n ON c.relnamespace = n.oid
  JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
WHERE
  n.nspname = ANY($1)
  AND i.indisprimary
ORDER BY c.oid, array_position(i.indkey, a.attnum)
