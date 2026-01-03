-- List functions (excluding triggers)
-- Parameters: $1 = schema names array (text[])
WITH functions AS (
  SELECT
    p.*,
    -- proargmodes is null when all arg modes are IN
    COALESCE(
      p.proargmodes,
      array_fill('i'::"char", ARRAY[COALESCE(cardinality(p.proallargtypes), cardinality(p.proargtypes))])
    ) AS arg_modes,
    -- proargnames is null when all args are unnamed
    COALESCE(
      p.proargnames,
      array_fill(''::text, ARRAY[COALESCE(cardinality(p.proallargtypes), cardinality(p.proargtypes))])
    ) AS arg_names,
    -- proallargtypes is null when all arg modes are IN
    COALESCE(p.proallargtypes, p.proargtypes) AS arg_types
  FROM pg_proc p
  JOIN pg_namespace n ON p.pronamespace = n.oid
  WHERE n.nspname = ANY($1)
    AND p.prokind = 'f'
    AND NOT EXISTS (
      SELECT 1 FROM pg_trigger t WHERE t.tgfoid = p.oid
    )
)
SELECT
  f.oid::int8 AS id,
  n.nspname AS schema,
  f.proname AS name,
  l.lanname AS language,
  CASE
    WHEN l.lanname = 'internal' THEN f.prosrc
    ELSE pg_get_functiondef(f.oid)
  END AS definition,
  pg_get_function_arguments(f.oid) AS argument_types,
  pg_get_function_identity_arguments(f.oid) AS identity_argument_types,
  f.prorettype::int8 AS return_type_id,
  pg_catalog.format_type(f.prorettype, NULL) AS return_type,
  NULLIF(rt.typrelid, 0)::int8 AS return_type_relation_id,
  f.proretset AS is_set_returning_function,
  CASE f.provolatile
    WHEN 'i' THEN 'IMMUTABLE'
    WHEN 's' THEN 'STABLE'
    WHEN 'v' THEN 'VOLATILE'
  END AS behavior,
  f.prosecdef AS security_definer,
  COALESCE(f_args.args, '[]'::jsonb) AS args,
  pg_catalog.obj_description(f.oid, 'pg_proc') AS comment,
  f.proconfig AS config_params
FROM functions f
JOIN pg_namespace n ON f.pronamespace = n.oid
JOIN pg_language l ON f.prolang = l.oid
LEFT JOIN pg_type rt ON rt.oid = f.prorettype
LEFT JOIN (
  SELECT
    oid,
    jsonb_agg(
      jsonb_build_object(
        'mode', CASE mode
          WHEN 'i' THEN 'in'
          WHEN 'o' THEN 'out'
          WHEN 'b' THEN 'inout'
          WHEN 'v' THEN 'variadic'
          WHEN 't' THEN 'table'
        END,
        'name', COALESCE(NULLIF(name, ''), ''),
        'type_id', type_id::int8,
        'has_default', FALSE
      )
      ORDER BY ord
    ) AS args
  FROM (
    SELECT
      oid,
      unnest(arg_modes) AS mode,
      unnest(arg_names) AS name,
      unnest(arg_types) AS type_id,
      generate_series(1, COALESCE(cardinality(arg_types), 0)) AS ord
    FROM functions
  ) t
  GROUP BY oid
) f_args ON f_args.oid = f.oid
ORDER BY n.nspname, f.proname
