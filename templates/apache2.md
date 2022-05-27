# Apache2 templates

## Access logs

```yaml
server-tag: "apache-access"
display-name: "Apache (access)"
log-file-path: "/var/log/apache2/access.log"
syntax-highlighting:
    -   field: "info"
        regex: '^(\d{1,3}\.){3}\d{1,3}'
    -   field: "time"
        regex: '(?<=(^(\d{1,3}\.){3}\d{1,3}\s-\s-\s))\[\d{1,2}\/\w{1,15}\/\d{4}(:\d{2}){3}\s\+\d{4}\]'
    -   field: "content"
        regex: '(?<=(^(\d{1,3}\.){3}\d{1,3}\s-\s-\s\[\d{1,2}\/\w{1,15}\/\d{4}(:\d{2}){3}\s\+\d{4}\]\s)).+$'
archived-logs-dir-path: "/var/log/apache2"
archive-log-filename-format: "access.log.*"
```

## Error logs

```yaml
server-tag: "apache-error"
display-name: "Apache (error)"
log-file-path: "/var/log/apache2/error.log"
syntax-highlighting:
    -   field: "time"
        regex: '^\[\w{3}\s\w{3}\s\d{2}\s\d{2}:\d{2}:\d{2}\.\d{6}\s\d{4}\]'
    -   field: "warn"
        regex: '(?<=(^\[\w{3}\s\w{3}\s\d{2}\s\d{2}:\d{2}:\d{2}\.\d{6}\s\d{4}\]\s)).+$'
    -   field: "error"
        regex: '^\w{2}\d{5}'
    -   field: "content"
        regex: '(?<=(^\w{2}\d{5}:\s)).+$'
archived-logs-dir-path: "/var/log/apache2"
archive-log-filename-format: "error.log.*"
```
