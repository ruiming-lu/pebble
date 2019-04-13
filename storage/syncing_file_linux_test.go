// Copyright 2019 The LevelDB-Go and Pebble Authors. All rights reserved. Use
// of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// +build linux

package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func BenchmarkDirectIOWrite(b *testing.B) {
	const targetSize = 16 << 20
	const alignment = 4096

	var wsizes []int
	if testing.Verbose() {
		wsizes = []int{4 << 10, 8 << 10, 16 << 10, 32 << 10}
	} else {
		wsizes = []int{4096}
	}

	for _, wsize := range wsizes {
		b.Run(fmt.Sprintf("wsize=%d", wsize), func(b *testing.B) {
			tmpf, err := ioutil.TempFile("", "pebble-db-syncing-file-")
			if err != nil {
				b.Fatal(err)
			}
			filename := tmpf.Name()
			_ = tmpf.Close()
			defer os.Remove(filename)

			var f *os.File
			var size int
			buf := make([]byte, wsize+alignment)
			if a := uintptr(unsafe.Pointer(&buf[0])) & uintptr(alignment-1); a != 0 {
				buf = buf[alignment-a:]
			}
			buf = buf[:wsize]
			init := true

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if f == nil {
					b.StopTimer()
					f, err = os.OpenFile(filename, syscall.O_DIRECT|os.O_RDWR, 0666)
					if err != nil {
						b.Fatal(err)
					}
					if init {
						for size = 0; size < targetSize; size += len(buf) {
							if _, err := f.WriteAt(buf, int64(size)); err != nil {
								b.Fatal(err)
							}
						}
					}
					if err := f.Sync(); err != nil {
						b.Fatal(err)
					}
					size = 0
					b.StartTimer()
				}
				if _, err := f.WriteAt(buf, int64(size)); err != nil {
					b.Fatal(err)
				}
				size += len(buf)
				if size >= targetSize {
					_ = f.Close()
					f = nil
				}
			}
			b.StopTimer()
		})
	}
}
