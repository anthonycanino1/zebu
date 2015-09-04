package zebu

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// Debug, erase when done
var _ = fmt.Printf

func primeName(name string) string {
	namep := name
	s := cc.symbols.lookup(namep)
	for s.defn != nil {
		namep = fmt.Sprintf("%s'", namep)
		s = cc.symbols.lookup(namep)
	}
	return namep
}

func leftFactor(top *Node) {
	for _, dcl := range top.nodes {
		if dcl.op != ORULE {
			continue
		}

		// First pass to categorize our possible left factor paths
		paths := make(map[*Node][]*Node)

		for _, prod := range dcl.nodes {
			fst := prod.nodes[0].left
			paths[fst] = append(paths[fst], prod)
		}

		newProds := make([]*Node, 0)

		for _, set := range paths {
			if len(set) <= 1 {
				newProds = append(newProds, set[0])
				continue
			}

			// Grab the greatest common left factor for each production
			// Set will save the state of our iteration
			common := make([]*Node, 0)

			factors := make([][]*Node, 0, len(set))
			for i := 0; i < len(set); i++ {
				factors = append(factors, set[i].nodes)
			}

			last := -1
		nomorecommon:
			for i := 0; i < len(factors[0]); i++ {
				for j := 1; j < len(factors); j++ {
					if i >= len(factors[j]) || factors[0][i].left != factors[j][i].left {
						break nomorecommon
					}
				}
				common = append(common, factors[0][i])
				last = i
			}
			if (last == -1) {
				panic("nothing found in common after determing a common path")
			}

			// CODE : I don't like how nodeRuleFromFactoring is used,
			// doesn't really make sense to have a cons, come back to fix
			for i := 0; i < len(factors); i++ {
				if last >= len(factors[i]) {
					continue
				}
				factors[i] = factors[i][last+1:]
			}

			// create a new rule for the common elems, and factor
			// out from each node
			fact := nodeRuleFromFactoring(dcl, factors)
			top.nodes = append(top.nodes, fact)

			// CODE : Maybe pull this into a cons? I don't like how it
			// is very specific, it basically becomes a function for this
			// use only
			newProds = append(newProds, &Node{
				op: OPROD,
				nodes: append(common, &Node{
					op:   OPRODELEM,
					left: fact,
				}),
			})
		}

		dcl.nodes = newProds
	}
}

func removeDirectRecursion(top *Node) {
	for _, dcl := range top.nodes {
		if dcl.op != ORULE {
			continue
		}

		nonRec := make([]*Node, 0)
		var leftRec *Node = nil

		for _, prod := range dcl.nodes {
			fst := prod.nodes[0]
			switch {
			case fst.left == dcl:
				if leftRec != nil {
					panic(fmt.Sprintf("multiple left recursion found for %s. left factoring should prevent this.", dcl.sym))
				}
				leftRec = prod
			default:
				nonRec = append(nonRec, prod)
			}
		}
		if leftRec == nil {
			continue
		}

		rule := nodeRuleFromLeftRecursion(dcl, leftRec)
		top.nodes = append(top.nodes, rule)

		for _, prod := range nonRec {
			prod.nodes = append(prod.nodes, rule.prodElem())
		}

		dcl.nodes = nonRec
	}
}

func removeIndirectRecursion(top *Node) {
}

func buildFirst(top *Node) {
	if top.op != OGRAM {
		panic("buildFirst called on non OGRAM node")
	}

	// Init table
	for _, dcl := range top.nodes {
		if dcl.op != ORULE {
			continue
		}
		cc.first[dcl] = make(map[*Node]bool)
	}

	anotherPass := true
	for anotherPass {
		anotherPass = false

		for _, dcl := range top.nodes {
			if dcl.op != ORULE {
				continue
			}

		productions:
			for _, prod := range dcl.nodes {
				for _, elem := range prod.nodes {
					e := elem.left
					switch e.op {
					case OEPSILON:
						if !cc.first[dcl][e] {
							cc.first[dcl][e] = true
							anotherPass = true
						}

					case OREGDEF, OSTRLIT:
						if !cc.first[dcl][e] {
							cc.first[dcl][e] = true
							anotherPass = true
						}
						continue productions

					case ORULE:
						for k, _ := range cc.first[e] {
							if !cc.first[dcl][k] {
								cc.first[dcl][k] = true
								anotherPass = true
							}
						}
						if !cc.first[e][nepsilon] {
							continue productions
						}

					default:
						panic(fmt.Sprintf("unexpected op %s while building first", e.op))
					}
				}
				if !cc.first[dcl][nepsilon] {
					cc.first[dcl][nepsilon] = true
					anotherPass = true
				}
			}
		}
	}
	return
}

func buildFollow(top *Node) {
	if top.op != OGRAM {
		panic("buildFirst called on non OGRAM node")
	}

	// Init table
	for _, dcl := range top.nodes {
		if dcl.op != ORULE {
			continue
		}
		cc.follow[dcl] = make(map[*Node]bool)
	}

	anotherPass := true
	for anotherPass {
		anotherPass = false

		for _, dcl := range top.nodes {
			if dcl.op != ORULE {
				continue
			}

			for _, prod := range dcl.nodes {
				for j, elem := range prod.nodes {
					e := elem.left
					if e.op != ORULE {
						continue
					}

					var next *Node = nil
					if j+1 < len(prod.nodes) {
						next = prod.nodes[j+1].left
					}

					if next != nil {
						switch next.op {
						case OREGDEF, OSTRLIT:
							if !cc.follow[e][next] {
								cc.follow[e][next] = true
								anotherPass = true
							}
						case ORULE:
							for k, _ := range cc.first[next] {
								if k == nepsilon {
									continue
								}
								if !cc.follow[e][k] {
									cc.follow[e][k] = true
									anotherPass = true
								}
							}
						case OEPSILON:
						default:
							panic(fmt.Sprintf("unexpected op %s found while building follow set\n", next))
						}
					}

					if next == nil || next.op == OEPSILON || (next.op == ORULE && cc.first[next][nepsilon]) {
						for k, _ := range cc.follow[dcl] {
							if !cc.follow[e][k] {
								cc.follow[e][k] = true
								anotherPass = true
							}
						}
					}
				}
			}
		}
	}
	return
}

func printSet(top *Node, name string, set *map[*Node]map[*Node]bool) {
	// Raw dump for now
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)

	fmt.Fprintf(w, "--------------------------------------------------------------------------------\n")
	fmt.Fprintf(w, "%s: %s Set\n", top.sym, name)
	fmt.Fprintf(w, "--------------------------------------------------------------------------------\n")

	for n, nset := range *set {
		fmt.Fprintf(w, "%s\t:\t", n.sym)
		for t, _ := range nset {
			switch t.op {
			case OREGDEF:
				fmt.Fprintf(w, "%s\t", t.sym)
			case OSTRLIT:
				fmt.Fprintf(w, "%s\t", escapeStrlit(t.lit.lit))
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

func printFirst(top *Node) {
	printSet(top, "First", &cc.first)
}

func printFollow(top *Node) {
	printSet(top, "Follow", &cc.follow)
}

func ll1Check(top *Node) {
	for _, dcl := range top.nodes {
		if dcl.op != ORULE {
			continue
		}

		disjoint := make(map[*Node]bool)
		followNotAdded := true
		for _, prod := range dcl.nodes {
			fst := prod.nodes[0].left
			if fst == nil {
				panic("unexpected nil in production element list")
			}
			switch fst.op {
			case OSTRLIT, OREGDEF:
				if disjoint[fst] {
					if (dcl.orig != nil) {
						cc.error(dcl.orig.pos, "%s is ambiguous", dcl.orig.sym)
					} else {
						cc.error(dcl.pos, "%s is ambiguous", dcl.sym)
					}
				}
				disjoint[fst] = true
			case OEPSILON:
				if followNotAdded {
					for k, _ := range cc.follow[dcl] {
						if disjoint[k] {
							if (dcl.orig != nil) {
								cc.error(dcl.orig.pos, "%s is ambiguous", dcl.orig.sym)
							} else {
								cc.error(dcl.pos, "%s is ambiguous", dcl.sym)
							}
						}
						disjoint[k] = true
					}
					followNotAdded = false
				}
			case ORULE:
				if cc.first[fst][nepsilon] && followNotAdded {
					// Use follow[dcl]
					for k, _ := range cc.follow[dcl] {
						if disjoint[k] {
							if (dcl.orig != nil) {
								cc.error(dcl.orig.pos, "%s is ambiguous", dcl.orig.sym)
							} else {
								cc.error(dcl.pos, "%s is ambiguous", dcl.sym)
							}
						}
						disjoint[k] = true
					}
					followNotAdded = false
				} else {
					// Use first[fst]
					for k, _ := range cc.first[fst] {
						if disjoint[k] {
							if (dcl.orig != nil) {
								cc.error(dcl.orig.pos, "%s is ambiguous", dcl.orig.sym)
							} else {
								cc.error(dcl.pos, "%s is ambiguous", dcl.sym)
							}
						}
						disjoint[k] = true
					}
				}
			default:
				panic(fmt.Sprintf("unexpected op %s in production\n", fst.op))
			}
		}

	}
}

// LAST : Check leftFactor and removeDirectRecursion again

func typeCheck(top *Node) {
	// 1. Perform transformation of the grammar, aiding the user
	// in writing a LL(1) language.
	leftFactor(top)
	removeDirectRecursion(top)
	//removeIndirectRecursion(top)

	if cc.opt['t'] {
		top.dumpTree()
	}

	// 2. Create first and follow sets to analyze the transformed
	// grammar.
	buildFirst(top)
	buildFollow(top)

	if cc.opt['g'] {
		printFirst(top)
		printFollow(top)
	}

	// 3. Check for LL(1) grammar
	ll1Check(top)
}
