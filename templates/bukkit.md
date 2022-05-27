# Bukkit / Spigot / Paper template

```yaml
server-tag: "server"
display-name: "Server"
log-file-path: "/path/to/server/logs/latest.log"
syntax-highlighting:
    -   field: "time"
        regex: '^\[\d{2}:\d{2}:\d{2}]'
    -   field: "info"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,30}\/INFO]'
    -   field: "warn"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,30}\/WARN]'
    -   field: "error"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,30}\/ERROR]'
    -   field: "content"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}] \[.{0,30}\/(INFO|WARN|ERROR)]: )).*$'
archived-logs-dir-path: "/path/to/server/logs"
archive-log-filename-format: "*.log.gz"
```
