package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// LoadConfig loads configuration from files into the provided target struct.
func LoadConfig(configName string, target interface{}, log *zap.Logger) error {
	v := viper.New()
	v.SetConfigName(strings.ToLower(configName))
	v.SetConfigType("yaml")

	// Add common configuration paths
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Load environment-specific configuration files with precedence
	envConfigs := []string{"GOLETAN_PROD_CONFIG", "GOLETAN_STAGE_CONFIG", "GOLETAN_LOCAL_CONFIG"}
	for _, envVar := range envConfigs {
		envValue := os.Getenv(envVar)
		if envValue != "" {
			configPath := fmt.Sprintf("./config/%s.yaml", envValue)
			loadConfigFiles([]string{configPath}, v, log)
		}
	}

	// Load common configuration files
	loadConfigFiles([]string{
		"./config/override.yaml",
		"./config/tests.yaml",
	}, v, log)

	// Read the configuration file
	if err := v.ReadInConfig(); err != nil {
		if log != nil {
			log.Error("Failed to read configuration file", zap.Error(err))
		}
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Unmarshal the configuration into the target struct
	if err := v.Unmarshal(target); err != nil {
		if log != nil {
			log.Error("Failed to parse configuration", zap.Error(err))
		}
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Set up hot-reloading
	v.OnConfigChange(func(e fsnotify.Event) {
		if log != nil {
			log.Info("Configuration file changed", zap.String("file", e.Name))
		}

		if err := v.Unmarshal(target); err != nil {
			if log != nil {
				log.Error("Failed to reload configuration", zap.Error(err))
			}
		} else {
			if log != nil {
				log.Info("Configuration reloaded successfully")
			}
			// Update the cache after reloading
			StoreConfigInCache(configName, target)
		}
	})
	v.WatchConfig()

	if log != nil {
		log.Info("Configuration loaded successfully")
	}

	// Cache the loaded configuration
	StoreConfigInCache(configName, target)

	return nil
}

// loadConfigFiles attempts to load a list of configuration files in order, with precedence.
func loadConfigFiles(configFiles []string, v *viper.Viper, log *zap.Logger) {
	for _, configPath := range configFiles {
		if _, err := os.Stat(configPath); err == nil {
			v.SetConfigFile(configPath)
			if err := v.MergeInConfig(); err != nil {
				if log != nil {
					log.Warn("Failed to merge configuration file", zap.String("file", configPath), zap.Error(err))
				}
			}
		}
	}
}