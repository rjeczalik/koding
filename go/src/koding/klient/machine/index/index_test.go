package index

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// filetree defines a simple directory structure that will be created for test
// purposes. The values of this map stores file sizes.
var filetree = map[string]int64{
	"a.txt":        128,
	"b.bin":        300 * 1024,
	"c/":           0,
	"c/ca.txt":     2 * 1024,
	"c/cb.bin":     1024 * 1024,
	"d/":           0,
	"d/da.txt":     5 * 1024,
	"d/db.txt":     256,
	"d/dc/":        0,
	"d/dc/dca.txt": 3 * 1024,
	"d/dc/dcb.txt": 1024,
}

func TestIndex(t *testing.T) {
	tests := map[string]struct {
		Op      func(string) error
		Changes []Change
	}{
		"add file": {
			Op: writeFile("d/test.bin", 40*1024),
			Changes: []Change{
				{
					Name: "d",
					Meta: ChangeMetaUpdate,
				},
				{
					Name: "d/test.bin",
					Size: 40 * 1024,
					Meta: ChangeMetaAdd,
				},
			},
		},
		"add dir": {
			Op: addDir("e"),
			Changes: []Change{
				{
					Name: "e",
					Meta: ChangeMetaAdd,
				},
			},
		},
		"remove file": {
			Op: rmAllFile("c/cb.bin"),
			Changes: []Change{
				{
					Name: "c",
					Meta: ChangeMetaUpdate,
				},
				{
					Name: "c/cb.bin",
					Meta: ChangeMetaRemove,
				},
			},
		},
		"remove dir": {
			Op: rmAllFile("c"),
			Changes: []Change{
				{
					Name: "c",
					Meta: ChangeMetaRemove,
				},
				{
					Name: "c/ca.txt",
					Meta: ChangeMetaRemove,
				},
				{
					Name: "c/cb.bin",
					Meta: ChangeMetaRemove,
				},
			},
		},
		"rename file": {
			Op: mvFile("b.bin", "c/cc.bin"),
			Changes: []Change{
				{
					Name: "b.bin",
					Meta: ChangeMetaRemove,
				},
				{
					Name: "c",
					Meta: ChangeMetaUpdate,
				},
				{
					Name: "c/cc.bin",
					Meta: ChangeMetaAdd,
				},
			},
		},
		"write file": {
			Op: writeFile("b.bin", 1024),
			Changes: []Change{
				{
					Name: "b.bin",
					Meta: ChangeMetaUpdate,
				},
			},
		},
		"chmod file": {
			Op: chmodFile("d/dc/dca.txt", 0766),
			Changes: []Change{
				{
					Name: "d/dc/dca.txt",
					Meta: ChangeMetaUpdate,
				},
			},
		},
	}

	for name, test := range tests {
		// capture range variable here
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root, clean, err := generateTree()
			if err != nil {
				t.Fatalf("want err = nil; got %v", err)
			}
			defer clean()

			idx, err := NewIndexFiles(root)
			if err != nil {
				t.Fatalf("want err = nil; got %v", err)
			}

			if err := test.Op(root); err != nil {
				t.Fatalf("want err = nil; got %v", err)
			}

			// Synchronize underlying file-system.
			Sync()

			cs := idx.Compare(root)
			sort.Sort(cs)
			if len(cs) != len(test.Changes) {
				t.Fatalf("want changes count = %d; got %d", len(test.Changes), len(cs))
			}

			// Copy time from result to tests.
			for i, tc := range test.Changes {
				if cs[i].Name != tc.Name {
					t.Errorf("want change name = %q; got %q", tc.Name, cs[i].Name)
				}
				if tc.Size != 0 && cs[i].Size != tc.Size {
					t.Errorf("want change size = %v; got %v", tc.Size, cs[i].Size)
				}
				if cs[i].Meta != tc.Meta {
					t.Errorf("want change meta = %bb; got %bb", tc.Meta, cs[i].Meta)
				}
			}

			idx.Apply(root, cs)
			if cs = idx.Compare(root); len(cs) != 0 {
				t.Errorf("want no changes after apply; got %#v", cs)
			}
		})
	}
}

func TestIndexCount(t *testing.T) {
	tests := map[string]struct {
		MaxSize  int64
		Expected int
	}{
		"all items": {
			MaxSize:  -1,
			Expected: 11,
		},
		"less than 100kiB": {
			MaxSize:  100 * 1024,
			Expected: 9,
		},
		"zero": {
			MaxSize:  0,
			Expected: 0,
		},
	}

	for name, test := range tests {
		// capture range variable here
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root, clean, err := generateTree()
			if err != nil {
				t.Fatalf("want err = nil; got %v", err)
			}
			defer clean()

			idx, err := NewIndexFiles(root)
			if err != nil {
				t.Fatalf("want err = nil; got %v", err)
			}

			if count := idx.Count(test.MaxSize); count != test.Expected {
				t.Errorf("want count = %d; got %d", test.Expected, count)
			}
		})
	}
}

func TestIndexJSON(t *testing.T) {
	root, clean, err := generateTree()
	if err != nil {
		t.Fatalf("want err = nil; got %v", err)
	}
	defer clean()

	idx, err := NewIndexFiles(root)
	if err != nil {
		t.Fatalf("want err = nil; got %v", err)
	}

	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("want err = nil; got %v", err)
	}

	idx = NewIndex()
	if err := json.Unmarshal(data, idx); err != nil {
		t.Fatalf("want err = nil; got %v", err)
	}

	if cs := idx.Compare(root); len(cs) != 0 {
		t.Errorf("want no changes after apply; got %#v", cs)
	}
}

func generateTree() (root string, clean func(), err error) {
	return generate(filetree)
}

func generate(tree map[string]int64) (root string, clean func(), err error) {
	type item struct {
		file string
		size int64
	}

	root, err = ioutil.TempDir("", "mount.index")
	if err != nil {
		return "", nil, err
	}
	clean = func() { os.RemoveAll(root) }

	items := make(chan item)

	go func() {
		for file, size := range tree {
			items <- item{
				file: file,
				size: size,
			}
		}

		close(items)
	}()

	var wg sync.WaitGroup
	n := 2 * runtime.NumCPU()
	ch := make(chan error, n)
	done := make(chan struct{})
	counter := int32(0)

	go func() {
		wg.Wait()
		close(done)
	}()

	fmt.Fprintf(os.Stderr, "Generating tree at %q... (%d files, %d goroutines)\n", root, len(tree), n)

	for ; n > 0; n-- {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range items {
				if err := addDir(item.file)(root); err != nil {
					ch <- err
					return
				}
				if err := writeFile(item.file, item.size)(root); err != nil {
					ch <- err
					return
				}

				if n := atomic.AddInt32(&counter, 1); n%1000 == 0 && n != 0 {
					fmt.Printf("Writing files %d/%d...\n", n, len(tree))
				}
			}
		}()
	}

	select {
	case <-done:
	case <-ch:
		clean()
		return "", nil, err
	}

	return root, clean, nil
}

func addDir(file string) func(string) error {
	return func(root string) error {
		defer Sync()

		dir := filepath.Join(root, filepath.FromSlash(file))
		if filepath.Ext(dir) != "" {
			dir = filepath.Dir(dir)
		}

		return os.MkdirAll(dir, 0777)
	}
}

func writeFile(file string, size int64) func(string) error {
	return func(root string) error {
		defer Sync()

		if filepath.Ext(file) == "" {
			return nil
		}

		lr := io.LimitReader(rand.New(rand.NewSource(time.Now().UnixNano())), size)
		content, err := ioutil.ReadAll(lr)
		if err != nil {
			return err
		}

		file := filepath.Join(root, filepath.FromSlash(file))
		return ioutil.WriteFile(file, content, 0666)
	}
}

func rmAllFile(file string) func(string) error {
	return func(root string) error {
		defer Sync()

		return os.RemoveAll(filepath.Join(root, filepath.FromSlash(file)))
	}
}

func mvFile(oldpath, newpath string) func(string) error {
	return func(root string) error {
		defer Sync()

		var (
			oldpath = filepath.Join(root, filepath.FromSlash(oldpath))
			newpath = filepath.Join(root, filepath.FromSlash(newpath))
		)

		return os.Rename(oldpath, newpath)
	}
}

func chmodFile(file string, mode os.FileMode) func(string) error {
	return func(root string) error {
		defer Sync()

		return os.Chmod(filepath.Join(root, filepath.FromSlash(file)), mode)
	}
}
