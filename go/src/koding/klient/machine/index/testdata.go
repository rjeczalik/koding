// +build ignore

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var output = flag.String("o", filepath.FromSlash("testdata/koding.json.gz"), "output file")
var schrink = flag.Int64("s", 5, "schrink tree by the given multiplier")

func max(i, j int64) int64 {
	if i > j {
		return i
	}
	return j
}

func nonil(err ...error) error {
	for _, e := range err {
		if e != nil {
			return e
		}
	}
	return nil
}

func die(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func main() {
	flag.Parse()

	f, err := os.OpenFile(*output, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		die(err)
	}

	p, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		die(err)
	}

	r, w := io.Pipe()
	ch := make(chan error)
	m := make(map[string]int64)
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = string(bytes.TrimSpace(p))
	cmd.Stdout = w

	go func() {
		ch <- nonil(cmd.Run(), w.Close())
	}()

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		rel := scanner.Text()

		// git escapes certain paths, which makes then os.Stat to fail, like:
		//
		//   stat /Users/rjeczalik/src/github.com/koding/koding/"docs/assets/img/landing/team/pawe\305\202_knap.png": no such file or directory
		//
		// best-effort attempt at unescaping paths
		if s, err := strconv.Unquote(rel); err == nil {
			rel = s
		}

		fi, err := os.Stat(filepath.Join(cmd.Dir, rel))
		if err != nil {
			die(err)
		}

		m[rel] = fi.Size()
	}

	if err := nonil(<-ch, scanner.Err(), r.Close()); err != nil {
		die(err)
	}

	var sum int64
	for file, size := range m {
		if *schrink > 1 {
			size = max(size / *schrink, 1)
			m[file] = size
		}

		sum += size
	}

	gw := gzip.NewWriter(f)
	enc := json.NewEncoder(gw)

	if err := nonil(enc.Encode(m), gw.Close(), f.Close()); err != nil {
		die(err)
	}

	fmt.Printf("%q was written succesfully (total size %.4f MiB)\n", *output, float64(sum)/1024/1024)
}
