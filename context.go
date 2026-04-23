package statter

import "context"

type contextKey struct{}

type ctxTags struct {
	tags []Tag
}

// WithContext returns a new context with the given tags attached. If tags are
// already present in ctx, the new tags are merged with them; a new tag whose
// key already exists overrides the stored value.
func WithContext(ctx context.Context, tags ...Tag) context.Context {
	if len(tags) == 0 {
		return ctx
	}

	existing, _ := ctx.Value(contextKey{}).(*ctxTags)

	var merged []Tag
	if existing != nil && len(existing.tags) > 0 {
		merged = make([]Tag, len(existing.tags), len(existing.tags)+len(tags))
		copy(merged, existing.tags)
		for _, tag := range tags {
			if i := tagIndex(merged, tag[0]); i >= 0 {
				merged[i][1] = tag[1]
			} else {
				merged = append(merged, tag)
			}
		}
	} else {
		merged = make([]Tag, len(tags))
		copy(merged, tags)
	}

	return context.WithValue(ctx, contextKey{}, &ctxTags{tags: merged})
}

// FromContext returns a sub-statter whose tags include those stored in ctx by
// WithContext. If no tags are found in ctx, s is returned unchanged.
func (s *Statter) FromContext(ctx context.Context) *Statter {
	ct, _ := ctx.Value(contextKey{}).(*ctxTags)
	if ct == nil || len(ct.tags) == 0 {
		return s
	}
	return s.With("", ct.tags...)
}
