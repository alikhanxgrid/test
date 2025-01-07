package db

import (
	"flag"
	"fmt"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// NewDBConfigFromFlags creates a new DBConfig from command-line flags
func NewDBConfigFromFlags() DBConfig {
	var config DBConfig
	flag.StringVar(&config.Host, "db-host", "localhost", "Database host")
	flag.StringVar(&config.Port, "db-port", "5432", "Database port")
	flag.StringVar(&config.User, "db-user", "abdullahshah", "Database user")
	flag.StringVar(&config.Password, "db-password", "", "Database password")
	flag.StringVar(&config.DBName, "db-name", "wms", "Database name")

	// Only parse flags if they haven't been parsed yet
	if !flag.Parsed() {
		flag.Parse()
	}

	return config
}

// ConnectionString returns the PostgreSQL connection string
func (c DBConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}
