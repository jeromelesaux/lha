lha golang pure implementation WIP
to compile standalone application ```go build -o lha cmd/lharc.go```

Actual status : 
- list archive content ok
- compress data ok 
- decompress data ok 
what ever header level and data compression method. 

to it as lib : 
- to get archive headers : 
``` 
v := NewLha("archive_filepath")
headers, err := v.Headers()
```

- to decompress a existing archive : 
```
	v := NewLha("existing_archive_filepath")
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
```
- to compress a new archive : 
```
v2 := NewLha("new_archive_filepath")
	err := v2.Compress("filepath_to_include", Lzhuff5MethodNum, 2)
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
```

- to compress with a slice of bytes : 
```
v2 := NewLha("new_archive_filepath")

	f, err := os.Open("file_to_include")
	if err != nil {
		t.Fatalf("cannot open file :%v\n", err.Error())
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("cannot read file :%v\n", err.Error())
	}
	err = v2.CompressBytes("filename.bin", data, Lzhuff5MethodNum, 2)
	if err != nil {
		t.Fatalf("error not expected :%v\n", err.Error())
	}
```

- to list headers content : 
```
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
```