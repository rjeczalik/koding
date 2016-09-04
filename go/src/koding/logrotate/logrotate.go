package logrotate

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"koding/klient/storage"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/koding/logging"
)

var defaultLog = logging.NewCustom("logrotate", false)

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

// Putter
type Putter interface {
	Put(key string, content io.ReadSeeker) error
}

// Logger
type Logger struct {
	// UserBucket
	UserBucket Putter

	// MetaStore
	MetaStore storage.ValueInterface

	// Log
	Log logging.Logger
}

// UploadFile
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

// Upload
//
// If the file name of the key ends with .gz extension, the content is going to
// be additionally compressed with gzip.
//
// If content implements io.Closer it will get called upon method return.
func (l *Logger) Upload(key string, content io.ReadSeeker) error {
	if c, ok := content.(io.Closer); ok {
		defer c.Close()
	}

	meta := l.meta(key)
	part, err := rotate(content, meta)
	if err != nil {
		return err
	}

	uniqueKey := fmt.Sprintf("%s.%d", key, len(meta.Parts))
	content = &CountingReader{RS: content, N: &part.Size}

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

func rotate(content io.ReadSeeker, meta *Metadata) (*MetadataPart, error) {
	part := &MetadataPart{
		CreatedAt: time.Now(),
	}

	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	part.ChecksumSize = min(size, DefaultChecksumSize)

	if last := meta.LastPart(); last != nil && last.Size < size {
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
	}

	part.Checksum, err = readChecksum(content, size-part.ChecksumSize, part.ChecksumSize)
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
