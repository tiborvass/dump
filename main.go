package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Dump struct{}

const alpineRepository = "ttl.sh/tiborvass-alpine"

func (m *Dump) Version() string {
	return "v26.0.0"
}

func (m *Dump) AlpineVersion(ctx context.Context) (string, error) {
	version, err := dag.Container().
		From(alpineRepository).
		WithExec([]string{
			"sh",
			"-c",
			`. /etc/os-release; printf '%s' "$VERSION_ID"`,
		}).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(version), nil
}

func (m *Dump) Reset(ctx context.Context) (string, error) {
	const manifestURL = "https://ttl.sh/v2/tiborvass-alpine/manifests/"
	tags := []string{"3.20.0", "3.21.0"}
	deleted := []string{}

	for _, tag := range tags {
		digest, found, err := manifestDigest(ctx, manifestURL+tag)
		if err != nil {
			return "", err
		}
		if !found {
			continue
		}

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodDelete,
			manifestURL+digest,
			nil,
		)
		if err != nil {
			return "", fmt.Errorf("create delete request for %s: %w", tag, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("delete %s: %w", tag, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK &&
			resp.StatusCode != http.StatusAccepted &&
			resp.StatusCode != http.StatusNotFound {
			return "", fmt.Errorf("delete %s: %s", tag, resp.Status)
		}
		if resp.StatusCode != http.StatusNotFound {
			deleted = append(deleted, tag)
		}
	}

	if len(deleted) == 0 {
		return "container tags already absent", nil
	}
	return "deleted container tags: " + strings.Join(deleted, ", "), nil
}

func manifestDigest(ctx context.Context, manifestURL string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", false, fmt.Errorf("create manifest request: %w", err)
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("resolve manifest: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("resolve manifest: %s", resp.Status)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", false, fmt.Errorf("resolve manifest: digest header is missing")
	}
	return digest, true, nil
}
