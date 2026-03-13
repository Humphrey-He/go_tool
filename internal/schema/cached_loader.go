package schema

import "context"

type CachedLoader struct {
	Loader     Loader
	Cache      Cache
	Source     string
	SearchPath []string
}

func (l CachedLoader) Load(ctx context.Context) (Schema, error) {
	return LoadSchemaWithCache(ctx, l.Loader, l.Cache, l.Source, l.SearchPath)
}
