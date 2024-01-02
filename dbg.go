package dmgo

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	dbgStateNewCmd int = iota
	dbgStateInCmd
	dbgStateRun
)

type debugger struct {
	keysJustPressed [256]bool
	keys            [256]bool
	lineBuf         []byte
	state           int
}

func lookupValue(root reflect.Value, lookups []string) (reflect.Value, bool) {
	v := root
	t := root.Type()
	for i := range lookups {
		if t.Kind() != reflect.Struct {
			fmt.Println("field", lookups[i], "is not a struct but field name lookup was asked for")
		}
		_, ok := t.FieldByName(lookups[i])
		if !ok {
			fmt.Println("field", lookups[i], "not found")
			return reflect.Value{}, false
		}
		v = v.FieldByName(lookups[i])
		t = v.Type()
	}
	return v, true
}
func getField(root reflect.Value, path string) (reflect.Value, bool) {
	return lookupValue(root, strings.Split(path, "."))
}
func getMethod(root reflect.Value, path string) (reflect.Value, bool) {
	v := root
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		var ok bool
		v, ok = lookupValue(root, parts[:len(parts)-1])
		if !ok {
			return reflect.Value{}, false
		}
	}
	t := v.Type()
	fmt.Println(t)
	fmt.Println(parts[len(parts)-1])
	var ok bool
	_, ok = t.MethodByName(parts[len(parts)-1])
	if ok {
		return v.MethodByName(parts[len(parts)-1]), true
	}
	// also allow pointer receivers
	v = v.Addr()
	t = v.Type()
	fmt.Println(t)
	_, ok = t.MethodByName(parts[len(parts)-1])
	if ok {
		return v.MethodByName(parts[len(parts)-1]), true
	}
	fmt.Println("method not found or private")
	return reflect.Value{}, false
}

var dbgCmdMap = map[string]func(*debugger, Emulator, []string){
	"run": func(d *debugger, emu Emulator, arg []string) {
		d.state = dbgStateRun
	},
	"x": func(d *debugger, emu Emulator, arg []string) {
		if len(arg) == 0 {
			fmt.Println("usage: x FIELD_PATH")
		}
		root := reflect.Indirect(reflect.ValueOf(emu))
		if v, ok := getField(root, arg[0]); ok {
			fmt.Println(v)
		}
	},
	"call": func(d *debugger, emu Emulator, arg []string) {
		if len(arg) == 0 {
			fmt.Println("usage: call METHOD_PATH")
		}
		if len(arg) > 1 {
			fmt.Println("method args not yet impl")
		}
		root := reflect.Indirect(reflect.ValueOf(emu))
		if v, ok := getMethod(root, arg[0]); ok {
			results := v.Call([]reflect.Value{})
			if len(results) > 0 {
				fmt.Println(results)
			}
		}
	},
}

func (d *debugger) step(emu Emulator) {
	switch d.state {
	case dbgStateNewCmd:
		d.lineBuf = []byte{}
		d.state = dbgStateInCmd
		fmt.Printf("\n> ")
	case dbgStateInCmd:
		for i := range d.keysJustPressed {
			if d.keysJustPressed[i] {
				d.lineBuf = append(d.lineBuf, byte(i))
				if rune(i) != '\b' {
					fmt.Printf("%c", rune(i))
				}
			}
		}
		if d.keysJustPressed['\b'] {
			d.lineBuf = d.lineBuf[:len(d.lineBuf)-1]
			if len(d.lineBuf) > 0 {
				d.lineBuf = d.lineBuf[:len(d.lineBuf)-1]
				fmt.Print("\b \b")
			}
		} else if d.keysJustPressed['\n'] {
			fields := strings.Fields(string(d.lineBuf))
			d.state = dbgStateNewCmd
			if len(fields) == 0 {
				break
			}
			if cmd, ok := dbgCmdMap[fields[0]]; ok {
				cmd(d, emu, fields[1:])
			} else {
				fmt.Println("unknown cmd")
			}
		}
	case dbgStateRun:
		emu.Step()
	}
}

func (d *debugger) updateInput(keys []bool) {
	for i := range d.keys {
		d.keysJustPressed[i] = keys[i] && !d.keys[i]
		d.keys[i] = keys[i]
	}
}
