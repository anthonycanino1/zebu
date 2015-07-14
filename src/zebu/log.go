package zebu

import (
	"fmt"
)

type Debug struct {
	ind   int
	funcs []string
}

func NewDebug() (d *Debug) {
	d = &Debug{
		ind:   0,
		funcs: make([]string, 8),
	}
	return
}

func (d *Debug) enter(f string) {
	d.funcs = append(d.funcs, f)
	d.ind++
	fmt.Printf("[Enter %s]\n", d.funcs[len(d.funcs)-1])
}

func (d *Debug) exit() {
	if d.ind == 0 {
		panic("exit on zero ind")
	}
	d.ind--
	fmt.Printf("[Exit %s]\n", d.funcs[len(d.funcs)-1])
	d.funcs = d.funcs[0 : len(d.funcs)-1]
}

func (d *Debug) log(fmtString string, args ...interface{}) {
	for i := 0; i < d.ind; i++ {
		fmt.Printf("\t")
	}
	fmt.Printf("%s:%s\n", d.funcs[len(d.funcs)-1], fmt.Sprintf(fmtString, args...))
}

// Defaults to make it easier to debug
var localDbg *Debug = NewDebug()

func dbgEnter(f string) {
	localDbg.enter(f)
}

func dbgExit() {
	localDbg.exit()
}

func dbg(fmtString string, args ...interface{}) {
	localDbg.log(fmtString, args...)
}
