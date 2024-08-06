create or replace function upsert_team(
    g_id text,
    g_set text,
    g_patch text,
    g_unit_ids text[],
    placement integer 
) returns void as $$
declare
r record;
placements integer[];
begin
    select * from teams where id = g_id into r;
    if (r.id is null) then
        placements = '{0,0,0,0,0,0,0,0}'::integer[];
        placements[placement] = 1;
        insert into teams (id, set, patch, unit_ids, places) values (g_id, g_set, g_patch, g_unit_ids, placements);
    else
        placements = r.places;
        placements[placement] = (placements[placement] + 1);
        update teams set places = placements where id = g_id;
    end if;
end
$$ language plpgsql;
