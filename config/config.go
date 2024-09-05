package config

import (
	"github.com/spf13/viper"
	"os"
)

/*
[gossip]
cache_size = 50
degree = 3
minConnections = 3
maxConnections = 30
bootstrapper =  127.0.0.1:1
p2pAddress = 127.0.0.1:6001
apiAddress = 127.0.0.1:7001
knownPeers = 127.0.0.1:6002, 127.0.0.1:6003
*/

type Config struct {
	CacheSize      int      `mapstructure:"cache_size"`
	Degree         int      `mapstructure:"degree"`
	MinConnections int      `mapstructure:"minConnections"`
	MaxConnections int      `mapstructure:"maxConnections"`
	Bootstrapper   string   `mapstructure:"bootstrapper"`
	P2PAddress     string   `mapstructure:"p2pAddress"`
	APIAddress     string   `mapstructure:"apiAddress"`
	KnownPeers     []string `mapstructure:"knownPeers"`
}

var P2PConfig *Config

// LoadConfig load config via viper from file from the path
// Input config path will indicate the path of the config file
func LoadConfig(configPath string) (*Config, error) {
	//check whether the file exits
	fileExists := fileExists(configPath)
	if !fileExists {
		return nil, os.ErrNotExist
	}
	// open the file with configPath
	viper.SetConfigFile(configPath)
	// read the file
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	P2PConfig = &Config{}
	//unmarshal the config file to the struct
	err = viper.UnmarshalKey("gossip", P2PConfig)
	if err != nil {
		return nil, err
	}
	return P2PConfig, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
