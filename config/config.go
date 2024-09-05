package config

import (
	"github.com/spf13/viper"
	"os"
)

//
//[log]
//level = debug
//destination = stdout
//logFile = ../logs/config_1.log

type GossipConfig struct {
	CacheSize      int      `mapstructure:"cache_size"`
	Degree         int      `mapstructure:"degree"`
	MinConnections int      `mapstructure:"minConnections"`
	MaxConnections int      `mapstructure:"maxConnections"`
	Bootstrapper   string   `mapstructure:"bootstrapper"`
	P2PAddress     string   `mapstructure:"p2pAddress"`
	APIAddress     string   `mapstructure:"apiAddress"`
	KnownPeers     []string `mapstructure:"knownPeers"`
}

type LogConfig struct {
	Level       string `mapstructure:"level"`
	Destination string `mapstructure:"destination"`
	LogFile     string `mapstructure:"logFile"`
}

var P2PConfig *GossipConfig
var LoggerConfig *LogConfig

// LoadConfig load config via viper from file from the path
// Input config path will indicate the path of the config file
func LoadConfig(configPath string) (*GossipConfig, error) {
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
	P2PConfig = &GossipConfig{}
	LoggerConfig = &LogConfig{}
	//unmarshal the config file to the struct
	err = viper.UnmarshalKey("gossip", P2PConfig)
	if err != nil {
		return nil, err
	}
	err = viper.UnmarshalKey("log", LoggerConfig)
	if err != nil {
		return nil, err
	}
	return P2PConfig, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
