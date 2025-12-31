-- Pokemon Database Schema
-- An advanced schema demonstrating various PostgreSQL features

-- =============================================================================
-- ENUMS
-- =============================================================================

-- Pokemon element types
CREATE TYPE pokemon_type AS ENUM (
    'normal', 'fire', 'water', 'electric', 'grass', 'ice',
    'fighting', 'poison', 'ground', 'flying', 'psychic', 'bug',
    'rock', 'ghost', 'dragon', 'dark', 'steel', 'fairy'
);

-- Pokemon status conditions
CREATE TYPE status_condition AS ENUM (
    'none', 'burned', 'frozen', 'paralyzed', 'poisoned', 'badly_poisoned', 'asleep'
);

-- Move categories
CREATE TYPE move_category AS ENUM (
    'physical', 'special', 'status'
);

-- Evolution methods
CREATE TYPE evolution_method AS ENUM (
    'level_up', 'trade', 'item', 'friendship', 'time_of_day', 'location', 'special'
);

-- Pokemon nature
CREATE TYPE pokemon_nature AS ENUM (
    'hardy', 'lonely', 'brave', 'adamant', 'naughty',
    'bold', 'docile', 'relaxed', 'impish', 'lax',
    'timid', 'hasty', 'serious', 'jolly', 'naive',
    'modest', 'mild', 'quiet', 'bashful', 'rash',
    'calm', 'gentle', 'sassy', 'careful', 'quirky'
);

-- Battle terrain
CREATE TYPE battle_terrain AS ENUM (
    'normal', 'electric', 'grassy', 'misty', 'psychic'
);

-- Weather conditions
CREATE TYPE weather_condition AS ENUM (
    'clear', 'sunny', 'rain', 'sandstorm', 'hail', 'snow', 'fog'
);

-- =============================================================================
-- COMPOSITE TYPES
-- =============================================================================

-- Stats structure used for base stats, IVs, EVs
CREATE TYPE pokemon_stats AS (
    hp integer,
    attack integer,
    defense integer,
    special_attack integer,
    special_defense integer,
    speed integer
);

-- Location coordinates
CREATE TYPE geo_location AS (
    latitude numeric(10, 7),
    longitude numeric(10, 7),
    altitude integer
);

-- =============================================================================
-- TABLES
-- =============================================================================

-- Regions in the Pokemon world
CREATE TABLE regions (
    id serial PRIMARY KEY,
    name varchar(100) NOT NULL UNIQUE,
    generation integer NOT NULL,
    description text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Locations within regions
CREATE TABLE locations (
    id serial PRIMARY KEY,
    region_id integer NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    location_type varchar(50) NOT NULL, -- city, route, cave, forest, etc.
    coordinates geo_location,
    has_pokemon_center boolean DEFAULT false,
    has_pokemart boolean DEFAULT false,
    description text,
    created_at timestamptz DEFAULT now()
);

-- Abilities that Pokemon can have
CREATE TABLE abilities (
    id serial PRIMARY KEY,
    name varchar(50) NOT NULL UNIQUE,
    description text NOT NULL,
    is_hidden boolean DEFAULT false,
    generation_introduced integer NOT NULL DEFAULT 1,
    created_at timestamptz DEFAULT now()
);

-- Pokemon species (the "Pokedex" entries)
CREATE TABLE pokemon_species (
    id serial PRIMARY KEY,
    national_dex_number integer NOT NULL UNIQUE,
    name varchar(50) NOT NULL UNIQUE,
    primary_type pokemon_type NOT NULL,
    secondary_type pokemon_type,
    base_stats pokemon_stats NOT NULL,
    height_cm integer NOT NULL,
    weight_grams integer NOT NULL,
    catch_rate integer NOT NULL CHECK (catch_rate >= 0 AND catch_rate <= 255),
    base_experience integer NOT NULL,
    base_happiness integer NOT NULL DEFAULT 70 CHECK (base_happiness >= 0 AND base_happiness <= 255),
    growth_rate varchar(20) NOT NULL,
    egg_groups text[] NOT NULL DEFAULT '{}',
    gender_ratio numeric(3,2) CHECK (gender_ratio >= 0 AND gender_ratio <= 1), -- NULL for genderless
    egg_cycles integer NOT NULL DEFAULT 20,
    is_legendary boolean DEFAULT false,
    is_mythical boolean DEFAULT false,
    is_baby boolean DEFAULT false,
    generation integer NOT NULL,
    description text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Pokemon species abilities (many-to-many)
CREATE TABLE pokemon_species_abilities (
    id serial PRIMARY KEY,
    pokemon_species_id integer NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    ability_id integer NOT NULL REFERENCES abilities(id) ON DELETE CASCADE,
    is_hidden boolean DEFAULT false,
    slot integer NOT NULL CHECK (slot >= 1 AND slot <= 3),
    UNIQUE (pokemon_species_id, ability_id),
    UNIQUE (pokemon_species_id, slot)
);

-- Evolution chains
CREATE TABLE evolution_chains (
    id serial PRIMARY KEY,
    created_at timestamptz DEFAULT now()
);

-- Evolutions
CREATE TABLE evolutions (
    id serial PRIMARY KEY,
    chain_id integer NOT NULL REFERENCES evolution_chains(id) ON DELETE CASCADE,
    from_species_id integer NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    to_species_id integer NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    method evolution_method NOT NULL,
    level_required integer,
    item_required varchar(50),
    trade_species_id integer REFERENCES pokemon_species(id),
    time_of_day varchar(10) CHECK (time_of_day IN ('day', 'night')),
    location_id integer REFERENCES locations(id),
    min_happiness integer,
    min_affection integer,
    known_move varchar(50),
    gender varchar(10) CHECK (gender IN ('male', 'female')),
    held_item varchar(50),
    weather weather_condition,
    other_conditions jsonb,
    created_at timestamptz DEFAULT now()
);

-- Moves
CREATE TABLE moves (
    id serial PRIMARY KEY,
    name varchar(50) NOT NULL UNIQUE,
    move_type pokemon_type NOT NULL,
    category move_category NOT NULL,
    power integer CHECK (power >= 0 AND power <= 250),
    accuracy integer CHECK (accuracy >= 0 AND accuracy <= 100),
    pp integer NOT NULL CHECK (pp >= 1 AND pp <= 40),
    priority integer NOT NULL DEFAULT 0 CHECK (priority >= -7 AND priority <= 5),
    target varchar(30) NOT NULL, -- single, all_opponents, self, ally, etc.
    makes_contact boolean DEFAULT false,
    effect_description text,
    effect_chance integer CHECK (effect_chance >= 0 AND effect_chance <= 100),
    status_inflicted status_condition,
    stat_changes jsonb, -- e.g., {"attack": -1, "speed": 2}
    generation_introduced integer NOT NULL DEFAULT 1,
    is_z_move boolean DEFAULT false,
    is_max_move boolean DEFAULT false,
    created_at timestamptz DEFAULT now()
);

-- Pokemon species learnable moves
CREATE TABLE pokemon_moves (
    id serial PRIMARY KEY,
    pokemon_species_id integer NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    move_id integer NOT NULL REFERENCES moves(id) ON DELETE CASCADE,
    learn_method varchar(20) NOT NULL, -- level_up, tm, hm, egg, tutor
    level_learned integer,
    tm_number varchar(10),
    created_at timestamptz DEFAULT now(),
    UNIQUE (pokemon_species_id, move_id, learn_method)
);

-- Trainers
CREATE TABLE trainers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username varchar(50) NOT NULL UNIQUE,
    email varchar(255) NOT NULL UNIQUE,
    display_name varchar(100),
    avatar_url text,
    home_region_id integer REFERENCES regions(id),
    trainer_class varchar(50) DEFAULT 'Pokemon Trainer',
    badges_earned integer[] DEFAULT '{}',
    pokedex_seen integer DEFAULT 0,
    pokedex_caught integer DEFAULT 0,
    total_battles integer DEFAULT 0,
    battles_won integer DEFAULT 0,
    bio text,
    is_verified boolean DEFAULT false,
    is_champion boolean DEFAULT false,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    last_active_at timestamptz DEFAULT now()
);

-- Individual Pokemon (owned by trainers)
CREATE TABLE pokemon (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    species_id integer NOT NULL REFERENCES pokemon_species(id),
    trainer_id uuid REFERENCES trainers(id) ON DELETE SET NULL,
    nickname varchar(50),
    level integer NOT NULL DEFAULT 1 CHECK (level >= 1 AND level <= 100),
    experience integer NOT NULL DEFAULT 0,
    nature pokemon_nature NOT NULL,
    ability_id integer NOT NULL REFERENCES abilities(id),
    gender varchar(10) CHECK (gender IN ('male', 'female', 'genderless')),
    is_shiny boolean DEFAULT false,
    individual_values pokemon_stats NOT NULL,
    effort_values pokemon_stats NOT NULL DEFAULT ROW(0, 0, 0, 0, 0, 0)::pokemon_stats,
    current_hp integer NOT NULL,
    status status_condition DEFAULT 'none',
    happiness integer NOT NULL DEFAULT 70 CHECK (happiness >= 0 AND happiness <= 255),
    held_item varchar(50),
    pokeball_type varchar(30) DEFAULT 'poke_ball',
    original_trainer_id uuid,
    caught_at timestamptz DEFAULT now(),
    caught_location_id integer REFERENCES locations(id),
    caught_level integer,
    is_egg boolean DEFAULT false,
    egg_cycles_remaining integer,
    form_id integer,
    dynamax_level integer DEFAULT 0 CHECK (dynamax_level >= 0 AND dynamax_level <= 10),
    can_gigantamax boolean DEFAULT false,
    tera_type pokemon_type,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Pokemon known moves (current moveset, max 4)
CREATE TABLE pokemon_known_moves (
    id serial PRIMARY KEY,
    pokemon_id uuid NOT NULL REFERENCES pokemon(id) ON DELETE CASCADE,
    move_id integer NOT NULL REFERENCES moves(id),
    slot integer NOT NULL CHECK (slot >= 1 AND slot <= 4),
    pp_remaining integer NOT NULL,
    pp_ups integer DEFAULT 0 CHECK (pp_ups >= 0 AND pp_ups <= 3),
    UNIQUE (pokemon_id, slot),
    UNIQUE (pokemon_id, move_id)
);

-- Teams
CREATE TABLE teams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trainer_id uuid NOT NULL REFERENCES trainers(id) ON DELETE CASCADE,
    name varchar(50) NOT NULL,
    description text,
    is_active boolean DEFAULT false,
    format varchar(30), -- singles, doubles, vgc, etc.
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Team members (max 6 Pokemon per team)
CREATE TABLE team_members (
    id serial PRIMARY KEY,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    pokemon_id uuid NOT NULL REFERENCES pokemon(id) ON DELETE CASCADE,
    slot integer NOT NULL CHECK (slot >= 1 AND slot <= 6),
    UNIQUE (team_id, slot),
    UNIQUE (team_id, pokemon_id)
);

-- Battles
CREATE TABLE battles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    format varchar(30) NOT NULL,
    terrain battle_terrain DEFAULT 'normal',
    weather weather_condition DEFAULT 'clear',
    trainer1_id uuid NOT NULL REFERENCES trainers(id),
    trainer2_id uuid NOT NULL REFERENCES trainers(id),
    winner_id uuid REFERENCES trainers(id),
    started_at timestamptz DEFAULT now(),
    ended_at timestamptz,
    turns integer,
    is_ranked boolean DEFAULT false,
    replay_data jsonb,
    CHECK (trainer1_id != trainer2_id)
);

-- Battle log entries
CREATE TABLE battle_logs (
    id serial PRIMARY KEY,
    battle_id uuid NOT NULL REFERENCES battles(id) ON DELETE CASCADE,
    turn integer NOT NULL,
    action_order integer NOT NULL,
    actor_trainer_id uuid NOT NULL REFERENCES trainers(id),
    action_type varchar(20) NOT NULL, -- move, switch, item, flee
    action_details jsonb NOT NULL,
    result jsonb,
    created_at timestamptz DEFAULT now()
);

-- Items
CREATE TABLE items (
    id serial PRIMARY KEY,
    name varchar(50) NOT NULL UNIQUE,
    category varchar(30) NOT NULL,
    description text NOT NULL,
    buy_price integer,
    sell_price integer,
    effect jsonb,
    is_holdable boolean DEFAULT false,
    is_consumable boolean DEFAULT false,
    fling_power integer,
    fling_effect text,
    sprite_url text,
    created_at timestamptz DEFAULT now()
);

-- Trainer inventory
CREATE TABLE trainer_inventory (
    id serial PRIMARY KEY,
    trainer_id uuid NOT NULL REFERENCES trainers(id) ON DELETE CASCADE,
    item_id integer NOT NULL REFERENCES items(id),
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity >= 0),
    UNIQUE (trainer_id, item_id)
);

-- Pokedex entries (tracks which Pokemon a trainer has seen/caught)
CREATE TABLE pokedex_entries (
    id serial PRIMARY KEY,
    trainer_id uuid NOT NULL REFERENCES trainers(id) ON DELETE CASCADE,
    pokemon_species_id integer NOT NULL REFERENCES pokemon_species(id),
    seen boolean DEFAULT true,
    caught boolean DEFAULT false,
    first_seen_at timestamptz DEFAULT now(),
    first_caught_at timestamptz,
    times_caught integer DEFAULT 0,
    UNIQUE (trainer_id, pokemon_species_id)
);

-- =============================================================================
-- VIEWS
-- =============================================================================

-- View for trainer stats with computed fields
CREATE VIEW trainer_stats AS
SELECT
    t.id,
    t.username,
    t.display_name,
    t.trainer_class,
    t.total_battles,
    t.battles_won,
    CASE
        WHEN t.total_battles > 0
        THEN ROUND((t.battles_won::numeric / t.total_battles) * 100, 2)
        ELSE 0
    END as win_rate,
    t.pokedex_caught,
    (SELECT COUNT(*) FROM pokemon WHERE trainer_id = t.id) as total_pokemon,
    (SELECT COUNT(*) FROM pokemon WHERE trainer_id = t.id AND is_shiny = true) as shiny_count,
    array_length(t.badges_earned, 1) as badge_count,
    t.is_champion,
    t.created_at as member_since
FROM trainers t;

-- View for Pokemon with full species info
CREATE VIEW pokemon_full AS
SELECT
    p.id,
    p.nickname,
    ps.name as species_name,
    ps.national_dex_number,
    ps.primary_type,
    ps.secondary_type,
    p.level,
    p.nature,
    a.name as ability_name,
    p.gender,
    p.is_shiny,
    p.individual_values,
    p.effort_values,
    p.current_hp,
    p.status,
    p.happiness,
    p.held_item,
    t.username as trainer_username,
    l.name as caught_location,
    r.name as caught_region,
    p.caught_at
FROM pokemon p
JOIN pokemon_species ps ON p.species_id = ps.id
JOIN abilities a ON p.ability_id = a.id
LEFT JOIN trainers t ON p.trainer_id = t.id
LEFT JOIN locations l ON p.caught_location_id = l.id
LEFT JOIN regions r ON l.region_id = r.id;

-- View for type effectiveness analysis
CREATE VIEW type_matchups AS
SELECT
    ps.id,
    ps.name,
    ps.primary_type,
    ps.secondary_type,
    ps.base_stats,
    (ps.base_stats).hp + (ps.base_stats).attack + (ps.base_stats).defense +
    (ps.base_stats).special_attack + (ps.base_stats).special_defense + (ps.base_stats).speed as base_stat_total,
    ps.is_legendary,
    ps.is_mythical
FROM pokemon_species ps;

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Function to calculate effective stats for a Pokemon at a given level
CREATE OR REPLACE FUNCTION calculate_stat(
    base_stat integer,
    iv integer,
    ev integer,
    level integer,
    nature_modifier numeric DEFAULT 1.0,
    is_hp boolean DEFAULT false
) RETURNS integer AS $$
BEGIN
    IF is_hp THEN
        RETURN FLOOR(((2 * base_stat + iv + FLOOR(ev / 4.0)) * level / 100.0) + level + 10);
    ELSE
        RETURN FLOOR((FLOOR(((2 * base_stat + iv + FLOOR(ev / 4.0)) * level / 100.0) + 5)) * nature_modifier);
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to get a Pokemon's full calculated stats
CREATE OR REPLACE FUNCTION get_pokemon_calculated_stats(pokemon_uuid uuid)
RETURNS pokemon_stats AS $$
DECLARE
    poke RECORD;
    species RECORD;
    nature_mods RECORD;
    result pokemon_stats;
BEGIN
    SELECT * INTO poke FROM pokemon WHERE id = pokemon_uuid;
    SELECT * INTO species FROM pokemon_species WHERE id = poke.species_id;

    -- Simplified nature modifiers (in reality, each nature affects different stats)
    SELECT 1.0 AS atk, 1.0 AS def, 1.0 AS spa, 1.0 AS spd, 1.0 AS spe INTO nature_mods;

    result.hp := calculate_stat(
        (species.base_stats).hp,
        (poke.individual_values).hp,
        (poke.effort_values).hp,
        poke.level,
        1.0,
        true
    );
    result.attack := calculate_stat(
        (species.base_stats).attack,
        (poke.individual_values).attack,
        (poke.effort_values).attack,
        poke.level,
        nature_mods.atk
    );
    result.defense := calculate_stat(
        (species.base_stats).defense,
        (poke.individual_values).defense,
        (poke.effort_values).defense,
        poke.level,
        nature_mods.def
    );
    result.special_attack := calculate_stat(
        (species.base_stats).special_attack,
        (poke.individual_values).special_attack,
        (poke.effort_values).special_attack,
        poke.level,
        nature_mods.spa
    );
    result.special_defense := calculate_stat(
        (species.base_stats).special_defense,
        (poke.individual_values).special_defense,
        (poke.effort_values).special_defense,
        poke.level,
        nature_mods.spd
    );
    result.speed := calculate_stat(
        (species.base_stats).speed,
        (poke.individual_values).speed,
        (poke.effort_values).speed,
        poke.level,
        nature_mods.spe
    );

    RETURN result;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to search Pokemon by type
CREATE OR REPLACE FUNCTION search_pokemon_by_type(
    type1 pokemon_type,
    type2 pokemon_type DEFAULT NULL
) RETURNS SETOF pokemon_species AS $$
BEGIN
    IF type2 IS NULL THEN
        RETURN QUERY
        SELECT * FROM pokemon_species
        WHERE primary_type = type1 OR secondary_type = type1
        ORDER BY national_dex_number;
    ELSE
        RETURN QUERY
        SELECT * FROM pokemon_species
        WHERE (primary_type = type1 AND secondary_type = type2)
           OR (primary_type = type2 AND secondary_type = type1)
        ORDER BY national_dex_number;
    END IF;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to get evolution chain for a species
CREATE OR REPLACE FUNCTION get_evolution_chain(species_id integer)
RETURNS TABLE (
    stage integer,
    pokemon_name varchar,
    evolves_from varchar,
    method evolution_method,
    requirements jsonb
) AS $$
WITH RECURSIVE chain AS (
    -- Base case: find the chain this species belongs to
    SELECT
        e.chain_id,
        e.from_species_id,
        e.to_species_id,
        e.method,
        jsonb_build_object(
            'level', e.level_required,
            'item', e.item_required,
            'happiness', e.min_happiness,
            'time', e.time_of_day
        ) as requirements,
        1 as stage
    FROM evolutions e
    WHERE e.from_species_id = species_id OR e.to_species_id = species_id

    UNION ALL

    -- Recursive case: find connected evolutions
    SELECT
        e.chain_id,
        e.from_species_id,
        e.to_species_id,
        e.method,
        jsonb_build_object(
            'level', e.level_required,
            'item', e.item_required,
            'happiness', e.min_happiness,
            'time', e.time_of_day
        ),
        c.stage + 1
    FROM evolutions e
    JOIN chain c ON e.from_species_id = c.to_species_id
    WHERE e.chain_id = c.chain_id
)
SELECT DISTINCT
    c.stage,
    ps.name as pokemon_name,
    ps_from.name as evolves_from,
    c.method,
    c.requirements
FROM chain c
JOIN pokemon_species ps ON ps.id = c.to_species_id
LEFT JOIN pokemon_species ps_from ON ps_from.id = c.from_species_id
ORDER BY c.stage;
$$ LANGUAGE SQL STABLE;

-- =============================================================================
-- INDEXES
-- =============================================================================

CREATE INDEX idx_pokemon_trainer ON pokemon(trainer_id);
CREATE INDEX idx_pokemon_species ON pokemon(species_id);
CREATE INDEX idx_pokemon_shiny ON pokemon(is_shiny) WHERE is_shiny = true;
CREATE INDEX idx_pokemon_species_type ON pokemon_species(primary_type, secondary_type);
CREATE INDEX idx_moves_type ON moves(move_type);
CREATE INDEX idx_locations_region ON locations(region_id);
CREATE INDEX idx_trainers_username ON trainers(username);
CREATE INDEX idx_battles_trainers ON battles(trainer1_id, trainer2_id);
CREATE INDEX idx_team_members_team ON team_members(team_id);

-- =============================================================================
-- SAMPLE DATA COMMENTS
-- =============================================================================

COMMENT ON TABLE pokemon_species IS 'The Pokedex entries - master list of all Pokemon species';
COMMENT ON TABLE pokemon IS 'Individual Pokemon instances owned by trainers';
COMMENT ON TABLE trainers IS 'Pokemon trainers (users of the system)';
COMMENT ON TABLE battles IS 'Battle records between trainers';
COMMENT ON TYPE pokemon_stats IS 'A composite type representing the six core Pokemon stats';
COMMENT ON TYPE pokemon_type IS 'The 18 Pokemon elemental types';
