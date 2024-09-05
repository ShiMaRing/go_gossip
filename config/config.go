package config

import (
	"github.com/spf13/viper"
	"os"
)

type PeerConfig struct {
	MinConnections int
	MaxConnections int
	Bootstraps     []string
	PeerAddr       string
	ApiAddr        string
	KnownPeers     []string
}

type GossipConfig struct {
	CacheSize int
	Degree    int
}

type Config struct {
	Peer   PeerConfig
	Gossip GossipConfig
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
	err = viper.Unmarshal(P2PConfig)
	if err != nil {
		return nil, err
	}
	return P2PConfig, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
