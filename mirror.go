package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Mirror struct {
	registry string
	basic    authn.Basic
}

// Ensure ensures an image is in the mirror registry
func (m Mirror) Ensure(img string) (dstImg string, err error) {
	srcRef, err := name.ParseReference(img)
	if err != nil {
		return "", fmt.Errorf("mirror parse src=%s: %w", img, err)
	}

	if strings.HasPrefix(srcRef.Name(), m.registry) {
		// no mirror needed
		return img, nil
	}

	return m.mirror(srcRef)
}

// src is the image to mirror,
// reg is the registry to mirror to,
// dst = dst.reg/src.reg/name:tag
func (m Mirror) mirror(srcRef name.Reference) (dst string, err error) {
	srcImg, err := remote.Image(srcRef) // TODO: auth
	if err != nil {
		return "", fmt.Errorf("mirror remote src=%s: %w", srcRef.String(), err)
	}

	// replace : for ports with _
	replacer := strings.NewReplacer(":", "_", "/", "_", ".", "_")
	dst = path.Join(m.registry, replacer.Replace(srcRef.Context().Name())+":"+srcRef.Identifier())
	dstRef, err := name.ParseReference(dst)
	if err != nil {
		return "", fmt.Errorf("mirror parse dst=%s: %w", dst, err)
	}

	err = remote.Write(dstRef, srcImg, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", fmt.Errorf("mirror write src=%s dst=%s: %w", srcRef.Name(), dst, err)
	}
	return dst, nil
}
