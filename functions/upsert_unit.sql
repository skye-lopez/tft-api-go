create or replace function upsert_unit (
    g_id text,
    g_name text,
    placement integer
) returns void as $$
declare r record;
begin
    select * from units where id = g_id into r;
    if (r.id is not null) then 
        -- Create new record
--      
    end if;
end
$$ language plpgsql;
