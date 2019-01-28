# aws-usage

AWS usage reports can be surprisingly hard to parse. They have complicated and varying 
column names which are described in a JSON manifest that's separate from the CSV file
and updates are handled by uploading a new report and changing the manifest to contain
the current path. They also contain relatively rich types like time intervals or
arbitrary precision decimals.

Due to this, tools like Logstash are unable to correctly parse these files and forward them
to Elasticsearch, which could then be used to query and aggregate the data. That's where
aws-usage comes into the picture. It downloads the manifests and reports, drops outdated
indices, recreates them with correct mappings and populates them with the contents of the
reports.

Configuration reference is located in the `config.example.toml` file. It's also recommended
to read AWS Usage Report documentation 
[here](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-reports-costusage.html).
Environment variables (with `_` used as a nested field separator) can be used to override data from
file-based configuration.