package main

import (
	"fmt"
)

type Procedure interface {
	//Runs the procedure with the given arguments
	Apply(args *SexpPair, env *Env) (result Expression, newEnv *Env, err string)

	//Sets the name of the Procedure if it doesn't have one already
	GiveName(name string)

	Expression
}

//TODO: decide if this should be called PTProc or something
type Proc struct {
	Name    string
	Vars    []Symbol
	Exp     Expression
	EvalEnv *Env
}

var _ Procedure = &Proc{}

func (p *Proc) Apply(args *SexpPair, env *Env) (result Expression, newEnv *Env, err string) {
	argSlice, err := evalArgs(args, env)
	if err != "" {
		return nil, nil, err
	}

	newEnv = MakeEnv(p.Vars, argSlice, p.EvalEnv)

	// Set the expression to be evaluated
	return p.Exp, newEnv, ""
}

func (p *Proc) GiveName(name string) {
	if p.Name == "" {
		p.Name = name
	}
}

func (p *Proc) String() string {
	if p.Name != "" {
		return fmt.Sprintf("#<procedure:%s>", p.Name)
	}

	return "#<procedure>"
}

func (p *Proc) Eval(env *Env) (result Expression, nextEnv *Env, err string) {
	return p, env, ""
}

func (_ *Proc) IsLiteral() bool {
	return true
}

type goProcPtr func(args ...Expression) (Expression, string)

type GoProc struct {
	Name    string
	funcPtr goProcPtr
}

func (g *GoProc) Apply(args *SexpPair, env *Env) (result Expression, newEnv *Env, err string) {
	newEnv = env
	argSlice, err := evalArgs(args, env)
	if err != "" {
		return nil, nil, err
	}
	result, err = g.funcPtr(argSlice...)
	return
}

func (g *GoProc) GiveName(name string) {
	if g.Name == "" {
		g.Name = name
	}
}

func (g *GoProc) String() string {
	if g.Name != "" {
		return fmt.Sprintf("#<procedure:%s>", g.Name)
	}

	return "#<procedure>"
}

func (g *GoProc) Eval(env *Env) (result Expression, nextEnv *Env, err string) {
	return g, env, ""
}

func (_ *GoProc) IsLiteral() bool {
	return true
}

func evalArgs(args *SexpPair, env *Env) (argSlice []Expression, err string) {
	argSlice = ToSlice(args)
	var evalErr string
	for i, _ := range argSlice {
		argSlice[i], evalErr = Eval(argSlice[i], env)

		if evalErr != "" {
			return nil, evalErr
		}
	}
	return
}
