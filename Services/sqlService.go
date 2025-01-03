package Services

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type dbData string

const DbConnectInfo dbData = "host=213.159.71.120 port=5432 user=postgres password=fabirshe dbname=maindb sslmode=disable"

func (info dbData) CreateConnectDb() *sql.DB {
	db, err := sql.Open("postgres", string(info))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = db.Ping()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	fmt.Println("Successfully connected!")
	return db
}
