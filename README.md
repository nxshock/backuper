# backuper

## Usage

### Incremental backup

```sh
backuper i <config file path>
```

### Full backup

```sh
backuper f <config file path>
```

### Search files in backup

```sh
backuper s <config file path> <mask>
```

### Recover files from backup

```sh
backuper r <config file path> <mask> <files datetime> <path to recover>
```

Examples:

```sh
# Recover Go files relevant as of 01.01.2023 to /home/user/go directory
backuper r config.conf "*.go" "01.01.2023" "/home/user/go"
```

### Test backup for errors

```sh
backuper t <config file path>
```

## Basic config example

Backup config files from `/etc` and sqlite files from `/var`:

```toml
FileName = "backup"

[[Patterns]]
Path = "/etc"
FileNamePatternList = ["*.conf", "*.toml", "*.ini", "*.yaml"]
Recursive = true

[[Patterns]]
Path = "/var"
FileNamePatternList = ["*.sqlite"]
Recursive = true
```
