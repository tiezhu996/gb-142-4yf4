package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                   string
	DBHost                 string
	DBPort                 string
	DBName                 string
	DBUser                 string
	DBPassword             string
	APIKey                 string
	ConfirmTimeout         time.Duration
	GreetingCronExpression string
	AlertCronExpression    string
}

func Load() (Config, error) {
	timeoutMinutes, err := strconv.Atoi(getenv("CONFIRM_TIMEOUT_MINUTES", "1440"))
	if err != nil || timeoutMinutes <= 0 {
		return Config{}, fmt.Errorf("parse CONFIRM_TIMEOUT_MINUTES: must be a positive integer")
	}
	return Config{
		Port: getenv("PORT", "3242"), DBHost: getenv("DB_HOST", "127.0.0.1"), DBPort: getenv("DB_PORT", "5742"),
		DBName: getenv("DB_NAME", "care_notify"), DBUser: getenv("DB_USER", "care_user"), DBPassword: getenv("DB_PASSWORD", "care_password"),
		APIKey: getenv("API_KEY", "change-me-for-external-query"), ConfirmTimeout: time.Duration(timeoutMinutes) * time.Minute,
		GreetingCronExpression: getenv("CRON_GREETING", "0 * * * *"), AlertCronExpression: getenv("CRON_ALERT", "*/10 * * * *"),
	}, nil
}
func (c Config) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}
func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
