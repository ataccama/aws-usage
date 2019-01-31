package main

import (
	"context"
	"time"

	"github.com/spf13/viper"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
)

// GetClientAndProcessor connects to Elasticsearch and creates a bulk
// processor and returns both the client that's been obtained and the
// bulk processor
func GetClientAndProcessor(ctx context.Context, server string, workers int) (*elastic.Client, *elastic.BulkProcessor, error) {
	client, err := elastic.NewClient(elastic.SetURL(server))
	if err != nil {
		return nil, nil, err
	}
	processor, err := client.BulkProcessor().
		Name("usage-processor").
		Workers(workers).
		After(indexingErrorHandler).
		Do(ctx)
	if err != nil {
		return nil, nil, err
	}
	return client, processor, nil
}

// RecreateIndex creates an index for data from the given Manifest,
// dropping the same-named index if it already exists
func RecreateIndex(ctx context.Context, client *elastic.Client, m *Manifest) error {
	index := m.IndexName()
	exists, err := client.IndexExists(index).Do(ctx)
	if err != nil {
		return err
	}
	if exists {
		log.WithField("index", index).Info("Deleting outdated index")
		_, err = client.DeleteIndex(index).Do(ctx)
		if err != nil {
			return err
		}
	}
	mapping := getMappings(m)
	mapping["settings"] = map[string]map[string]int{
		"index": map[string]int{
			"number_of_shards": 3,
		},
	}
	_, err = client.CreateIndex(index).BodyJson(mapping).Do(ctx)
	return err
}

// getMappings creates body for
func getMappings(m *Manifest) map[string]interface{} {
	type properties struct {
		Properties interface{} `json:"properties"`
	}
	categories := make(map[string]properties)
	for _, col := range m.Columns {
		cat, ok := categories[col.NormalizedCategory()]
		if !ok {
			cat = properties{make(map[string]interface{})}
			categories[col.NormalizedCategory()] = cat
		}
		cat.Properties.(map[string]interface{})[col.NormalizedName()] = columnMapping(col)
	}
	body := map[string]properties{
		"_doc": properties{categories},
	}
	return map[string]interface{}{"mappings": body}
}

func columnMapping(c Column) map[string]interface{} {
	props := make(map[string]interface{})
	awsType := c.NormalizedType()
	switch awsType {
	case StringType:
		props["type"] = "keyword"
	case DateTimeType:
		props["type"] = "date"
	case BigDecimalType:
		props["type"] = "scaled_float"
		props["scaling_factor"] = 1e8
	case IntervalType:
		props["type"] = "date_range"
		props["format"] = "strict_date_optional_time"
	}
	return props
}

func indexingErrorHandler(executionID int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	if err != nil {
		log.WithError(err).Error("Bulk index error")
	} else {
		log.WithFields(log.Fields{
			"id":   executionID,
			"docs": len(requests),
		}).Debug("Documents commited")
	}
}

// CreateMetadataIndex tries to create an index for storing report
// ingestion metadata. If the index already exists, it does nothing.
func CreateMetadataIndex(ctx context.Context, client *elastic.Client) error {
	index := viper.GetString("es.metadata_index")
	exists, err := client.IndexExists(index).Do(ctx)
	if err == nil && !exists {
		_, err = client.CreateIndex(index).Body(metadataIndexBody).Do(ctx)
	}
	return err
}

// GetAssemblyIDForPeriod gets the ID of the last assembly for the given period from
// metadata index. If no ID is stored yet, it returns an empty string.
func GetAssemblyIDForPeriod(ctx context.Context, period string, client *elastic.Client) (string, error) {
	index := viper.GetString("es.metadata_index")
	doc, err := client.Get().Index(index).Id(period).StoredFields("assembly_id").Do(ctx)
	if err != nil && !elastic.IsNotFound(err) {
		return "", err
	}
	if elastic.IsNotFound(err) {
		return "", nil
	}
	return doc.Fields["assembly_id"].([]interface{})[0].(string), nil
}

// SetAssemblyIDForPeriod overwrites the assembly ID for the given period in metadata index.
func SetAssemblyIDForPeriod(ctx context.Context, period string, id string, client *elastic.Client) error {
	index := viper.GetString("es.metadata_index")
	_, err := client.Index().Id(period).Index(index).Type("_doc").BodyJson(map[string]string{
		"assembly_id": id,
		"last_update": time.Now().Format(time.RFC3339),
	}).Do(ctx)
	return err
}

const metadataIndexBody = `{
	"settings": {
		"index": {
			"number_of_shards": 1
		}
	}, 
	"mappings": {
		"_doc": {
			"properties": {
				"assembly_id": {"type": "text", "store": true},
				"last_update": {"type": "date"}
			}
		}
	}
}`
