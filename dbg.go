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

const (
	breakOpChange int = iota
	breakOpEq
	breakOpNeq

	// TODO: add these once we parse vals/args into ints and such
	// breakOpGt
	// breakOpGte
	// breakOpLt
	// breakOpLte
)

var breakOpsMap = map[string]int{
	"change": breakOpChange,
	"=":      breakOpEq,
	"==":     breakOpEq,
	"!=":     breakOpNeq,

	// ">": breakOpGt,
	// ">=": breakOpGte,
	// "<": breakOpLt,
	// "<=": breakOpLte,
}

type breakpoint struct {
	fieldPath string
	breakVal  string
	op        int
}

type debugger struct {
	keysJustPressed [256]bool
	keys            [256]bool
	lineBuf         []byte
	state           int
	breakpoints     []breakpoint
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
func getField(emu Emulator, path string) (reflect.Value, bool) {
	root := reflect.Indirect(reflect.ValueOf(emu))
	return lookupValue(root, strings.Split(path, "."))
}
func getMethod(emu Emulator, path string) (reflect.Value, bool) {
	root := reflect.Indirect(reflect.ValueOf(emu))
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
	var ok bool
	_, ok = t.MethodByName(parts[len(parts)-1])
	if ok {
		return v.MethodByName(parts[len(parts)-1]), true
	}
	// also allow pointer receivers
	v = v.Addr()
	t = v.Type()
	_, ok = t.MethodByName(parts[len(parts)-1])
	if ok {
		return v.MethodByName(parts[len(parts)-1]), true
	}
	fmt.Println("method not found or private")
	return reflect.Value{}, false
}

func strIndexOf(strs []string, str string) int {
	for i := range strs {
		if strs[i] == str {
			return i
		}
	}
	return -1
}

var dbgCmdMap = map[string]func(*debugger, Emulator, []string){
	"run": func(d *debugger, emu Emulator, arg []string) {
		d.state = dbgStateRun
	},
	"x": func(d *debugger, emu Emulator, arg []string) {
		if len(arg) == 0 {
			fmt.Println("usage: x FIELD_PATH")
			return
		}
		if v, ok := getField(emu, arg[0]); ok {
			fmt.Println(v)
		}
	},
	"break": func(d *debugger, emu Emulator, arg []string) {
		if len(arg) == 0 {
			fmt.Println("usage: break FIELD_NAME OP [VAL]")
			return
		}
		v, field_ok := getField(emu, arg[0])
		if !field_ok {
			return
		}
		opStr := "change"
		if len(arg) > 1 {
			opStr = arg[1]
		}
		op, op_ok := breakOpsMap[opStr]
		if !op_ok {
			fmt.Println("bad OP arg for break")
			return
		}
		var valStr string
		if op != breakOpChange {
			if len(arg) < 3 {
				fmt.Println("need val for break op of", opStr)
				return
			}
			valStr = arg[2]
		} else {
			valStr = fmt.Sprintf("%v", v) // change works like != lastVal
		}
		bp := breakpoint{fieldPath: arg[0], op: op, breakVal: valStr}
		d.breakpoints = append(d.breakpoints, bp)
	},
	"call": func(d *debugger, emu Emulator, arg []string) {
		if len(arg) == 0 {
			fmt.Println("usage: call METHOD_PATH")
			return
		}
		if len(arg) > 1 {
			fmt.Println("method args not yet impl")
			return
		}
		if v, ok := getMethod(emu, arg[0]); ok {
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
		for i := range d.breakpoints {
			bp := &d.breakpoints[i]
			f, ok := getField(emu, bp.fieldPath)
			if !ok {
				fmt.Println("couldn't find field listed in breakpoint, something screwy's going on...")
				d.state = dbgStateNewCmd
				return
			}
			valStr := fmt.Sprintf("%v", f)
			// fmt.Println("checking bp for", bp.fieldPath, "val is", valStr)
			switch bp.op {
			case breakOpChange:
				if valStr != bp.breakVal {
					fmt.Println("hit breakpoint:", bp.fieldPath, "changed from", bp.breakVal, "to", valStr)
					bp.breakVal = valStr
					d.state = dbgStateNewCmd
					return
				}
			case breakOpEq:
				if valStr == bp.breakVal {
					fmt.Println("hit breakpoint:", f, "==", valStr)
					d.state = dbgStateNewCmd
					return
				}
			case breakOpNeq:
				if valStr != bp.breakVal {
					fmt.Println("hit breakpoint:", f, "!=", bp.breakVal, "- now", valStr)
					d.state = dbgStateNewCmd
					return
				}
			default:
				fmt.Println("unexpected bp op, something screwy's going on...")
				d.state = dbgStateNewCmd
				return
			}
		}
		emu.Step()
	}
}

func (d *debugger) updateInput(keys []bool) {
	for i := range d.keys {
		d.keysJustPressed[i] = keys[i] && !d.keys[i]
		d.keys[i] = keys[i]
	}
}
