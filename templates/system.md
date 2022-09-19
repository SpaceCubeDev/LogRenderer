# System template

```yaml
server-tag: "system"
display-name: "Système"
log-file-path: "/var/log/syslog"
syntax-highlighting:
    -   field: "time"
        regex: '^\w{3}\s+\d+\s(\d{2}:){2}\d{2}'
    -   field: "host"
        regex: '(?<=(^\w{3}\s+\d+\s(\d{2}:){2}\d{2}\s))[\w\d-]+'
    -   field: "user"
        regex: '(?<=(^\w{3}\s+\d+\s(\d{2}:){2}\d{2}\s[\w\d-]+\s))[\w\d\.\[\]-]+'
    -   field: "info"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[INFO]'
    -   field: "warn"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,64}\/WARN]'
    -   field: "error"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}]) )\[.{0,64}\/ERROR]'
    -   field: "ufw"
        regex: '\[UFW BLOCK\]'
    -   field: "content"
        regex: '(?<=(^\[\d{2}:\d{2}:\d{2}] \[.{0,64}\/(INFO|WARN|ERROR)]: )).*$'
archived-logs-dir-path: "/var/log"
# The archived log reader supports plain text and gzip plain text files
archived-logs-filename-format: "syslog.*"
```