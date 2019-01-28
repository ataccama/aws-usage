package main

import (
	"encoding/csv"
	"io"
	"strings"

	"github.com/olivere/elastic"
)

// BulkIndexCsv reads a CSV file from the given reader, ignores the first row
// and indexes the remainder of the file according to the column list from the manifest
func BulkIndexCsv(m *Manifest, r io.Reader, processor *elastic.BulkProcessor) error {
	csvReader := csv.NewReader(r)
	var err error
	_, err = csvReader.Read() // First row is the header
	if err != nil {
		return err
	}
	for record, err := csvReader.Read(); err == nil; record, err = csvReader.Read() {
		req := elastic.NewBulkIndexRequest().
			Index(m.IndexName()).
			Type("_doc").
			Doc(parseCSVRecord(record, m.Columns))
		processor.Add(req)
	}
	if err != io.EOF {
		return err
	}
	return nil
}

func parseCSVRecord(record []string, columns []Column) map[string]map[string]interface{} {
	parsed := make(map[string]map[string]interface{})
	for i, data := range record {
		if data == "" {
			continue
		}
		column := &columns[i]
		cat, ok := parsed[column.NormalizedCategory()]
		if !ok {
			cat = make(map[string]interface{})
			parsed[column.NormalizedCategory()] = cat
		}
		if column.NormalizedType() == IntervalType {
			components := strings.Split(data, "/")
			cat[column.NormalizedName()] = map[string]string{
				"gte": components[0],
				"lte": components[1],
			}
		} else {
			cat[column.NormalizedName()] = data
		}
	}
	return parsed
}
