package storage

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

func JoinPath(elem ...string) string {
	var res []string
	for _, e := range elem {
		if e != "" {
			res = append(res, strings.Trim(e, "/"))
		}
	}
	return strings.Join(res, "/")
}
func AddDelimiterToPath(path string) string {
	if strings.HasSuffix(path, "/") || path == "" {
		return path
	}
	return path + "/"
}

func GetPathFromPrefix(prefix string) (bucket, server string, err error) {
	bucket, server, err = ParsePrefixAsURL(prefix)
	if err != nil {
		return "", "", err
	}

	// Allover the code this parameter is concatenated with '/'.
	// TODO: Get rid of numerous string literals concatenated with this
	server = strings.Trim(server, "/")

	return bucket, server, nil
}

func ParsePrefixAsURL(prefix string) (bucket, server string, err error) {
	storageURL, err := url.Parse(prefix)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to parse url '%s'", prefix)
	}
	if storageURL.Scheme == "" || storageURL.Host == "" {
		return "", "", errors.Errorf("missing url scheme=%q and/or host=%q", storageURL.Scheme, storageURL.Host)
	}

	return storageURL.Host, storageURL.Path, nil
}
