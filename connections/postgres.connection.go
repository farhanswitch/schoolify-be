package connections

import (
	"database/sql"
	"fmt"
	"log"
	"schoolify_be/configs"

	_ "github.com/lib/pq"
)

var dbInstancePostgres *sql.DB

func DbPostgres() *sql.DB {
	if dbInstancePostgres == nil {
		dbConf := configs.GetConfig().DbPostgres
		psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbConf.Host, dbConf.Port, dbConf.User, dbConf.Pass, dbConf.Database)

		var err error
		db, err := sql.Open("postgres", psqlInfo)
		if err != nil {
			log.Fatalf("[Postgres] Cannot connect to Database.\nError: %s", err.Error())
		}
		dbInstancePostgres = db
		pingErr := db.Ping()
		if pingErr != nil {
			log.Fatalf("[Postgres] Cannot ping the database. Error: %s", pingErr.Error())
		}

		log.Println("[Postgres] Database connected!")
	}
	return dbInstancePostgres
}
