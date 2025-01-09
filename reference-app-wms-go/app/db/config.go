package db

import (
	"flag"
	"fmt"
	"os"
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
	flag.StringVar(&config.Host, "db-host", os.Getenv("DB_HOST"), "Database host")
	flag.StringVar(&config.Port, "db-port", os.Getenv("DB_PORT"), "Database port")
	flag.StringVar(&config.User, "db-user", os.Getenv("DB_USER"), "Database user")
	flag.StringVar(&config.Password, "db-password", os.Getenv("DB_PASSWORD"), "Database password")
	flag.StringVar(&config.DBName, "db-name", os.Getenv("DB_NAME"), "Database name")

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
