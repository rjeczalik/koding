package node

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/djherbis/times"
)

// EntryPromise describes the promised state of index entry.
type EntryPromise uint32

const (
	EntryPromiseVirtual EntryPromise = 1 << iota // V: promise that file exists, doesn't exist locally.
	EntryPromiseAdd                              // A: promise after adding, exists locally.
	EntryPromiseUpdate                           // U: promise after updating, exists locally.
	EntryPromiseDel                              // D: promise after deleting, doesn't exist locally.
	EntryPromiseUnlink                           // N: promise after hard delete, doesn't exist locally.
)

var epMapping = map[byte]EntryPromise{
	'V': EntryPromiseVirtual,
	'A': EntryPromiseAdd,
	'U': EntryPromiseUpdate,
	'D': EntryPromiseDel,
	'N': EntryPromiseUnlink,
}

// String implements fmt.Stringer interface and pretty prints stored promise.
func (ep EntryPromise) String() string {
	var buf [8]byte // Promise is uint32 but we don't need more than 8.
	for c, i := range epMapping {
		w := getPowerOf2(uint64(i))
		if ep&i != 0 {
			buf[w] = c
		} else {
			buf[w] = '-'
		}
	}

	return string(buf[:len(epMapping)])
}

// Deleted checks if the promise is set to be deleted.
func (ep EntryPromise) Deleted() bool {
	const del = EntryPromiseDel | EntryPromiseUnlink

	return ep&del != 0
}

// Virtual checks if the promise is set to be virtual.
func (ep EntryPromise) Virtual() bool {
	return ep&EntryPromiseVirtual != 0
}

// Exist checks if the promise indicates that the entry exists locally.
func (ep EntryPromise) Exist() bool {
	return !ep.Deleted() && !ep.Virtual()
}

// Swap flips the value of a promise field, setting the set bits and
// unsetting the unset ones.
func (ep *EntryPromise) Swap(set, unset EntryPromise) {
	*ep = (*ep | set) &^ unset
}

// File describes the part of entry which is independent of underlying file
// system. The fields of this type can be stored on disk and transferred over
// the network.
type File struct {
	CTime int64       `json:"c"` // Metadata change time since EPOCH.
	MTime int64       `json:"m"` // File data change time since EPOCH.
	Size  int64       `json:"s"` // Size of the file.
	Mode  os.FileMode `json:"o"` // File mode and permission bits.
}

// Virtual stores virtual file system dependent data that is lost during
// serialization and should be recreated by VFS which manages the entries.
type Virtual struct {
	Inode    uint64       // Inode ID of a mounted file.
	RefCount int32        // Reference count of file handlers.
	Promise  EntryPromise // Metadata of files's memory state.
}

var (
	_ json.Marshaler   = (*Entry)(nil)
	_ json.Unmarshaler = (*Entry)(nil)
)

// Entry represents a single file registered to index.
type Entry struct {
	File    File
	Virtual Virtual
}

// NewEntry creates a new entry that describes the wile with specified size and
// mode. VFS are zero values and must be set manually.
func NewEntry(size int64, mode os.FileMode) *Entry {
	t := time.Now().UTC().UnixNano()
	return NewEntryTime(t, t, size, mode)
}

// NewEntryFileInfo creates a new entry from a given file info.
func NewEntryFileInfo(info os.FileInfo) *Entry {
	return NewEntryTime(
		ctime(info),
		info.ModTime().UTC().UnixNano(),
		info.Size(),
		info.Mode(),
	)
}

// NewEntryTime creates a new entry with custom file change and modify times.
func NewEntryTime(ctime, mtime, size int64, mode os.FileMode) *Entry {
	return &Entry{
		File: File{
			CTime: ctime,
			MTime: mtime,
			Size:  size,
			Mode:  mode,
		},
	}
}

// NewEntryFile creates a new entry which describes the given file.
func NewEntryFile(path string) (*Entry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	return NewEntryFileInfo(info), nil
}

// Clone returns a deep copy of the e value.
//
// RefCount field is ignored and set to 0.
func (e *Entry) Clone() *Entry {
	return &Entry{
		File: e.File,
		Virtual: Virtual{
			Inode:    e.Virtual.Inode,
			RefCount: 0,
			Promise:  e.Virtual.Promise,
		},
	}
}

// MergeIn overwrites e's fields with f's ones, but only
// with those values the are non-zero.
//
// RefCount and Promise fields are ignored.
func (e *Entry) MergeIn(f *Entry) {
	if t := f.File.CTime; t != 0 {
		e.File.CTime = t
	}
	if t := f.File.MTime; t != 0 {
		e.File.MTime = t
	}
	if n := f.File.Size; n != 0 {
		e.File.Size = n
	}
	if m := f.File.Mode; m != 0 {
		e.File.Mode = m
	}

	if n := f.Virtual.Inode; n != 0 {
		e.Virtual.Inode = n
	}
}

// String implements fmt.Stringer interface. It pretty prints the entry.
func (e *Entry) String() string {
	var (
		ctime = time.Unix(0, e.File.CTime)
		mtime = time.Unix(0, e.File.MTime)
	)

	return fmt.Sprintf("[INODE %d, REFS %d, PROMISE %s][CTIME %s, MTIME %s, SIZE %d, MODE %s]",
		e.Virtual.Inode, e.Virtual.RefCount, e.Virtual.Promise,
		ctime.Format(time.StampMilli), mtime.Format(time.StampMilli), e.File.Size, e.File.Mode,
	)
}

// MarshalJSON satisfies json.Marshaler interface. It marshals only the entry
// file data to JSON format.
func (e *Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.File)
}

// UnmarshalJSON satisfies json.Unmarshaler interface.
func (e *Entry) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &e.File); err != nil {
		return err
	}

	return nil
}

func getPowerOf2(i uint64) (count int) {
	for ; i > 1; count++ {
		i = i >> 1
	}

	return count
}

// ctime gets file's change time in UNIX Nano format.
func ctime(fi os.FileInfo) int64 {
	if tspec := times.Get(fi); tspec.HasChangeTime() {
		return tspec.ChangeTime().UnixNano()
	}

	return 0
}
