# yell

A toolset to work with Graphite whisper files.

## Installation

### From Source

```bash
go install github.com/ljurk/yell@latest
```

### Build from Repository

```bash
git clone https://github.com/ljurk/yell.git
cd yell
go build -o yell .
```

## Features

### 1. Schema Check (`yell schema check`)

Check if whisper files match the defined retentions in `storage-schemas.conf`.

```bash
yell schema check /var/lib/graphite/whisper -s storage-schemas.conf
```

**Options:**
- `-s, --schema string` - Path to storage-schemas.conf (required)
- `--root-dir string` - Root directory for metric name calculation (optional)

**Usage with single file:**
```bash
yell schema check /var/lib/graphite/whisper/server.wsp --root-dir /var/lib/graphite/whisper -s storage-schemas.conf
```

### 2. Schema Apply (`yell schema apply`)

Resize whisper files to match the defined retentions in `storage-schemas.conf`.

```bash
yell schema apply /var/lib/graphite/whisper -s storage-schemas.conf
```

**Options:**
- `-s, --schema string` - Path to storage-schemas.conf (required)
- `--root-dir string` - Root directory for metric name calculation (optional)
- `-a, --aggregate string` - Aggregation method (default: "average")
- `-x, --xff float` - xFilesFactor (default: 0.5)
- `--compressed` - Use compressed format (default: keep current)

**Usage examples:**
```bash
# Apply with custom aggregation
yell schema apply /var/lib/graphite/whisper -s storage-schemas.conf -a sum

# Apply to single file
yell schema apply /var/lib/graphite/whisper/server.wsp --root-dir /var/lib/graphite/whisper -s storage-schemas.conf

# Apply with compressed output
yell schema apply /var/lib/graphite/whisper -s storage-schemas.conf --compressed
```

### 3. Schema Count (`yell schema count`)

Count matching metrics per schema definition.

```bash
yell schema count /var/lib/graphite/whisper -s storage-schemas.conf
```

**Options:**
- `-s, --schema string` - Path to storage-schemas.conf (required)
- `--root-dir string` - Root directory for metric name calculation (optional)

### 4. Info (`yell info`)

Dump information about a whisper file.

```bash
yell info /var/lib/graphite/whisper/server.wsp
```

## Storage Schema Format

The `storage-schemas.conf` file follows the standard Graphite format:

```ini
[default]
pattern = .*
retentions = 10s:7d,1m:53d,1h:5y

[carbon]
pattern = ^carbon\.
retentions = 1m:90d

[servers]
pattern = ^servers\.
retentions = 10s:1h,1m:7d,1h:2y
```

### Retention Format

Retentions are specified as `resolution:retention` pairs:
- `10s:7d` - 10 seconds per point, retention of 7 days
- `1m:30d` - 1 minute per point, retention of 30 days
- `1h:1y` - 1 hour per point, retention of 1 year

Multiple retentions can be combined with commas.

### Pattern Matching

- Patterns are regular expressions
- First matching schema (top-to-bottom) wins
- A default schema matching `.*` is required for `yell schema apply`

## Examples

### Check all files in a directory

```bash
yell schema check /var/lib/whisper -s storage-schemas.conf
```

Output:
```
status    metric                    expected           actual             detail
OK        servers.web01.cpu         10s:1h,1m:7d       10s:1h,1m:7d       matched schema[servers]
MISMATCH  servers.web01.memory      expected:10s:1h    got:1m:7d          schema[servers]
```

### Resize files to match schema

```bash
yell schema apply /var/lib/whisper -s storage-schemas.conf
```

Output:
```
status   metric                  old              new              detail
RESIZED  servers.web01.memory    1m:7d            10s:1h           -
SKIP     servers.web01.cpu       10s:1h,1m:7d    10s:1h,1m:7d     already matching
```

### Check specific files

```bash
yell schema check /var/lib/whisper/server.wsp --root-dir /var/lib/whisper -s storage-schemas.conf
```

This allows you to check/apply changes to specific files while using a different root directory for metric name calculation.

## Development

### Building

```bash
go build -v ./...
```

### Linting

```bash
golangci-lint run ./...
```

See [AGENTS.md](AGENTS.md) for detailed development instructions.

### Testing

```bash
go test -v ./...
```

## License

MIT