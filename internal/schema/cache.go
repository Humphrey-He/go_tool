package schema

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Snapshot struct {
	Schema       Schema `json:"schema"`
	ChecksumMD5  string `json:"checksum_md5"`
	GeneratedAt  int64  `json:"generated_at"`
	Source       string `json:"source"`
	SearchPath   []string `json:"search_path"`
}

type Cache struct {
	Path string
	TTL  time.Duration
}

func (c Cache) Load() (Snapshot, bool, error) {
	if c.Path == "" || c.TTL == 0 {
		return Snapshot{}, false, nil
	}
	data, err := os.ReadFile(c.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, false, err
	}
	if time.Since(time.Unix(snap.GeneratedAt, 0)) > c.TTL {
		return snap, false, nil
	}
	return snap, true, nil
}

func (c Cache) Store(snap Snapshot) error {
	if c.Path == "" {
		return nil
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, data, 0644)
}

func NewSnapshot(schema Schema, source string, searchPath []string) Snapshot {
	checksum := md5.Sum([]byte(schemaChecksumInput(schema)))
	return Snapshot{
		Schema:      schema,
		ChecksumMD5: hex.EncodeToString(checksum[:]),
		GeneratedAt: time.Now().Unix(),
		Source:      source,
		SearchPath:  append([]string{}, searchPath...),
	}
}

func schemaChecksumInput(schema Schema) string {
	data, _ := json.Marshal(schema)
	return string(data)
}

func ensureSearchPath(ctx context.Context, searchPath []string) []string {
	if len(searchPath) == 0 {
		return []string{"public"}
	}
	return append([]string{}, searchPath...)
}

func LoadSchemaWithCache(ctx context.Context, loader Loader, cache Cache, source string, searchPath []string) (Schema, error) {
	if loader == nil {
		return Schema{}, fmt.Errorf("schema loader is nil")
	}

	if snap, ok, err := cache.Load(); err == nil && ok {
		return snap.Schema, nil
	}

	schema, err := loader.Load(ctx)
	if err != nil {
		return Schema{}, err
	}

	snap := NewSnapshot(schema, source, ensureSearchPath(ctx, searchPath))
	_ = cache.Store(snap)
	return schema, nil
}
