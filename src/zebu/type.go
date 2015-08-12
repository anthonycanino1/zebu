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
	for dcls := top.rlist; dcls != nil; dcls = dcls.next {
		dcl := dcls.n
		if dcl.op != ORULE {
			continue
		}

		// First pass to categorize our possible left factor paths
		paths := make(map[*Node][]*Node)

		for prods := dcl.rlist; prods != nil; prods = prods.next {
			prod := prods.n
			fst := prod.llist.n.left
			paths[fst] = append(paths[fst], prod)
		}

		newProds := new(NodeList)

		for _, set := range paths {
			if len(set) <= 1 {
				newProds.add(set[0])
				continue
			}

			// Grab the greatest common left factor for each production
			// Set will save the state of our iteration
			common := new(NodeList)

			factors := make([]*NodeList, 0, len(set))
			for i := 0; i < len(set); i++ {
				factors = append(factors, set[i].llist)
			}

		nomorecommon:
			for factors[0] != nil {
				for i := 1; i < len(factors); i++ {
					if factors[0].n.left != factors[i].n.left {
						break nomorecommon
					}
					factors[i] = factors[i].next
				}
				common.add(factors[0].n)
				factors[0] = factors[0].next
			}

			// create a new rule for the common elems, and factor
			// out from each node
			fact := nodeRuleFromFactoring(dcl, &factors)
			top.rlist.add(fact)

			// CODE : Maybe pull this into a cons? I don't like how it
			// is very specific, it basically becomes a functions for this
			// use only
			newProds.add(&Node{
				op: OPROD,
				llist: common.concat(nodeAsList(&Node{
					op:   OPRODELEM,
					left: fact,
				})),
			})
		}

		dcl.rlist = newProds
	}
}

func removeDirectRecursion(top *Node) {
	//newRules := new(NodeList)
	for dcls := top.rlist; dcls != nil; dcls = dcls.next {
		dcl := dcls.n
		if dcl.op != ORULE {
			continue
		}

		nonRec := new(NodeList)
		var leftRec *Node = nil

		for prods := dcl.rlist; prods != nil; prods = prods.next {
			prod := prods.n
			fst := prod.llist.n
			switch {
			case fst.left == dcl:
				if leftRec != nil {
					panic(fmt.Sprintf("multiple left recursion found for %s. left factoring should prevent this.", dcl.sym))
				}
				leftRec = prod
			default:
				nonRec.add(prod)
			}
		}
		if leftRec == nil {
			continue
		}

		rule := nodeRuleFromLeftRecursion(dcl, leftRec)
		top.rlist.add(rule)

		for prods := nonRec; prods != nil; prods = prods.next {
			prod := prods.n
			prod.llist.add(rule.prodElem())
		}

		dcl.rlist = nonRec
	}
}

func removeIndirectRecursion(top *Node) {
}

func buildFirst(n *Node) {
	if n.op != OGRAM {
		panic("buildFirst called on non OGRAM node")
	}

	// Init table
	for dcls := n.rlist; dcls != nil; dcls = dcls.next {
		dcl := dcls.n
		if dcl.op != ORULE {
			continue
		}
		cc.first[dcl] = make(map[*Node]bool)
	}

	anotherPass := true
	for anotherPass {
		anotherPass = false

		for dcls := n.rlist; dcls != nil; dcls = dcls.next {
			dcl := dcls.n
			if dcl.op != ORULE {
				continue
			}

		productions:
			for prods := dcl.rlist; prods != nil; prods = prods.next {
				// Productions for rule
				prod := prods.n
				for elem := prod.llist; elem != nil; elem = elem.next {
					e := elem.n.left
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
	for dcls := top.rlist; dcls != nil; dcls = dcls.next {
		dcl := dcls.n
		if dcl.op != ORULE {
			continue
		}
		cc.follow[dcl] = make(map[*Node]bool)
	}

	anotherPass := true
	for anotherPass {
		anotherPass = false

		for dcls := top.rlist; dcls != nil; dcls = dcls.next {
			dcl := dcls.n
			if dcl.op != ORULE {
				continue
			}

			for prods := dcl.rlist; prods != nil; prods = prods.next {
				// Productions for rule
				prod := prods.n
				for elem := prod.llist; elem != nil; elem = elem.next {
					e := elem.n.left
					if e.op != ORULE {
						continue
					}

					var next *Node = nil
					if elem.next != nil {
						next = elem.next.n.left
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
}

func typeCheck(top *Node) {
	// 1. Perform transformation of the grammar, aiding the user
	// in writing a LL(1) language.
	leftFactor(top)	
	removeDirectRecursion(top)
	removeIndirectRecursion(top)

	// 2. Create first and follow sets to analyze the transformed
	// grammar.
	buildFirst(top)
	buildFollow(top)

	if (cc.opt['g']) {
		printFirst(top)
		printFollow(top)
	}

	// 3. Check for LL(1) grammar
	ll1Check(top)
}

