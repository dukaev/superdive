package dive

import (
	"fmt"
	"strings"

	"github.com/wagoodman/dive/dive/image"
	"github.com/wagoodman/dive/dive/image/apple"
	"github.com/wagoodman/dive/dive/image/docker"
	"github.com/wagoodman/dive/dive/image/podman"
)

const (
	SourceUnknown ImageSource = iota
	SourceDockerEngine
	SourcePodmanEngine
	SourceDockerArchive
	SourceAppleContainer
)

type ImageSource int

var ImageSources = []string{SourceDockerEngine.String(), SourcePodmanEngine.String(), SourceDockerArchive.String(), SourceAppleContainer.String()}

func (r ImageSource) String() string {
	return [...]string{"unknown", "docker", "podman", "docker-archive", "apple"}[r]
}

func ParseImageSource(r string) ImageSource {
	switch r {
	case SourceDockerEngine.String():
		return SourceDockerEngine
	case SourcePodmanEngine.String():
		return SourcePodmanEngine
	case SourceDockerArchive.String():
		return SourceDockerArchive
	case "docker-tar":
		return SourceDockerArchive
	case SourceAppleContainer.String():
		return SourceAppleContainer
	default:
		return SourceUnknown
	}
}

func DeriveImageSource(image string) (ImageSource, string) {
	s := strings.SplitN(image, "://", 2)
	if len(s) < 2 {
		return SourceUnknown, ""
	}
	scheme, imageSource := s[0], s[1]

	switch scheme {
	case SourceDockerEngine.String():
		return SourceDockerEngine, imageSource
	case SourcePodmanEngine.String():
		return SourcePodmanEngine, imageSource
	case SourceDockerArchive.String():
		return SourceDockerArchive, imageSource
	case "docker-tar":
		return SourceDockerArchive, imageSource
	case SourceAppleContainer.String():
		return SourceAppleContainer, imageSource
	}
	return SourceUnknown, ""
}

func GetImageResolver(r ImageSource) (image.Resolver, error) {
	switch r {
	case SourceDockerEngine:
		return docker.NewResolverFromEngine(), nil
	case SourcePodmanEngine:
		return podman.NewResolverFromEngine(), nil
	case SourceDockerArchive:
		return docker.NewResolverFromArchive(), nil
	case SourceAppleContainer:
		return apple.NewResolverFromEngine(), nil
	}

	return nil, fmt.Errorf("unable to determine image resolver")
}
