create table teams (
    id text primary key,
    set text not null,
    patch text not null,
    unit_ids text[] not null,
    places integer[] not null default '{0,0,0,0,0,0,0,0}'::integer[]
);

-- unit.id = CharacterId + ~ + Set + ~ Patch
create table units (
    id text primary key,
    name text not null,
    places integer[] not null default '{0,0,0,0,0,0,0,0}'::integer[]
);

-- units_items.id = units.id + ~ + item1 ~ + item2 + ~ item3
create table unit_item (
    id text primary key,
    units_id text references units(id),
    avg integer not null default 0,
    places integer[] not null default '{0,0,0,0,0,0,0,0}'::integer[]
);

create table units_items (
    uuid uuid primary key default gen_random_uuid(),
    unit_id text not null references units(id),
    unit_item_id text not null references unit_item(id)
);

-- augments.id = augmentName + ~ set + ~ + patch 
create table augments (
    id text primary key,
    name text not null,
    places integer[] not null default '{0,0,0,0,0,0,0,0}'::integer[]
);

create table matches (
    id text primary key,
    set text not null,
    patch text not null
);

create table history (
    uuid uuid primary key default gen_random_uuid(),
    created_at timestamp not null default now(),
    data JSONB not null
);
