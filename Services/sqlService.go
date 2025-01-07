package Services

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

type dbData string
type Thumbnail_insrt struct {
	Name      string
	Save_byte []byte
}

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
func InsertDb(db *sql.DB, tI Thumbnail_insrt) bool {
	_, err := db.Exec("insert into thumbnails(name,byt) values($1,$2)", tI.Name, tI.Save_byte)
	if err != nil {
		log.Fatal("Ошибка при отправке запроса в базу данных:", err)
	}
	return true
}
