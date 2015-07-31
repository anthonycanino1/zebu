package zebu

import (
	"fmt"
	"text/tabwriter"
	"os"
)

// Debug, erase when done
var _ = fmt.Printf

func walkFirst(n *Node) {
	if (n.op != ORULE) {
		panic("walkFirst called on non ORULE node")
	}

	cc.first[n] = make(map[*Node]bool)

	n.dumpOneLevel()

	for prods := n.rlist; prods != nil; prods = prods.next {
		// Productions for rule
		prod := prods.n
		cutoffEpsilon := false
		for elem := prod.llist; elem != nil; elem = elem.next {
			e := elem.n.left
			switch e.op {
			case OEPSILON:
				cc.first[n][nepsilon] = true

			case OREGDEF, OSTRLIT:
				cutoffEpsilon = true
				cc.first[n][e] = true

			case ORULE:

				if !e.first {
					walkFirst(e)
				}
				if !cutoffEpsilon {
					if contains := cc.first[e][nepsilon]; contains {
						cutoffEpsilon = true
					}
					for k, _ := range cc.first[e] {
						if k == nepsilon {
							continue
						}
						cc.first[n][k] = true
					}
				}

			default:
				panic(fmt.Sprintf("unexpected op %s while building first", e.op))
			}
		}
		if !cutoffEpsilon {
			cc.first[n][nepsilon] = true
		}
	}
	return
}

func printFirst(top *Node) {
	// Raw dump for now
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)

	fmt.Fprintf(w, "--------------------------------------------------------------------------------\n")
	fmt.Fprintf(w, "%s: First Set\n", top.sym)
	fmt.Fprintf(w, "--------------------------------------------------------------------------------\n")

	for n, set := range cc.first {
		fmt.Fprintf(w, "%s\t:\t", n.sym)
		for t, _ := range set {
			switch t.op {
			case OREGDEF:
				fmt.Fprintf(w, "%s\t", t.sym)
			case OSTRLIT:
				fmt.Fprintf(w, "%s\t", t.lit)
			case OEPSILON:
				fmt.Fprintf(w, "epsilon\t")
			default:
				panic(fmt.Sprintf("unexpected op %s while printing first", t.op))
			}
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "--------------------------------------------------------------------------------\n")
	w.Flush()
}

func semanticPass(n *Node) {
	cc.numSavedErrs = 0

	walkFirst(n.left)
	if cc.opt['g'] {
		printFirst(n)
	}
}
