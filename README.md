# tftgo
Simple data analysis of top level play in golang

# need:
- aws cli
- psql 
- go 

# Manual usage:
for testing

`go run main.go`
in pgcli:
`\copy (select get_team_data(100, '14.15')) to /home/a/Desktop/example.json`
