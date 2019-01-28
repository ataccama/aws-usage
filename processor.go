package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"sync"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
)

// ProcessPeriod downloads the manifest for the given report period and if
// a newer report than the current one is available, it downloads it and
// replaces data in Elasticsearch.
func ProcessPeriod(ctx context.Context, period string, esClient *elastic.Client, processor *elastic.BulkProcessor, downloader *S3Downloader) {
	log.WithField("period", period).Debug("Starting processing")
	m, err := downloader.ManifestForPeriod(period)
	if err != nil {
		logProcessingError(err, period, "Couldn't download manifest")
		return
	}
	lastReport, err := GetReportIDForPeriod(ctx, period, esClient)
	if err != nil {
		logProcessingError(err, period, "Couldn't get previous report ID")
		return
	}
	if lastReport == m.ReportID {
		log.WithField("period", period).Infof("No new data for period")
		return
	}
	err = RecreateIndex(ctx, esClient, m)
	if err != nil {
		logProcessingError(err, period, "Couldn't recreate index")
		return
	}
	csvs, err := downloader.ReportsForManifest(m)
	if err != nil {
		logProcessingError(err, period, "Couldn't download spending reports")
		return
	}
	var indexingWg sync.WaitGroup
	indexingWg.Add(len(csvs))
	for _, csv := range csvs {
		log.WithField("period", period).Debug("Indexing csv")
		go func(c []byte) {
			defer indexingWg.Done()
			indexByteReport(context.WithValue(ctx, "period", period), processor, m, c)
		}(csv)
	}
	indexingWg.Wait()
	err = SetReportIDForPeriod(ctx, period, m.ReportID, esClient)
	if err != nil {
		logProcessingError(err, period, "Couldn't set report ID")
		return
	}
}

func logProcessingError(err error, period, msg string) {
	log.WithFields(log.Fields{
		"error":  err,
		"period": period,
	}).Error(msg)
}

func indexByteReport(ctx context.Context, processor *elastic.BulkProcessor, m *Manifest, report []byte) {
	buf := bytes.NewBuffer(report)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		logProcessingError(err, ctx.Value("period").(string), "GZip reader creation failed")
		return
	}
	defer logDeferredError(gz.Close)
	err = BulkIndexCsv(m, gz, processor)
	if err != nil {
		logProcessingError(err, ctx.Value("period").(string), "Couldn't parse report CSV")
	}
}

func logDeferredError(f func() error) {
	err := f()
	if err != nil {
		log.WithError(err).Error("Error when running deferred function")
	}
}
