package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	//log "github.com/sirupsen/logrus"
	"strings"
)

// S3Downloader finds and downloads AWS Usage reports from an S3 bucket
type S3Downloader struct {
	ctx    context.Context
	bucket string
	prefix string
	sess   *session.Session
	client *s3.S3
}

// NewS3Downloader creates a new S3Downloader
func NewS3Downloader(ctx context.Context, sess *session.Session, bucket string, prefix string) (S3Downloader, error) {
	region, err := s3manager.GetBucketRegion(context.Background(), sess, bucket, "eu-central-1")
	if err != nil {
		return S3Downloader{}, err
	}
	if prefix[0] == '/' {
		prefix = prefix[1:]
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	return S3Downloader{
		ctx:    ctx,
		bucket: bucket,
		sess:   sess,
		prefix: prefix,
		client: s3.New(sess, &aws.Config{
			Region: aws.String(region),
		}),
	}, nil
}

// Periods returns a slice of periods reports are available for
func (d *S3Downloader) Periods() ([]string, error) {
	res, err := d.client.ListObjectsV2WithContext(d.ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(d.prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(res.CommonPrefixes))
	for i, obj := range res.CommonPrefixes {
		keys[i] = (*obj.Prefix)[len(d.prefix) : len(*obj.Prefix)-1]
	}
	return keys, nil
}

// ManifestForPeriod returns the Manifest object corresponding to the latest
// spending report for that period
func (d *S3Downloader) ManifestForPeriod(period string) (*Manifest, error) {
	res, err := d.client.ListObjectsV2WithContext(d.ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(d.prefix + period + "/"),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, err
	}
	manifestKey := getJSONFileKey(res.Contents)
	if manifestKey == "" {
		return nil, errors.New("report manifest not found")
	}
	manifestObject, err := d.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(manifestKey),
	})
	if err != nil {
		return nil, err
	}
	defer manifestObject.Body.Close()
	var manifest Manifest
	dec := json.NewDecoder(manifestObject.Body)
	err = dec.Decode(&manifest)
	return &manifest, err
}

func getJSONFileKey(objs []*s3.Object) string {
	for _, obj := range objs {
		if strings.HasSuffix(*obj.Key, ".json") {
			return *obj.Key
		}
	}
	return ""
}

// ReportsForManifest downloads usage reports specified in the manifest from S3
func (d *S3Downloader) ReportsForManifest(m *Manifest) ([][]byte, error) {
	objects := make([]s3manager.BatchDownloadObject, len(m.ReportKeys))
	buffers := make([]aws.WriteAtBuffer, len(m.ReportKeys))
	for i, key := range m.ReportKeys {
		objects[i].Object = &s3.GetObjectInput{
			Bucket: aws.String(d.bucket),
			Key:    aws.String(key),
		}
		objects[i].Writer = &buffers[i]
	}
	downloader := s3manager.NewDownloaderWithClient(d.client)
	if err := downloader.DownloadWithIterator(d.ctx, &s3manager.DownloadObjectsIterator{Objects: objects}); err != nil {
		return nil, err
	}
	results := make([][]byte, len(m.ReportKeys))
	for i := range results {
		results[i] = buffers[i].Bytes()
	}
	return results, nil
}
