package config

import (
	"errors"
	"io/fs"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort                        string `mapstructure:"SERVER_PORT"`
	RedisAddr                         string `mapstructure:"REDIS_ADDR"`
	IpRequestsPerSecondLimitter       int    `mapstructure:"IP_REQS_PER_SEC_LIMITTER"`
	RequestBlockTime                  int    `mapstructure:"REQS_BLOCK_TIME_SEC"` // em segundos
	ApiToken                          string `mapstructure:"API_TOKEN"`
	ApiTokenRequestsPerSecondLimitter int    `mapstructure:"API_TOKEN_REQS_PER_SEC_LIMITTER"`
	TrustProxyHeaders                 bool   `mapstructure:"TRUST_PROXY_HEADERS"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("app_rate_limit")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	// AutomaticEnv resolve cada chave pelo nome da variável em maiúsculas (ex: SERVER_PORT).
	// Não use viper.BindEnv com várias chaves: os argumentos extras viram aliases da primeira
	// chave, fazendo, por exemplo, SERVER_PORT ler o valor de REDIS_ADDR.
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// Read each key directly: Unmarshal only sees keys present in the file,
	// which drops env-only values when the file is absent.
	cfg := &Config{
		ServerPort:                        viper.GetString("SERVER_PORT"),
		RedisAddr:                         viper.GetString("REDIS_ADDR"),
		IpRequestsPerSecondLimitter:       viper.GetInt("IP_REQS_PER_SEC_LIMITTER"),
		RequestBlockTime:                  viper.GetInt("REQS_BLOCK_TIME_SEC"),
		ApiToken:                          viper.GetString("API_TOKEN"),
		ApiTokenRequestsPerSecondLimitter: viper.GetInt("API_TOKEN_REQS_PER_SEC_LIMITTER"),
		TrustProxyHeaders:                 viper.GetBool("TRUST_PROXY_HEADERS"),
	}

	if cfg.IpRequestsPerSecondLimitter <= 0 ||
		cfg.ApiTokenRequestsPerSecondLimitter <= 0 ||
		cfg.RequestBlockTime <= 0 {
		return nil, errors.New("IP_REQS_PER_SEC_LIMITTER, API_TOKEN_REQS_PER_SEC_LIMITTER and REQS_BLOCK_TIME_SEC must be positive integers")
	}

	return cfg, nil
}
