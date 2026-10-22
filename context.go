package statter

import "context"

type contextKey struct{}

type ctxTags struct {
	tags []Tag
}

// WithContext returns a context derived from ctx with tags attached. If ctx
// already carries tags added by WithContext, the new tags are merged and a
// value with the same key replaces the existing value. With no tags, it
// returns ctx unchanged.
func WithContext(ctx context.Context, tags ...Tag) context.Context {
	if len(tags) == 0 {
		return ctx
	}

	existing, _ := ctx.Value(contextKey{}).(*ctxTags)

	var merged []Tag
	if existing != nil && len(existing.tags) > 0 {
		merged = make([]Tag, len(existing.tags), len(existing.tags)+len(tags))
		copy(merged, existing.tags)
		merged = mergeTags(merged, tags)
	} else {
		merged = make([]Tag, len(tags))
		copy(merged, tags)
	}

	return context.WithValue(ctx, contextKey{}, &ctxTags{tags: merged})
}

// FromContext returns a sub-statter with the tags stored in ctx by
// WithContext. If ctx has no such tags, it returns s unchanged.
func (s *Statter) FromContext(ctx context.Context) *Statter {
	ct, _ := ctx.Value(contextKey{}).(*ctxTags)
	if ct == nil || len(ct.tags) == 0 {
		return s
	}
	return s.With("", ct.tags...)
}
