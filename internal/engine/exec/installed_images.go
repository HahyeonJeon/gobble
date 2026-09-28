package exec

import "context"

type installedImagesKey struct{}

// WithInstalledImages pins the accepted local image set for this execution only.
// Missing images fail; neither a registry pull nor a different local image is allowed.
func WithInstalledImages(ctx context.Context, images map[string]string) context.Context {
	copy := make(map[string]string, len(images))
	for k, v := range images {
		copy[k] = v
	}
	return context.WithValue(ctx, installedImagesKey{}, copy)
}
