create or replace function get_match(matchid text) returns text as $$
declare r record; result text;
begin
    select * from matches where id = matchid into r;
    if (r.id is null) then
        result = 'none';
    else
        result = r.id;
    end if;
    return result;
end
$$ language plpgsql;
