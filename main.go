package main

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	log.SetLevel(log.TraceLevel)
	setUpConfig()
}

func main() {
	if !viper.IsSet("s3.bucket") {
		log.Fatal("No S3 bucket name set in configuration")
	}
	ctx := context.Background()
	client, processor, err := GetClientAndProcessor(ctx, viper.GetString("es.server"), viper.GetInt("es.workers"))
	if err != nil {
		log.WithError(err).Fatal("Couldn't get Elasticsearch client")
	}
	defer logDeferredError(processor.Close)
	defer client.Stop()

	sess, err := session.NewSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't create an AWS session")
	}
	d, err := NewS3Downloader(ctx, sess, viper.GetString("s3.bucket"), viper.GetString("s3.prefix"))
	if err != nil {
		log.WithError(err).Fatal("Couldn't get S3 region")
	}
	periods, err := d.Periods()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get S3 bucket directory list")
	}
	if len(periods) == 0 {
		log.Warn("No prefixes with reports were found, check whether your bucket name or prefix is correct")
	}
	err = CreateMetadataIndex(ctx, client)
	if err != nil {
		log.WithError(err).WithField("index_name", viper.GetString("es.metadata_index")).Fatal("Couldn't create metadata index")
	}
	var wg sync.WaitGroup
	wg.Add(len(periods))
	for _, period := range periods {
		go func(p string) {
			defer wg.Done()
			ProcessPeriod(ctx, p, client, processor, &d)
		}(period)
	}
	wg.Wait()
	err = processor.Flush()
	if err != nil {
		log.WithError(err).Fatal("Flushing remaining records to Elasticsearch failed")
	}
	log.Info("Indexing done")
}
