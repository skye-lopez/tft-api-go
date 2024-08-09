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
) returns json[] as $$
declare
t record;
obj json;
team_data json[];
augment_data json[];
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
        team_data = team_data || array_append('{}'::json[], obj);
    end loop;

    return team_data;
end
$$ language plpgsql;
