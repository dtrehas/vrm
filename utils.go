package vrm

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// DataSourceName creates the Database connection string.
// If the password is not provided, is not mentioned in the string.

type DbConfig struct {
	Host, Database, Schema, User, Password string
	Port                                   int
}

func DataSourceName(host string, port int, db, user, pass string) string {

	source := fmt.Sprintf("host=%s port=%d dbname=%s user=%s",
		host,
		port,
		db,
		user)
	if pass != "" {
		source = source + fmt.Sprintf(" password=%s", pass)
	}

	return source
}

func Open(ctx context.Context, conf *DbConfig) *pgx.Conn {

	conn, err := pgx.Connect(ctx,
		DataSourceName(conf.Host,
			conf.Port,
			conf.Database,
			conf.User,
			conf.Password))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	return conn
}

func OpenPool(conf *DbConfig) *pgxpool.Pool {

	conn, err := pgxpool.Connect(context.Background(),
		DataSourceName(conf.Host, conf.Port, conf.Database, conf.User, conf.Password))

	if err != nil {
		//TODO connection error
		log.Fatal(err)
	}
	return conn

}
