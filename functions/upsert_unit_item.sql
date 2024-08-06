create or replace function upsert_unit_item(
    g_id text,
    g_unit_id text,
    placement integer
) returns void as $$
declare
r record;
placements integer[];
begin
    select * from unit_item where id = g_id into r;
    if (r.id is null) then
        placements = '{0,0,0,0,0,0,0,0}'::integer[];
        placements[placement] = 1;
        insert into unit_item (id, units_id, places) values (g_id, g_unit_id, placements);

        insert into units_items (unit_id, unit_item_id) values (g_unit_id, g_id);
    else
        placements = r.places;
        placements[placement] = (placements[placement] + 1);
        update unit_item set places = placements where id = g_id;
    end if;
end
$$ language plpgsql;
