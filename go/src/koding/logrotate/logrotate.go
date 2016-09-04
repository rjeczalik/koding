// Package logrotate is used to stream logs content.
//
// It caches stream metadata per key, so each time e.g. the same
// log file is streamed, it streams only the part that was not
// already streamed (rotates a log's content in-place).
//
// It caches also SHA1 checksum of the last n bytes of every
// streamed part, so it handles situation, where underlying
// log file was replaced and must not be rotated.
package logrotate

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"koding/klient/storage"

	"github.com/koding/logging"
)

var defaultLog = logging.NewCustom("logrotate", false)

// DefaultChecksumSize is a size of a part that is going to be checksummed.
//
// E.g. if your log file contains a lot of lines that have exact the same
// content and additionally those lines are long ones (>128 characters),
// you may want to increase the DefaultChecksumSize to ensure
// there are not false-positive checksum matches.
//
// TODO(rjeczalik): make Logger use configurable ChecksumSize
const DefaultChecksumSize = 128

// Metadata
type Metadata struct {
	Key   string          `json:"key"`
	Parts []*MetadataPart `json:"parts"`
}

// LastPart returns last element from Parts slice.
//
// If Parts is empty, it returns nil.
func (meta *Metadata) LastPart() *MetadataPart {
	if n := len(meta.Parts); n != 0 {
		return meta.Parts[n-1]
	}

	return nil
}

// MetadataPart
type MetadataPart struct {
	CreatedAt      time.Time `json:"createdAt"`
	Offset         int64     `json:"offset"`
	Size           int64     `json:"size"`
	CompressedSize int64     `json:"compressedSize"`
	Checksum       string    `json:"checksum"`
	ChecksumSize   int64     `json:"checksumSize"`
}

// NopError is returned by rotate, if there's no content
// after rotating.
//
// It is used by Upload to not stream empty content.
type NopError struct {
	Key string //content's key
	N   int    // part number
}

// Error implements the error interface.
func (ne *NopError) Error() string {
	return fmt.Sprintf(`no content left after rotating "%s.%d"`, ne.Key, ne.N)
}

// Putter is responsible for streaming the content under the given key path.
type Putter interface {
	Put(key string, content io.ReadSeeker) error
}

// Logger is used to stream logs' contents.
//
// It is also compresses log content.
type Logger struct {
	// UserBucket is used to stream log contents.
	// Required.
	UserBucket Putter

	// MetaStore is used for caching metadata.
	// Required.
	MetaStore storage.ValueInterface

	// Log to use for logging.
	// If nil, defaultLog is used.
	Log logging.Logger
}

// UploadFile uploads the content's of the given file.
//
// The key is constructed by joining prefix and the
// canonical form of the file path.
func (l *Logger) UploadFile(prefix, file string) error {
	key := filepath.ToSlash(filepath.Clean(file))
	if prefix != "" {
		key = prefix + "/" + key
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}

	return l.Upload(key, f)
}

// Upload streams the given content under the given key.
//
// If content implements io.Closer it will get called upon method return.
//
// If the file name of the key ends with .gz extension, the content is going to
// be additionally compressed with gzip.
//
// TODO(rjeczalik): detect if content is already gzipped and do not
// double-compress it
func (l *Logger) Upload(key string, content io.ReadSeeker) error {
	if c, ok := content.(io.Closer); ok {
		defer c.Close()
	}

	meta := l.meta(key)
	part, err := rotate(content, meta)
	switch err.(type) {
	case nil:
	case *NopError:
		return nil
	default:
		return err
	}

	uniqueKey := fmt.Sprintf("%s.%d", key, len(meta.Parts))

	if isGzip(key) {
		c, err := l.gzip(uniqueKey, content, &part.CompressedSize)
		if err != nil {
			return err
		}

		content = c
	}

	if err = l.UserBucket.Put(uniqueKey, content); err != nil {
		return err
	}

	meta.Parts = append(meta.Parts, part)

	if err := l.MetaStore.SetValue(key, meta); err != nil {
		l.log().Error("failure updating metadata for %q: %s", key, err)
	}

	return nil
}

func (l *Logger) meta(key string) *Metadata {
	var meta Metadata

	err := l.MetaStore.GetValue(key, &meta)
	if err == nil {
		return &meta
	}

	if err != storage.ErrKeyNotFound {
		l.log().Warning("failure reading metadata for %q: %s", key, err)
	}

	return &Metadata{Key: key}
}

func (l *Logger) gzip(key string, rs io.ReadSeeker, n *int64) (io.ReadSeeker, error) {
	var buf bytes.Buffer

	cw := gzip.NewWriter(&buf)

	_, err := io.Copy(&CountingWriter{W: cw, N: n}, rs)
	if err = nonil(err, cw.Close()); err != nil {
		if _, e := rs.Seek(0, io.SeekStart); e != nil {
			return nil, err
		}

		l.log().Error("failure compressing content for %q: %s", key, err)

		return rs, nil
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func (l *Logger) log() logging.Logger {
	if l.Log != nil {
		return l.Log
	}

	return defaultLog
}

func rotate(content io.ReadSeeker, meta *Metadata) (part *MetadataPart, err error) {
	part = &MetadataPart{
		CreatedAt: time.Now(),
	}

	part.Size, err = content.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	part.ChecksumSize = min(part.Size, DefaultChecksumSize)

	if last := meta.LastPart(); last != nil && last.Size <= part.Size {
		if last.ChecksumSize == 0 {
			last.ChecksumSize = min(last.Size, DefaultChecksumSize)
		}

		chk, err := readChecksum(content, last.Size-last.ChecksumSize, last.ChecksumSize)
		if err != nil {
			return nil, err
		}

		// If chk matches last.Checksum it means the content
		// did not change and we can rotate it.
		if chk == last.Checksum {
			part.Offset = last.Size
		}

		if part.Offset == part.Size {
			return nil, &NopError{
				Key: meta.Key,
				N:   len(meta.Parts) - 1,
			}
		}
	}

	part.Checksum, err = readChecksum(content, part.Size-part.ChecksumSize, part.ChecksumSize)
	if err != nil {
		return nil, err
	}

	_, err = content.Seek(part.Offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return part, nil
}

func readChecksum(rs io.ReadSeeker, offset, size int64) (string, error) {
	_, err := rs.Seek(offset, io.SeekStart)
	if err != nil {
		return "", err
	}

	h := sha1.New()

	_, err = io.Copy(h, &io.LimitedReader{R: rs, N: size})
	if err != nil {
		return "", err
	}

	p := h.Sum(nil)
	return hex.EncodeToString(p[:]), nil
}

func isGzip(key string) bool {
	return strings.HasSuffix(strings.ToLower(path.Base(key)), ".gz")
}
