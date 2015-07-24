package main

import (
	"github.com/anthonycanino1/zebu/src/zebu"
	"fmt"
	"os"
)

type Flag struct {
	short byte
	long string
	help string
}

type FlagMap struct {
	flags []*Flag
	shortMap map[byte]*Flag
	longMap map[string]*Flag
}

func (fm *FlagMap) addFlag(s byte, l string, h string) {
	f := &Flag{
		short: s,
		long: l,
		help: h,
	}
	_, ok1 := fm.shortMap[s]
	_, ok2 := fm.longMap[l]
	if ok1 || ok2 {
		panic("conflict in flag map")
	}
	fm.flags = append(fm.flags, f)
	fm.shortMap[s] = f
	fm.longMap[l] = f
	return
}

func (fm *FlagMap) hasShort(s byte) bool {
	_, ok := fm.shortMap[s]
	return ok
}

func (fm *FlagMap) hasLong(l string) bool {
	_, ok := fm.longMap[l]
	return ok
}

func (fm *FlagMap) shortFromLong(l string) (byte, bool) {
	if f, ok := fm.longMap[l]; ok {
		return f.short, true
	}
	return 0, false
}

func (fm *FlagMap) printHelp() {
	fmt.Printf("\nzebu usage: -opt [files]\n")
	for _, f := range fm.flags {
		fmt.Printf("-%c --%s %s\n", f.short, f.long, f.help)
	}
}

var flagMap *FlagMap = nil

func init() {
	flagMap = &FlagMap {
		flags: make([]*Flag, 0, 10),
		shortMap: make(map[byte]*Flag),
		longMap: make(map[string]*Flag),
	}

	flagMap.addFlag('d', "dump", "dump the ast after parsing pass")
	flagMap.addFlag('h', "help", "print this help message")
}

func parseArgs(osArgs []string) (args []string, flags [512]bool, ok bool) {
	if len(osArgs) == 0 {
		return
	}

	ok = true
	for _, e := range osArgs {
		if e[0] == '-' {
			if len(e) == 1 || (len(e) == 2 && e[1] == '-') {
				fmt.Printf("invalid option %s\n", e)
				ok = false
				continue
			}
			var s byte
			if e[1] == '-' {
				l := e[2:]
				var ok1 bool
				if s, ok1 = flagMap.shortFromLong(l); !ok1 {
					fmt.Printf("invalid long option %s\n", l)
					ok = false
					continue
				}
			} else {
				if len(e) != 2 {
					fmt.Printf("short option must be single char only\n")
					ok = false
					continue
				}
				s = e[1]
				if !flagMap.hasShort(s) {
					fmt.Printf("invalid short option %c\n", s)
					ok = false
					continue
				}
			}

			flags[s] = true
		} else {
			args = append(args, e)
		}
	}
	return
}

func main() {
	osArgs := os.Args[1:]
	_, flags, ok := parseArgs(osArgs)

	if flags['h'] || !ok {
		flagMap.printHelp()
		return
	}

	zebu.Options(flags)
	/*
	zebu.Compile(file)
	*/
}
