-- List types (enums and composite types)
-- Parameters: $1 = schema names array (text[])
SELECT
  t.oid::int8 AS id,
  t.typname AS name,
  n.nspname AS schema,
  format_type(t.oid, null) AS format,
  COALESCE(t_enums.enums, '[]'::jsonb) AS enums,
  COALESCE(t_attributes.attributes, '[]'::jsonb) AS attributes,
  obj_description(t.oid, 'pg_type') AS comment
FROM
  pg_type t
  LEFT JOIN pg_namespace n ON n.oid = t.typnamespace
  LEFT JOIN (
    SELECT
      enumtypid,
      jsonb_agg(enumlabel ORDER BY enumsortorder) AS enums
    FROM pg_enum
    GROUP BY enumtypid
  ) AS t_enums ON t_enums.enumtypid = t.oid
  LEFT JOIN (
    SELECT
      c.oid,
      jsonb_agg(
        jsonb_build_object('name', a.attname, 'type_id', a.atttypid::int8)
        ORDER BY a.attnum
      ) AS attributes
    FROM
      pg_class c
      JOIN pg_attribute a ON a.attrelid = c.oid
    WHERE
      c.relkind = 'c' AND NOT a.attisdropped
    GROUP BY c.oid
  ) AS t_attributes ON t_attributes.oid = t.typrelid
WHERE
  n.nspname = ANY($1)
  AND (
    t.typrelid = 0
    OR (SELECT c.relkind = 'c' FROM pg_class c WHERE c.oid = t.typrelid)
  )
  AND NOT EXISTS (
    SELECT 1 FROM pg_type el
    WHERE el.oid = t.typelem AND el.typarray = t.oid
  )
ORDER BY n.nspname, t.typname
