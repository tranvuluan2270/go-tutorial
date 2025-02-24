package config

type Config struct {
	MongoURI string
	Database string
	Port     string
}

func LoadConfig() *Config {
	return &Config{
		MongoURI: "mongodb://localhost:27017/",
		Database: "test-db",
		Port:     ":80",
	}
}
