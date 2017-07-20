package dmgo

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

const currentSnapshotVersion = 2

const infoString = "dmgo snapshot"

type snapshot struct {
	Version int
	Info    string
	State   json.RawMessage
	MBC     marshalledMBC
}

func (cs *cpuState) loadSnapshot(snapBytes []byte) (*cpuState, error) {
	var err error
	var reader io.Reader
	var unpackedBytes []byte
	var snap snapshot
	if reader, err = gzip.NewReader(bytes.NewReader(snapBytes)); err != nil {
		return nil, err
	} else if unpackedBytes, err = ioutil.ReadAll(reader); err != nil {
		return nil, err
	} else if err = json.Unmarshal(unpackedBytes, &snap); err != nil {
		return nil, err
	} else if snap.Version < currentSnapshotVersion {
		return cs.convertOldSnapshot(&snap)
	} else if snap.Version > currentSnapshotVersion {
		return nil, fmt.Errorf("this version of dmgo is too old to open this snapshot")
	}

	// NOTE: what about external RAM? Doesn't this overwrite .sav files with whatever's in the snapshot?

	return cs.convertLatestSnapshot(&snap)
}

func (cs *cpuState) convertLatestSnapshot(snap *snapshot) (*cpuState, error) {
	var err error
	var newState cpuState
	if err = json.Unmarshal(snap.State, &newState); err != nil {
		return nil, err
	} else if newState.Mem.mbc, err = unmarshalMBC(snap.MBC); err != nil {
		return nil, err
	}
	newState.Mem.cart = cs.Mem.cart
	return &newState, nil
}

var snapshotConverters = map[int]func([]byte) ([]byte, error){

	// NOTE: Be careful with the json here. use/read the pack.go functions.
	// golang's json marshalling can sometimes do the unexpected, e.g. byte
	// slices must be packed as base64 strings.

	// NOTE: If an MBC ever has incompatible changes, the marshalledMBC will have to
	// be passed through all these conv fns as well.

	// NOTE: If new field can be zero, no need for converter.

	// added 2017-03-01
	1: func(stateBytes []byte) ([]byte, error) {
		newState := map[string]interface{}{}
		if err := json.Unmarshal(stateBytes, &newState); err != nil {
			return nil, fmt.Errorf("bad unmarshal during conversion from version one snapshot")
		}

		if vram, err := getByteSliceFromJSON(newState, "LCD", "VideoRAM"); err == nil {
			var newVRAM [0x4000]byte
			copy(newVRAM[:], vram)
			if err := replaceNodeInJSON(newState, newVRAM, "LCD", "VideoRAM"); err != nil {
				return nil, fmt.Errorf("could not convert version one snapshot: %v", err)
			}
		} else {
			return nil, fmt.Errorf("could not convert version one snapshot: %v", err)
		}

		if ram, err := getByteSliceFromJSON(newState, "Mem", "InternalRAM"); err == nil {
			var newRAM [0x8000]byte
			copy(newRAM[:], ram)
			if err := replaceNodeInJSON(newState, newRAM, "Mem", "InternalRAM"); err != nil {
				return nil, fmt.Errorf("could not convert version one snapshot: %v", err)
			}
		} else {
			return nil, fmt.Errorf("could not convert version one snapshot: %v", err)
		}

		if _, mem, err := followJSON(newState, "Mem", "InternalRAM"); err == nil {
			mem["InternalRAMBankNumber"] = 1
		} else {
			return nil, fmt.Errorf("could not convert version one snapshot: %v", err)
		}

		convertedBytes, err := json.Marshal(newState)
		if err != nil {
			return nil, fmt.Errorf("bad marshal during conversion from version one snapshot")
		}

		return convertedBytes, nil
	},
}

func (cs *cpuState) convertOldSnapshot(snap *snapshot) (*cpuState, error) {

	var err error
	var newState cpuState

	stateBytes := []byte(snap.State)

	for i := snap.Version; i < currentSnapshotVersion; i++ {
		converterFn, ok := snapshotConverters[snap.Version]
		if !ok {
			return nil, fmt.Errorf("unknown snapshot version: %v", snap.Version)
		}
		stateBytes, err = converterFn(stateBytes)
		if err != nil {
			return nil, err
		}
	}

	if err = json.Unmarshal(stateBytes, &newState); err != nil {
		return nil, fmt.Errorf("post-convert unpack err: %v", err)
	} else if newState.Mem.mbc, err = unmarshalMBC(snap.MBC); err != nil {
		return nil, fmt.Errorf("unpack mbc err: %v", err)
	}
	newState.Mem.cart = cs.Mem.cart
	return &newState, nil
}

func (cs *cpuState) makeSnapshot() []byte {
	var err error
	var csJSON []byte
	var snapJSON []byte
	if csJSON, err = json.Marshal(cs); err != nil {
		panic(err)
	}
	snap := snapshot{
		Version: currentSnapshotVersion,
		Info:    infoString,
		State:   json.RawMessage(csJSON),
		MBC:     cs.Mem.mbc.Marshal(),
	}
	if snapJSON, err = json.Marshal(&snap); err != nil {
		panic(err)
	}
	buf := &bytes.Buffer{}
	writer := gzip.NewWriter(buf)
	writer.Write(snapJSON)
	writer.Close()
	return buf.Bytes()
}
