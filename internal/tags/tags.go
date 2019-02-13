package tags

func Normalize(tags []interface{}) []interface{} {
	// tags need to be even as they are key/value pairs
	if len(tags)%2 != 0 {
		tags = append(tags, nil, "STATTER_ERROR", "Normalised odd number of tags by adding nil")
	}

	return tags
}

func Deduplicate(tags []interface{}) []interface{} {
	for i := 0; i < len(tags); i += 2 {
		for j := i + 2; j < len(tags); j += 2 {
			if tags[i] == tags[j] {
				tags[i+1] = tags[j+1]
				tags = append(tags[:j], tags[j+2:]...)
				j -= 2
			}
		}
	}

	return tags
}
