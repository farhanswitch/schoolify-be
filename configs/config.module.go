package configs

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type dbConfig struct {
	Host     string
	Port     uint16
	User     string
	Pass     string
	Database string
}

type redisConfig struct {
	Host string
	Port uint16
}
type serviceConfig struct {
	Port       uint16
	Host       string
	EncryptKey string
}

type appConfig struct {
	DbMySql    dbConfig
	DbPostgres dbConfig
	Redis      redisConfig
	Service    serviceConfig
}

func InitModule(path string) {
	err := godotenv.Load(path)
	if err != nil {
		log.Fatalf("Cannot load environment file: %s\n", err.Error())
	}
}

var appConf appConfig

func GetConfig() appConfig {
	if appConf == (appConfig{}) {

		dbPortPostgres, err := strconv.Atoi(os.Getenv("DB_PORT2"))
		if err != nil {
			log.Fatalf("Invalid Database Port 2. Error: %s\n", err.Error())
		}
		var dbConfPostgres dbConfig = dbConfig{
			Host:     os.Getenv("DB_HOST2"),
			Port:     uint16(dbPortPostgres),
			User:     os.Getenv("DB_USER2"),
			Pass:     os.Getenv("DB_PASS2"),
			Database: os.Getenv("DB_NAME2"),
		}
		redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
		if err != nil {
			log.Fatalf("Invalid Redis Port. Error: %s\n", err.Error())
		}
		var redisConf redisConfig = redisConfig{
			Host: os.Getenv("REDIS_HOST"),
			Port: uint16(redisPort),
		}
		appPort, err := strconv.Atoi(os.Getenv("APP_PORT"))
		if err != nil {
			log.Fatalf("Invalid Application Port. Error: %s\n", err.Error())
		}
		var serviceConf serviceConfig = serviceConfig{
			Host:       os.Getenv("APP_HOST"),
			Port:       uint16(appPort),
			EncryptKey: os.Getenv("ENCRYPT_KEY"),
		}

		appConf = appConfig{
			DbPostgres: dbConfPostgres,
			Redis:      redisConf,
			Service:    serviceConf,
		}
		log.Println("Configuration loaded")
	}
	return appConf
}
