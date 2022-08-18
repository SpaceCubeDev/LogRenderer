# Nginx templates

## Access logs

```yaml
server-tag: "nginx-access"
display-name: "Nginx (access)"
log-file-path: "/var/log/nginx/access.log"
syntax-highlighting:
    -   field: "info"
        regex: '^(\d{1,3}\.){3}\d{1,3}'
    -   field: "time"
        regex: '(?<=(^(\d{1,3}\.){3}\d{1,3}\s-\s(-|[\w-]+)\s))\[\d{1,2}\/\w{1,15}\/\d{4}(:\d{2}){3}\s\+\d{4}\]'
    -   field: "content"
        regex: '(?<=(^(\d{1,3}\.){3}\d{1,3}\s-\s(-|[\w-]+)\s\[\d{1,2}\/\w{1,15}\/\d{4}(:\d{2}){3}\s\+\d{4}\]\s)).+$'
archived-logs-dir-path: "/var/log/nginx"
archive-log-filename-format: "access.log.*"
```

## Error logs

```yaml
server-tag: "nginx-error"
display-name: "Nginx (error)"
log-file-path: "/var/log/nginx/error.log"
syntax-highlighting:
    -   field: "time"
        regex: '^\d{4}\/\d{2}\/\d{2}\s\d{2}:\d{2}:\d{2}'
    -   field: "warn"
        regex: '(?<=^\d{4}\/\d{2}\/\d{2}\s\d{2}:\d{2}:\d{2}\s)\[warn]'
    -   field: "error"
        regex: '(?<=^\d{4}\/\d{2}\/\d{2}\s\d{2}:\d{2}:\d{2}\s)\[crit]'
    -   field: "content"
        regex: '(?<=^\d{4}\/\d{2}\/\d{2}\s\d{2}:\d{2}:\d{2}\s\[\w+]\s\d+#\d+:\s).*$'
archived-logs-dir-path: "/var/log/nginx"
archive-log-filename-format: "error.log.*"
```