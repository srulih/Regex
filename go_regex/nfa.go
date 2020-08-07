package main

import "log"

const (
	Split rune = 128
	Match      = 129
)

type Paren struct {
	nalt  int
	natom int
}

func re2post(re string) string {
	var nalt, natom int

	buf := make([]rune, 0)
	parens := make([]Paren, 0, 1000)
	p := 0

	for _, r := range re {
		switch r {
		case '(':
			if natom > 0 {
				natom--
				buf = append(buf, '.')
			}

			parens[p].nalt = nalt
			parens[p].natom = natom
			nalt = 0
			natom = 0

		case '|':
			if natom == 0 {
				log.Fatal("expected number of atoms to be larger then zero")
			}
			natom--
			for natom > 0 {
				buf = append(buf, '.')
				natom--
			}
			nalt++
		case ')':
			natom--
			for natom > 0 {
				buf = append(buf, '.')
				natom--
			}

			for ; nalt > 0; nalt-- {
				buf = append(buf, '|')
			}
			p--
			nalt = parens[p].nalt
			natom = parens[p].natom
			natom++
		case '*', '+', '?':
			buf = append(buf, r)
		default:
			if natom > 1 {
				natom--
				buf = append(buf, '.')
			}
			buf = append(buf, r)
			natom++
		}
	}
	if p != 0 {
		log.Fatal("unmatched parens")
	}

	natom--
	for natom > 0 {
		buf = append(buf, '.')
		natom--
	}
	for ; nalt > 0; nalt-- {
		buf = append(buf, '|')
	}
	return string(buf)
}

func re2post2(re string) string {
	stack := make([]rune, 0)
	buf := make([]rune, 0)

	natom := 0
	precMap := map[rune]int{
		'*': 30,
		'+': 30,
		'?': 30,
		'.': 20,
		'|': 10,
	}

	for _, r := range re {
		switch r {
		case '(':
			if natom > 1 {
				natom--
				concatOp := '.'
				p := len(stack) - 1
				for p >= 0 && precMap[stack[p]] >= precMap[concatOp] {
					buf = append(buf, stack[p])
					p--
				}
				if p == -1 {
					p = 0
				}
				stack = stack[:p]
				stack = append(stack, '.')
			}
			stack = append(stack, r)
		case ')':
			p := len(stack) - 1
			for stack[p] != ')' {
				buf = append(buf, stack[p])
				p--
			}
			stack = stack[:p+1]
		case '|':
			if natom > 1 {
				natom--
				concatOp := '.'
				p := len(stack) - 1
				for p >= 0 && precMap[stack[p]] >= precMap[concatOp] {
					buf = append(buf, stack[p])
					p--
				}
				if p == -1 {
					p = 0
				}
				stack = stack[:p+1]
				stack = append(stack, '.')
			}
			natom = 0
			stack = append(stack, r)
		case '*', '+', '?':
			p := len(stack) - 1
			for p >= 0 && precMap[stack[p]] >= precMap[r] {
				buf = append(buf, stack[p])
				p--
			}

			stack = stack[:p+1]
			stack = append(stack, r)

		default:
			natom++
			if natom > 1 {
				natom--
				concatOp := '.'
				p := len(stack) - 1
				for p >= 0 && precMap[stack[p]] >= precMap[concatOp] {
					buf = append(buf, stack[p])
					p--
				}
				if p == -1 {
					p = 0
				}
				stack = stack[:p]
				stack = append(stack, '.')
			}
			buf = append(buf, r)

		}
	}

	for p := len(stack) - 1; p >= 0; p-- {
		buf = append(buf, stack[p])
	}

	return string(buf)
}

type State struct {
	c    rune
	out  *State
	out1 *State
}

func (s State) PreOrder() []rune {
	rs := make([]rune, 0)
	s.preorder(&rs)
	return rs
}

func (s State) preorder(rs *[]rune) {
	*rs = append(*rs, s.c)
	if s.out != nil {
		s.out.preorder(rs)
	}
	if s.out1 != nil {
		s.out1.preorder(rs)
	}
}

type Frag struct {
	start   *State
	ptrlist []**State
}

type Ptrlist []**State

func patch(pl Ptrlist, state *State) {
	for _, p := range pl {
		*p = state
	}
}

var matchstate *State = &State{Match, nil, nil}

func post2nfa(postfix string) *State {
	var stack []Frag

	for _, r := range postfix {
		switch r {
		default:
			s := &State{r, nil, nil}
			stack = append(stack, Frag{s, []**State{&s.out}})
		case '.':
			p := len(stack) - 1
			e2 := stack[p]
			e1 := stack[p-1]
			stack = stack[:p-1]
			patch(e1.ptrlist, e2.start)
			stack = append(stack, Frag{e1.start, e2.ptrlist})
		case '|':
			p := len(stack) - 1
			e2 := stack[p]
			e1 := stack[p-1]
			stack = stack[:p-1]
			pl := append(e1.ptrlist, e2.ptrlist...)
			stack = append(stack, Frag{e1.start, pl})
		case '?':
			p := len(stack) - 1
			e := stack[p]
			stack = stack[:p]
			s := State{Split, e.start, nil}
			pl := append(e.ptrlist, &s.out1)
			stack = append(stack, Frag{e.start, pl})
		case '*':
			p := len(stack) - 1
			e := stack[p]
			stack = stack[:p]
			s := &State{Split, e.start, nil}
			patch(e.ptrlist, s)
			stack = append(stack, Frag{s, []**State{&s.out1}})
		case '+':
			p := len(stack) - 1
			e := stack[p]
			stack = stack[:p]
			s := &State{Split, e.start, nil}
			patch(e.ptrlist, s)
			stack = append(stack, Frag{e.start, []**State{&s.out1}})
		}
	}

	e := stack[0]
	patch(e.ptrlist, matchstate)
	return e.start
}

type List []*State

func ismatch(list List) bool {
	for _, s := range list {
		if s.c == Match {
			return true
		}
	}

	return false
}

func addstate(list *List, s *State) {
	if s == nil {
		return
	}
	if s.c == Split {
		addstate(list, s.out)
		addstate(list, s.out1)
		return
	}

	*list = append(*list, s)
}

func step(clist List, r rune) List {
	nlist := make(List, 0, len(clist))
	for _, s := range clist {
		if s.c == r {
			addstate(&nlist, s.out)
		}
	}

	return nlist
}

func match(start *State, s string) bool {

	list := make(List, 0)
	addstate(&list, start)

	for _, r := range s {
		list = step(list, r)
	}
	return ismatch(list)
}
