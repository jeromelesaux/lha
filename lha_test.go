package lha

import (
	"io"
	"os"
	"testing"
)

const (
	oldArchivename = "Deep_Space.ym"
	newArchivename = "test.ym"
)

func TestListHeader(t *testing.T) {
	v := NewLha(oldArchivename)
	headers, err := v.Headers()
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	if len(headers) != 1 {
		t.Fatalf("expected 1 header and gets %d headers\n", len(headers))
	}

	for i, vv := range headers {
		t.Logf("Headers[%d] filename : %s\n", i, string(vv.Realname))
	}
}

func TestDecompress(t *testing.T) {
	v := NewLha(oldArchivename)
	headers, err := v.Headers()
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	if len(headers) != 1 {
		t.Fatalf("expected 1 header and gets %d headers\n", len(headers))
	}
	err = v.Decompress(headers[0])
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
}

func TestDecompressBytes(t *testing.T) {
	v := NewLha(oldArchivename)
	headers, err := v.Headers()
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	if len(headers) != 1 {
		t.Fatalf("expected 1 header and gets %d headers\n", len(headers))
	}
	d, err := v.DecompresBytes(headers[0])
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	if len(d) == 0 {
		t.Fatalf("error not expected an empty content\n")
	}
}

func TestCompress(t *testing.T) {
	os.Remove(newArchivename)
	v2 := NewLha(newArchivename)
	err := v2.Compress("Deep Space Main Part (1995)(GPA)(Resonance - Targhan)().ym", Lzhuff5MethodNum, 2)
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	_, err = os.Lstat(newArchivename)
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	os.Remove("test.ym")
}

func TestCompressBytes(t *testing.T) {
	os.Remove(newArchivename)
	v2 := NewLha(newArchivename)

	f, err := os.Open("Deep Space Main Part (1995)(GPA)(Resonance - Targhan)().ym")
	if err != nil {
		t.Fatalf("cannot open file :%v\n", err.Error())
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("cannot read file :%v\n", err.Error())
	}
	err = v2.CompressBytes("Deep Space Main Part (1995)(GPA)(Resonance - Targhan)().ym", data, Lzhuff5MethodNum, 2)
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	os.Remove("test.ym")
}

func TestInformationHeader(t *testing.T) {
	v := NewLha(oldArchivename)
	headers, err := v.Headers()
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
	if len(headers) != 1 {
		t.Fatalf("expected 1 header and gets %d headers\n", len(headers))
	}

	for i, vv := range headers {
		t.Logf("Headers[%d] info :%s\n", i, vv.String())
	}
}
