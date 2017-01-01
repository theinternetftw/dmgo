package dmgo

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const currentSnapshotVersion = 1

var supportedSnapshotVersions = []int{
	1,
}

const infoString = "dmgo snapshot"

type snapshot struct {
	Version int
	Info    string
	State   cpuState
	MBC     marshalledMBC
}

func (cs *cpuState) loadSnapshot(snapBytes []byte) (*cpuState, error) {
	reader, err := gzip.NewReader(bytes.NewReader(snapBytes))
	if err != nil {
		return nil, err
	}
	unpackedBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var snap snapshot
	err = json.Unmarshal(unpackedBytes, &snap)
	if err != nil {
		return nil, err
	}
	if snap.Version != currentSnapshotVersion {
		return nil, fmt.Errorf("old snapshot version! todo: write version converter")
	}
	snap.State.Mem.cart = cs.Mem.cart
	snap.State.Mem.mbc, err = unmarshalMBC(snap.MBC)
	if err != nil {
		return nil, err
	}

	// NOTE: what about external RAM? Doesn't this overwrite .sav files with whatever's in the snapshot?

	return &snap.State, nil
}

func (cs *cpuState) makeSnapshot() []byte {
	snap := snapshot{
		Version: currentSnapshotVersion,
		Info:    infoString,
		State:   *cs,
		MBC:     cs.Mem.mbc.Marshal(),
	}
	rawJSON, err := json.Marshal(&snap)
	if err != nil {
		panic(err)
	}
	buf := &bytes.Buffer{}
	writer := gzip.NewWriter(buf)
	writer.Write(rawJSON)
	writer.Close()
	return buf.Bytes()
}
