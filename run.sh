#!/usr/bin/bash
source ./config.txt

# generate new data
go run main.go

# move any old report and timestamp it
timestamp=$(date +%Y-%m-%d_%H-%M-%S)
mv ./current/data.json ./history/data_$timestamp.json

# generate new report
export PGPASSWORD=$PG_PWD
psql -U $PG_USER -d $PG_DBNAME -c "\copy (select get_team_data(100, '14.15')) to '/home/a/tftgo/current/data.json'"

# edit current s3 obj
aws s3 mv s3://tft-wtf-static/data.json s3://tft-wtf-static/data_$timestamp.json

# upload new data obj
aws s3 cp ./current/data.json s3://tft-wtf-static/data.json
