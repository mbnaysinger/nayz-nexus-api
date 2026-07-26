package config

import (
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Driver do postgres otimizado
	"github.com/jmoiron/sqlx"
)

// ConnectDB inicializa a conexão com o Postgres usando sqlx
func ConnectDB(dsn string) *sqlx.DB {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		log.Fatalf("Erro crítico ao conectar no banco de dados: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Conexão com PostgreSQL estabelecida com sucesso!")
	return db
}
