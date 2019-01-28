package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	camelCaseRegexp   = regexp.MustCompile(`([a-z])([A-Z])`)
	reportIndexPrefix string
)

// AwsType describes the type that a value in a spending report has
type AwsType uint8

const (
	unknownType    AwsType = iota
	StringType     AwsType = iota
	DateTimeType   AwsType = iota
	BigDecimalType AwsType = iota
	IntervalType   AwsType = iota
)

// Manifest represents data read from a billing report manifest
type Manifest struct {
	AssemblyID    string   `json:"assemblyId"`
	Account       string   `json:"account"`
	Columns       []Column `json:"columns"`
	Charset       string   `json:"charset"`
	Compression   string   `json:"compression"`
	ContentType   string   `json:"contentType"`
	ReportID      string   `json:"reportId"`
	ReportName    string   `json:"reportName"`
	BillingPeriod struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"billingPeriod"`
	Bucket     string   `json:"bucket"`
	ReportKeys []string `json:"reportKeys"`
}

// IndexName returns the name of the elasticsearch index
// containing data from this manifest's report
func (m *Manifest) IndexName() string {
	start := strings.Split(m.BillingPeriod.Start, "T")[0]
	end := strings.Split(m.BillingPeriod.End, "T")[0]
	return fmt.Sprintf("billing-%s-%s", start, end)
}

// Column contains metadata of a column from the billing report CSV
type Column struct {
	Category           string `json:"category"`
	normalizedCategory string
	Name               string `json:"name"`
	normalizedName     string
	Type               string `json:"type"`
	normalizedType     AwsType
}

// NormalizedName returns a snake_cased name of the column
// with colons replaced with underscores
func (c *Column) NormalizedName() string {
	if c.normalizedName == "" {
		c.normalizedName = strings.Replace(snakeCase(c.Name), ":", "_", -1)
	}
	return c.normalizedName
}

// NormalizedCategory returns a snake_cased name of the category
func (c *Column) NormalizedCategory() string {
	if c.normalizedCategory == "" {
		c.normalizedCategory = snakeCase(c.Category)
	}
	return c.normalizedCategory
}

// NormalizedType returns AwsType of the column
func (c *Column) NormalizedType() AwsType {
	if c.normalizedType == unknownType {
		awsType := strings.Replace(c.Type, "Optional", "", 1)
		switch awsType {
		case "String":
			c.normalizedType = StringType
		case "DateTime":
			c.normalizedType = DateTimeType
		case "BigDecimal":
			c.normalizedType = BigDecimalType
		case "Interval":
			c.normalizedType = IntervalType
		}
	}
	return c.normalizedType
}

func snakeCase(s string) string {
	return strings.ToLower(camelCaseRegexp.ReplaceAllString(s, "${1}_${2}"))
}
