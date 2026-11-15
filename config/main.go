package config

import (
	"errors"
	"io/fs"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("app_rate_limit")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	// BindEnv only fails when called without key names, so its error is not reachable here.
	_ = viper.BindEnv("SERVER_PORT")
	if err := viper.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// Read each key directly: Unmarshal only sees keys present in the file,
	// which drops env-only values when the file is absent.
	return &Config{
		ServerPort: viper.GetString("SERVER_PORT"),
	}, nil
}