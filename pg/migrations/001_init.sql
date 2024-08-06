create table teams (
    id text primary key,
    set text not null,
    patch text not null,
    unit_ids text[] not null,
    place_8 integer not null default 0,
    place_7 integer not null default 0,
    place_6 integer not null default 0,
    place_5 integer not null default 0,
    place_4 integer not null default 0,
    place_3 integer not null default 0,
    place_2 integer not null default 0,
    place_1 integer not null default 0
);

-- unit.id = CharacterId + ~ + Set + ~ Patch
create table units (
    id text primary key,
    name text not null,
    avg integer not null default 0,
    place_8 integer not null default 0,
    place_7 integer not null default 0,
    place_6 integer not null default 0,
    place_5 integer not null default 0,
    place_4 integer not null default 0,
    place_3 integer not null default 0,
    place_2 integer not null default 0,
    place_1 integer not null default 0
);

-- units_items.id = units.id + ~ + item1 ~ + item2 + ~ item3
create table unit_item (
    id text primary key,
    units_id text references units(id),
    avg integer not null default 0,
    place_8 integer not null default 0,
    place_7 integer not null default 0,
    place_6 integer not null default 0,
    place_5 integer not null default 0,
    place_4 integer not null default 0,
    place_3 integer not null default 0,
    place_2 integer not null default 0,
    place_1 integer not null default 0
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
    avg integer not null default 0,
    place_8 integer not null default 0,
    place_7 integer not null default 0,
    place_6 integer not null default 0,
    place_5 integer not null default 0,
    place_4 integer not null default 0,
    place_3 integer not null default 0,
    place_2 integer not null default 0,
    place_1 integer not null default 0
);
