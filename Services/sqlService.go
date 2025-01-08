package Services

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"sync"
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
	log.Println("БД успешно подключена!")
	return db
}
func InsertDb_MatchData(db *sql.DB, data map[string][]byte) (matchedData [][]byte) {
	log.Println("Начало просмотра таблицы на наличие совпадений")
	for i, _ := range data {
		rows, err := db.Query("select byt from thumbnails where name ilike $1", i)
		if err != nil {
			log.Fatal("Ошибка при отправке запроса в базу данных:", err)
		}
		for rows.Next() {
			var byt []byte
			// Считываем значения из каждой строки
			err := rows.Scan(&byt)
			if err != nil {
				log.Fatal("ошибка чтения строки: %w", err)
			}
			matchedData = append(matchedData, byt)
		}
	}
	log.Println("Начало инсерта в таблицу")
	var wg sync.WaitGroup
	for i, v := range data {
		wg.Add(1)
		go func(i string, v []byte) {
			defer wg.Done()
			_, err := db.Exec("insert into thumbnails(name,byt) values($1,$2) on conflict (name) do nothing", i, v)
			if err != nil {
				log.Fatal("Ошибка при отправке инсерта в базу данных:", err)
			}
			log.Println("Данные загружены")
		}(i, v)
	}
	wg.Wait()
	return
}
