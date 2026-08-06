package database

import "fmt"

// DBConfig chứa cấu hình kết nối tới PostgreSQL
type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
	MaxConns int32  `json:"max_conns"`
	MinConns int32  `json:"min_conns"`
}

func (c *DBConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%d&pool_min_conns=%d",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode, c.MaxConns, c.MinConns,
	)
}

func DefaultDBConfig() DBConfig {
	return DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "paynexus_db",
		SSLMode:  "disable",
		MaxConns: 20,
		MinConns: 5,
	}
}
