# SQLite P2P Storage & Replication Guide

This guide explains how to use **`sqlite-p2p`** as the default storage engine for both the **DAO Persistence Layer (`IPrimaryDao`)** and the **FileStore Asset Storage Layer (`IFileStore`)** in `AIGenApp`.

---

## Overview

`sqlite-p2p` is a decentralized, peer-to-peer storage engine powered by pure-Go SQLite (`modernc.org/sqlite`) and Holepunch/Pear P2P networking protocols (`go-pear`).

Key features:
- **Zero-CGO Pure Go**: Runs seamlessly across platforms without external C compiler dependencies.
- **Single-Table JSON Model**: Compatible with `AIGenApp`'s `aigen_records` schema-on-read persistence and SQLite's native `json_extract()` querying.
- **Peer-to-Peer Replication**: Background synchronization across peer nodes using append-only changeset feeds (Hypercore) and Hyperswarm DHT discovery.
- **Integrated Binary Storage**: Binary asset and chunked multi-part upload support in a synchronized `filesys` table.

---

## 1. Using SQLite P2P for the DAO Layer (`IPrimaryDao`)

`AIGenApp` automatically uses `sqlite-p2p` as the default database engine when `FORMCMS_DB_DSN` or `database_dsn` is omitted or configured with a `sqlite-p2p://` or `p2p://` scheme.

### Connection String Schemes & Parameters

| Connection URI Format | Description |
| :--- | :--- |
| `""` or `default` | Uses default local database `data/aigen.db` with WAL mode enabled. |
| `sqlite-p2p://:memory:` | In-memory ephemeral SQLite instance (ideal for fast automated testing). |
| `sqlite-p2p://data/aigen.db` | Local file database at `data/aigen.db`. |
| `p2p://data/aigen.db?topic=aigen-cluster-1&port=4001&wal=true` | P2P node joining swarm topic `aigen-cluster-1` on port `4001` with WAL mode. |
| `p2p://data/node2.db?topic=aigen-cluster-1&bootstrap=192.168.1.10:4001` | Joins existing peer cluster using bootstrap node address. |

### Query Parameters

- `wal` (`true` / `false` / `1` / `0`): Enable/disable Write-Ahead Logging (`PRAGMA journal_mode=WAL`). Default: `true`.
- `crypto` (`true` / `false`): Enable/disable per-key AES-256-GCM encryption with crypto-shredding. Default: `false`.
- `topic` (string): 32-byte swarm cluster discovery topic for peer synchronization.
- `port` (int): UDP/TCP listening port for peer replication swarm.
- `bootstrap` (comma-separated string): Bootstrap peer addresses (e.g., `127.0.0.1:4001,192.168.1.50:4001`).

---

## 2. Using SQLite P2P for the FileStore Layer (`IFileStore`)

When `driver` is set to `sqlite-p2p`, `p2p`, or left empty (`""`), `AIGenApp` uses `SqliteP2PFileStore` to store uploaded assets and temporary files directly inside a synchronized SQLite database table (`filesys`).

### Key Capabilities

1. **Direct Upload / Download**: Stores file content as raw binary blobs alongside JSON metadata (file size, MIME content type, creation timestamps).
2. **Chunked & Resumable Uploads**: Multi-part chunk uploads (`UploadChunk`, `GetUploadedChunks`, `CommitChunks`) are assembled directly into the database.
3. **Prefix Deletion & Expired Purging**: Supports fast prefix cleanup and automated TTL deletion (`PurgeExpired`).
4. **URL Resolution**: Exposes public file URL endpoint (default prefix: `/api/files/p2p/{path}`).

---

## 3. Configuration

You can configure SQLite P2P via `config.yaml` or environment variables.

### Example `config.yaml`

```yaml
bizdefs_dir: "bizdefs"
www_root: "wwwroot"
port: "5000"

# 1. Database (DAO Layer)
database_dsn: "sqlite-p2p://data/aigen.db?topic=aigen-mesh-topic&port=4001&wal=true"

# 2. Storage (FileStore Layer)
storage:
  driver: "sqlite-p2p"
  sqlite_p2p:
    url: "sqlite-p2p://data/filestore.db?topic=aigen-mesh-topic&port=4002&wal=true"
    url_prefix: "/api/files/p2p"

# 3. Logging & Rotation
log:
  level: "INFO"
  console_enabled: true
  file_enabled: true
  dir: "logs"
  file_name: "server.log"
```

### Environment Variable Overrides

```bash
# Database DSN
export FORMCMS_DB_DSN="sqlite-p2p://data/aigen.db?topic=aigen-mesh-topic&port=4001"

# Port & Server configuration
export PORT="5000"
export FORMCMS_CONFIG_PATH="config.yaml"
```

---

## 4. Multi-Node Replication Setup Example

To run two synchronized nodes on the same host or local network:

### Node 1 (Primary / Seed)

```bash
FORMCMS_DB_DSN="p2p://data/node1.db?topic=my-cluster&port=4001" \
PORT="5001" \
go run main.go
```

### Node 2 (Peer)

```bash
FORMCMS_DB_DSN="p2p://data/node2.db?topic=my-cluster&port=4002&bootstrap=127.0.0.1:4001" \
PORT="5002" \
go run main.go
```

Any records or files created, modified, or deleted on Node 1 will automatically replicate to Node 2 via Hyperswarm P2P feeds.
