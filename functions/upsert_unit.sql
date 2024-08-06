create or replace function upsert_unit (
    g_id text,
    g_name text,
    placement integer
) returns void as $$
declare 
r record;
placements integer[];
begin
    select * from units where id = g_id into r;
    if (r.id is null) then 
        placements = '{0,0,0,0,0,0,0,0}'::integer[];
        placements[placement] := 1;
        insert into units (id, name, places) values (g_id, g_name, placements);
    else
        placements := r.places;
        placements[placement] := (placements[placement] + 1);
        update units set places = placements where id = g_id;
    end if;
end
$$ language plpgsql;
