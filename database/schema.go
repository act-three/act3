package database

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed ddl/*_*.up.sql
var ddl embed.FS

//go:embed frozen.txt
var frozen []byte

type update struct {
	name    string
	version string
	digest  string
	ddl     []byte
}

func readUpdates() ([]update, error) {
	ent, err := fs.ReadDir(ddl, "ddl")
	if err != nil {
		return nil, err
	}
	var available []update
	computedDigest := ""
	for _, d := range ent {
		if d.IsDir() {
			panic(fmt.Errorf("directory in ddl: %s", d.Name()))
		}
		version, _, _ := strings.Cut(d.Name(), "_")
		b, err := fs.ReadFile(ddl, path.Join("ddl", d.Name()))
		if err != nil {
			panic(fmt.Errorf("cannot read ddl: %s", d.Name()))
		}
		computedDigest = hash(computedDigest, b)
		if n := len(available); n > 0 && version == available[n-1].version {
			panic(fmt.Errorf("duplicate version: %s and %s", available[n-1].name, d.Name()))
		}
		available = append(available, update{
			name:    d.Name(),
			version: version,
			digest:  computedDigest,
			ddl:     b,
		})
	}
	checkUpdates(available)
	return available, nil
}

func checkUpdates(check []update) {
	s := bufio.NewScanner(bytes.NewReader(frozen))
	for i := 0; s.Scan(); i++ {
		fields := strings.Fields(s.Text())
		if len(fields) != 3 {
			panic(fmt.Errorf("frozen.txt:%d: parse error", i+1))
		}
		if len(check) == 0 {
			panic(fmt.Errorf("frozen.txt:%d: %s is frozen but missing from ddl",
				i+1, fields[2]))
		}
		if check[0].version != fields[0] ||
			check[0].digest != fields[1] ||
			check[0].name != fields[2] {
			panic(fmt.Errorf("frozen.txt:%d: ddl no longer matches frozen update: "+
				"\nhave {%s %s %s},\nwant {%s %s %s}", i+1,
				check[0].version, check[0].digest, check[0].name,
				fields[0], fields[1], fields[2]))
		}
		check = check[1:]
	}
	if len(check) > 1 { // newest update is still in development
		panic(fmt.Errorf("ddl: %d unfrozen updates; only 1 allowed", len(check)))
	}
	checkProdUpdates(len(check))
}
