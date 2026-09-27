package connections

import (
	"database/sql"
	"fmt"
	"log"
	"schoolify_be/configs"

	"github.com/go-sql-driver/mysql"
)

var dbInstance *sql.DB

func DbMySQL() *sql.DB {
	if dbInstance == nil {
		dbConf := configs.GetConfig().DbMySql
		dbPort := dbConf.Port
		config := mysql.Config{
			User:                 dbConf.User,
			Passwd:               dbConf.Pass,
			Net:                  "tcp",
			Addr:                 fmt.Sprintf("%s:%d", dbConf.Host, dbPort),
			DBName:               dbConf.Database,
			AllowNativePasswords: true,
		}

		db, err := sql.Open("mysql", config.FormatDSN())
		if err != nil {
			log.Fatalf("[MySQL] Cannot connect to Database.\nError: %s", err.Error())
		}
		dbInstance = db
		pingErr := db.Ping()
		if pingErr != nil {
			log.Fatalf("[MySQL] Cannot ping the database. Error: %s", pingErr.Error())
		}

		log.Println("[MYSQL] Database connected!")
	}
	return dbInstance
}
