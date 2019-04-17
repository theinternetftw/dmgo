package dmgo

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

const currentSnapshotVersion = 3

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
	}
	if newState.Mem.mbc, err = unmarshalMBC(snap.MBC); err != nil {
		return nil, err
	}
	newState.Mem.cart = cs.Mem.cart

	newState.devMode = cs.devMode

	return &newState, nil
}

var snapshotConverters = map[int]func(map[string]interface{}) error{

	// NOTE: Be careful with the json here. use/read the pack.go functions.
	// golang's json marshalling can sometimes do the unexpected, e.g. byte
	// slices must be packed as base64 strings.

	// NOTE: If an MBC ever has incompatible changes, the marshalledMBC will have to
	// be passed through all these conv fns as well.

	// NOTE: If new field can be zero, no need for converter.

	// added 2017-03-01
	1: func(state map[string]interface{}) error {

		if vramStr, lcd, err := followJSON(state, "LCD", "VideoRAM"); err == nil {
			if vram, err := getByteSliceFromJSON(vramStr); err == nil {
				newVRAM := [0x4000]byte{}
				copy(newVRAM[:], vram)
				lcd["VideoRAM"] = newVRAM
			} else {
				return fmt.Errorf("could not convert old v1 snapshot: %v", err)
			}
		} else {
			return fmt.Errorf("could not convert old v1 snapshot: %v", err)
		}

		if ramStr, mem, err := followJSON(state, "Mem", "InternalRAM"); err == nil {
			if ram, err := getByteSliceFromJSON(ramStr); err == nil {
				var newRAM [0x8000]byte
				copy(newRAM[:], ram)
				mem["InternalRAM"] = newRAM
				mem["InternalRAMBankNumber"] = 1
			} else {
				return fmt.Errorf("could not convert old v1 snapshot: %v", err)
			}
		} else {
			return fmt.Errorf("could not convert old v1 snapshot: %v", err)
		}

		return nil
	},

	// added 2018-12-21
	2: func(state map[string]interface{}) error {
		if apu, _, err := followJSON(state, "APU"); err == nil {
			if apuMap, ok := apu.(map[string]interface{}); ok {
				apuMap["EnvTimeCounter"] = 0
				apuMap["SweepTimeCounter"] = 0
				apuMap["LengthTimeCounter"] = 0
			} else {
				return fmt.Errorf("could not convert old v2 snapshot: apu var is of unknown type")
			}
		} else {
			return fmt.Errorf("could not convert old v2 snapshot: %v", err)
		}

		if sounds, _, err := followJSON(state, "APU", "Sounds"); err == nil {
			if soundsArr, ok := sounds.([]interface{}); ok {
				for _, sound := range soundsArr {
					if soundMap, ok := sound.(map[string]interface{}); ok {
						soundMap["T"] = 0
						soundMap["PolySample"] = 0
					} else {
						return fmt.Errorf("could not convert old v2 snapshot: Sound var is of unknown type")
					}
				}
			} else {
				return fmt.Errorf("could not convert old v2 snapshot: Sounds var is of unknown type")
			}
		} else {
			return fmt.Errorf("could not convert old v2 snapshot: %v", err)
		}
		return nil
	},
}

func (cs *cpuState) convertOldSnapshot(snap *snapshot) (*cpuState, error) {

	var state map[string]interface{}
	if err := json.Unmarshal(snap.State, &state); err != nil {
		return nil, fmt.Errorf("json unpack err: %v", err)
	}

	for i := snap.Version; i < currentSnapshotVersion; i++ {
		if converterFn, ok := snapshotConverters[i]; !ok {
			return nil, fmt.Errorf("could not find converter for snapshot version: %v", i)
		} else if err := converterFn(state); err != nil {
			return nil, fmt.Errorf("error converting snapshot version %v: %v", i, err)
		}
	}

	var err error
	if snap.State, err = json.Marshal(state); err != nil {
		return nil, fmt.Errorf("json pack err: %v", err)
	}

	return cs.convertLatestSnapshot(snap)
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
