# BungeeCord template

```yaml
server-tag: "proxy"
display-name: "Proxy"
log-file-path: "/path/to/proxy/logs/latest.log"
syntax-highlighting:
    -   field: "time"
        regex: '^\[\d{2}:\d{2}:\d{2}]'
    -   field: "info"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,35}\/INFO]'
    -   field: "warn"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,35}\/WARN]'
    -   field: "error"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,35}\/ERROR]'
    -   field: "content"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}] \[.{0,35}\/(INFO|WARN|ERROR)]: )).*$'
archived-logs-dir-path: "/path/to/proxy/logs"
archive-log-filename-format: "*.log.gz"
```
