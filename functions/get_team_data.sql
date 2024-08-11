create or replace function get_avg_placement(places integer[], sample int) returns numeric as $$
declare
idx integer;
sum_of_places integer;
counter integer;
begin
    idx := 0;
    sum_of_places := 0;
    foreach counter in array places loop
        idx = (idx + 1);
        while counter >= 0 loop
            sum_of_places = (sum_of_places + idx);
            counter = (counter - 1);
        end loop;
    end loop;

    return (sum_of_places::numeric / sample::numeric);
end
$$ language plpgsql;

create or replace function get_team_data(
    fetch_count integer,
    search_patch text
) returns json as $$
declare
t record;
a record;
obj json;
team_data json[];
augment_data json[];
unit_data json[];
unit_item_data json[];
all_data json;
counter integer;
idx integer;
sum_of_places integer;
sample integer;
avg numeric;
begin
    -- GENERATE TEAM DATA 
    team_data := '{}'::json[];
    for t in select id, unit_ids, places, (select sum(s) from unnest(places) s) as sample from teams where patch = search_patch order by sample desc limit fetch_count loop
        avg := get_avg_placement(t.places::integer[], t.sample::int);

        obj := json_build_object(
            'id', t.id,
            'unit_ids', t.unit_ids,
            'places', t.places,
            'sample', t.sample,
            'avg', avg,
            'top4', ((t.places[1] + t.places[2] + t.places[3] + t.places[4])::numeric / t.sample::numeric) * 100,
            'top3', ((t.places[1] + t.places[2] + t.places[3])::numeric / t.sample::numeric) * 100,
            'top2', ((t.places[1] + t.places[2])::numeric / t.sample::numeric) * 100,
            'top1', (t.places[1]::numeric / t.sample::numeric) * 100
        );
        team_data = team_data || obj;
    end loop;

    augment_data := '{}'::json[];
    for t in select id, name, places, (select sum(s) from unnest(places) s) as sample from augments where split_part(id, '~', 3) = search_patch loop
        avg := get_avg_placement(t.places::integer[], t.sample::int);
        
        obj := json_build_object(
            'id', t.id,
            'places', t.places,
            'sample', t.sample,
            'avg', avg,
            'top4', ((t.places[1] + t.places[2] + t.places[3] + t.places[4])::numeric / t.sample::numeric) * 100,
            'top3', ((t.places[1] + t.places[2] + t.places[3])::numeric / t.sample::numeric) * 100,
            'top2', ((t.places[1] + t.places[2])::numeric / t.sample::numeric) * 100,
            'top1', (t.places[1]::numeric / t.sample::numeric) * 100
        );
        augment_data = augment_data || obj;
    end loop;

    unit_data := '{}'::json[];
    for t in select id, name, places, (select sum(s) from unnest(places) s) as sample from units where split_part(id, '~', 3) = search_patch loop
        unit_item_data := '{}'::json[];
        for a in select id, places, (select sum(s) from unnest(places) s) as sample from unit_item where units_id = t.id order by sample desc limit 50 loop
            avg := get_avg_placement(a.places::integer[], a.sample::int);
            obj := json_build_object(
                'id', a.id,
                'places', a.places,
                'sample', a.sample,
                'avg', avg,
                'top4', ((t.places[1] + t.places[2] + t.places[3] + t.places[4])::numeric / t.sample::numeric) * 100,
                'top3', ((t.places[1] + t.places[2] + t.places[3])::numeric / t.sample::numeric) * 100,
                'top2', ((t.places[1] + t.places[2])::numeric / t.sample::numeric) * 100,
                'top1', (t.places[1]::numeric / t.sample::numeric) * 100
            );
            unit_item_data = unit_item_data || obj;
        end loop;

        avg := get_avg_placement(t.places::integer[], t.sample::int);
        
        obj := json_build_object(
            'id', t.id,
            'places', t.places,
            'sample', t.sample,
            'items', unit_item_data,
            'avg', avg,
            'top4', ((t.places[1] + t.places[2] + t.places[3] + t.places[4])::numeric / t.sample::numeric) * 100,
            'top3', ((t.places[1] + t.places[2] + t.places[3])::numeric / t.sample::numeric) * 100,
            'top2', ((t.places[1] + t.places[2])::numeric / t.sample::numeric) * 100,
            'top1', (t.places[1]::numeric / t.sample::numeric) * 100
        );
        unit_data  = unit_data || obj;
    end loop;

    select count(*) as sample from matches into t;

    all_data := json_build_object(
        'teams', team_data,
        'augments', augment_data,
        'units', unit_data,
        'sample', t.sample
    );

    -- create backup
    insert into history (data, patch) values (all_data, search_patch);

    return all_data;
end
$$ language plpgsql;
