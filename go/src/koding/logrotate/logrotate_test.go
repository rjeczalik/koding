package logrotate_test

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

type UserBucket map[string][]byte

func (ub UserBucket) Put(key string, rs io.ReadSeeker) error {
	var r io.Reader = rs

	if logrotate.IsGzip(key) {
		r = gzip.NewReader(rs)
	}

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	ub[key] = p

	return nil
}

func sum(s string) string {
	return hex.EncodeToString(sha1.Sum([]byte(s)))
}

var (
	n = logrotate.DefaultChecksumSize

	content = strings.Repeat("A", n/2) +
		strings.Repeat("B", n) +
		strings.Repeat("C", n)

	parts = []*logrotate.MetadataPart{{
		Offset:       0,
		Size:         n / 2,
		Checksum:     sum(content[:n/2]),
		ChecksumSize: n / 2,
	}, {
		Offset:       n / 2,
		Size:         3 * n / 2,
		Checksum:     sum(content[n/2 : 3*n/2]),
		ChecksumSize: n,
	}, {
		Offset:   3 * n / 2,
		Size:     5 * n / 2,
		Checksum: sum(content[3*n/2 : 5*n/2]),
	}}
)

func TestLogrotate_Rotate(t *testing.T) {
	// TODO
}
