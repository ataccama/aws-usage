package main

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func setUpConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/aws-usage/")
	viper.AddConfigPath("$HOME/.aws-usage")
	viper.AddConfigPath(".")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	viper.SetDefault("es.server", "http://localhost:9200")
	viper.SetDefault("es.report_index_prefix", "billing")
	viper.SetDefault("es.metadata_index", "aws-usage-metadata")
	viper.SetDefault("es.indexing_workers", "2")
	err := viper.ReadInConfig()
	if err != nil {
		log.WithError(err).Fatal("Error reading configuration")
	}
	reportIndexPrefix = viper.GetString("es.report_index_prefix")
}
