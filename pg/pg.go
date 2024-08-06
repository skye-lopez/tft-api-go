package pg

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goquery "github.com/skye-lopez/go-query"
)

func Connect() (goquery.GoQuery, error) {
	godotenv.Load("../.env")
	connString := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("PG_USER"),
		os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_DBNAME"),
		os.Getenv("PG_PORT"))

	conn, err := sql.Open("postgres", connString)
	gq := goquery.NewGoQuery(conn)
	return gq, err
}
