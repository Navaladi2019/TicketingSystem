package pipeline

// mergePatch applies patch to target as RFC 7396 JSON Merge Patch and returns
// the result: a null member removes the key, an object member merges into an
// object (or replaces a non-object), and every other value, arrays included,
// replaces the target's value whole. target is changed in place; patch is not,
// and no part of patch is shared with the result. Both hold decoded JSON.
func mergePatch(target, patch map[string]any) map[string]any {
	if target == nil {
		target = map[string]any{}
	}
	for k, v := range patch {
		switch pv := v.(type) {
		case nil:
			delete(target, k)
		case map[string]any:
			sub, ok := target[k].(map[string]any)
			if !ok {
				sub = map[string]any{}
			}
			target[k] = mergePatch(sub, pv)
		default:
			target[k] = copyValue(v)
		}
	}
	return target
}

// copyValue deep-copies a decoded JSON value.
func copyValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = copyValue(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = copyValue(e)
		}
		return out
	default:
		return v
	}
}
