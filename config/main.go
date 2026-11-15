package config

import (
	"errors"
	"io/fs"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
	RedisAddr string `mapstructure:"REDIS_ADDR"`
	IpRequestsPerSecondLimitter int `mapstructure:"IP_REQS_PER_SEC_LIMITTER"`
	RequestBlockTime time.Duration `mapstructure:"REQS_BLOCK_TIME_SEC"`
	ApiToken string `mapstructure:"API_TOKEN"`
	ApiTokenRequestsPerSecondLimitter int `mapstructure:"API_TOKEN_REQS_PER_SEC_LIMITTER"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("app_rate_limit")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	// BindEnv only fails when called without key names, so its error is not reachable here.
	_ = viper.BindEnv(
		"SERVER_PORT", 
		"REDIS_ADDR", 
		"REQS_PER_SEC_LIMITTER", 
		"IP_REQS_PER_SEC_LIMITTER", 
		"API_TOKEN",
		"API_TOKEN_REQS_PER_SEC_LIMITTER",
	)
	if err := viper.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// Read each key directly: Unmarshal only sees keys present in the file,
	// which drops env-only values when the file is absent.
	return &Config{
		ServerPort: viper.GetString("SERVER_PORT"),
		RedisAddr: viper.GetString("REDIS_ADDR"),
		IpRequestsPerSecondLimitter: viper.GetInt("IP_REQS_PER_SEC_LIMITTER"),
		RequestBlockTime: viper.GetDuration("REQS_BLOCK_TIME_SEC"),
		ApiToken: viper.GetString("API_TOKEN"),
		ApiTokenRequestsPerSecondLimitter: viper.GetInt("API_TOKEN_REQS_PER_SEC_LIMITTER"),
	}, nil
}