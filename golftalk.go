package main

import (
	"fmt"
	"strings"
	"regexp"
	"container/list"
	"strconv"
	"bufio"
	"os"
	"io"
)

type Env struct {
	Dict map[string]interface{}
	Outer *Env
}

func (e Env) Find(val string) *Env {
	if e.Dict[val] != nil {
		return &e
	} else if e.Outer != nil {
		return e.Outer.Find(val)
	}

	return nil
}

func NewEnv() *Env {
	env := &Env{}
	env.Dict = make(map[string]interface{})
	return env
}

func MakeEnv(keys []string, vals []interface{}, outer *Env) *Env {
	env := &Env{}
	env.Dict = make(map[string]interface{})

	for i, key := range keys {
		env.Dict[key] = vals[i]
	}

	env.Outer = outer

	return env
}

func get(lst *list.List, n int) interface{} {
	obj := lst.Front()

	for i := 0; i < n; i++ {
		obj = obj.Next()
	}

	return obj.Value
}

func toSlice(lst *list.List) []interface{} {
	slice := make([]interface{}, lst.Len())
	i := 0
	for e := lst.Front(); e != nil; e = e.Next() {
		slice[i] = e.Value
		i++
	}
	return slice
}

func tokenize(s string) string {
	return strings.Trim(strings.Replace(strings.Replace(s, "(", " ( ", -1), ")", " ) ", -1), " ")
}

func splitByRegex(str, regex string) *list.List {
	re := regexp.MustCompile(regex)
	matches := re.FindAllStringIndex(str, -1)

	result := list.New()

	start := 0
	for _, match := range matches {
		result.PushBack(str[start:match[0]])
		start = match[1]

	}

	result.PushBack(str[start:len(str)])

	return result
}

// And here's where we abandon type safety...
func atomize(str string) interface{} {
	// First, try to atomize it as an integer
	if i, err := strconv.ParseInt(str, 10, 32); err == nil {
		return int(i)
	}

	// That didn't work? Maybe it's a float
	// if f, err := strconv.ParseFloat(str, 32); err == nil {
	// 	return f
	// }

	// Fuck it; it's a string
	return str
}

func parseSexp(tokens *list.List) interface{} {
	token, _ := tokens.Remove(tokens.Front()).(string)

	if token == "(" {
		sexp := list.New()
		for true {
			firstTok, _ := tokens.Front().Value.(string)
			if firstTok == ")" {
				break
			}
			sexp.PushBack(parseSexp(tokens))
		}
		tokens.Remove(tokens.Front())
		return sexp
	} else if token == ")" {
		fmt.Println("Unexpected )")
		return nil
	} else {
		return atomize(token)
	}

	return nil
}

func sexpToString(sexp interface{}) string {
	if i, ok := sexp.(int); ok {
		return fmt.Sprintf("%d", i)
	}

	// if f, ok := sexp.(float64); ok {
	// 	return fmt.Sprintf("%f", f)
	// }

	if s, ok := sexp.(string); ok {
		return s
	}

	if l, ok := sexp.(*list.List); ok {
		ret := "("
		for e := l.Front(); e != nil; e = e.Next() {
			ret = ret + sexpToString(e.Value)
			if e.Next() != nil {
				ret = ret + " "
			}
		}
		return ret + ")"
	}

	return ""
}

func eval(val interface{}, env *Env) (interface{}, string) {
	// Make sure the value is an S-expression
	sexp := val
	valStr, wasStr := val.(string)
	if wasStr {
		sexp = parseSexp(splitByRegex(tokenize(valStr), "\\s+"))
	}
	
	// Is the sexp just a symbol?
	// If so, let's look it up and evaluate it!
	if symbol, ok := sexp.(string); ok {
		lookupEnv := env.Find(symbol)
		if lookupEnv != nil {
			return eval(lookupEnv.Dict[symbol], env)
		} else {
			return nil, fmt.Sprintf("'%s' not found in scope chain.", symbol)
		}
	}

	// Is the sexp just a list?
	// If so, let's apply the first symbol as a function to the rest of it!
	if lst, ok := sexp.(*list.List); ok {
		// The "car" of the list will be a symbol representing a function
		car, _ := lst.Front().Value.(string)

		switch car {
			case "insofaras":
				test := get(lst, 1)
				conseq := get(lst, 2)
				alt := get(lst, 3)
				
				
				evalTest, testErr := eval(test, env)
				
				if testErr != "" {
					return nil, testErr
				}
				
				result, wasInt := evalTest.(int)
				
				if !wasInt {
					return nil, "Test given to conditional evaluated as a non-integer."
				} else if result > 0 {
					return eval(conseq, env)
				} else {
					return eval(alt, env)
				}
			case "you-folks":
				literal := list.New()

				for e := lst.Front().Next(); e != nil; e = e.Next() {
					literal.PushBack(e.Value)
				}

				return literal, ""
			case "yknow":
				sym, wasStr := get(lst, 1).(string)
				symExp := get(lst, 2)
				
				if !wasStr {
					return nil, "Symbol given to define wasn't a string."
				}
				
				// TODO: Is there an elegant way to do this?
				evalErr := ""
				env.Dict[sym], evalErr = eval(symExp, env)
				
				return nil, evalErr
			case "apply":
				evalFunc, _ := eval(get(lst, 1), env)
				proc, wasFunc := evalFunc.(func(args ...interface{}) (interface{}, string))
				evalList, _ := eval(get(lst, 2), env)
				args, wasList := evalList.(*list.List)
				
				if !wasFunc {
					return nil, "Function given to apply doesn't evaluate as a function."
				}
				
				if !wasList {
					return nil, "List given to apply doesn't evaluate as a list."
				}
				
				argArr := toSlice(args)
				return proc(argArr...)
			case "bring-me-back-something-good":
				vars, wasList := get(lst, 1).(*list.List)
				exp := get(lst, 2)
				
				if !wasList {
					return nil, "Symbol list to bind within lambda wasn't a list."
				}

				return func(args ...interface{}) (interface{}, string) {
					lambVars := make([]string, vars.Len())
					for i := range lambVars {
						// Outer scope handles possible non-string bindables
						lambVar, _ := get(vars, i).(string)
						lambVars[i] = lambVar
					}

					newEnv := MakeEnv(lambVars, args, env)

					return eval(exp, newEnv)
				}, ""
			case "exit":
				os.Exit(0)
			default:
				evalFunc, funcErr := eval(get(lst, 0), env)
				
				if funcErr != "" {
					return nil, funcErr
				}
				
				args := make([]interface{}, lst.Len() - 1)
				for i := range args {
					// TODO: Do we really need to evaluate here?
					// Lazy evaluation seems to be the way to go, but then wouldn't we have to evaluate arguments in a more limited scope?
					args[i], _ = eval(get(lst, i + 1), env)
				}
				
				proc, wasFunc := evalFunc.(func(args ...interface{}) (interface{}, string))
				if wasFunc {
					return proc(args...)
				} else {
					return nil, "Function to execute was not a valid function."
				}

		}
	}

	// No other choices left; the sexp must be a literal.
	// Let's just return it!
	return sexp, ""
}

func initGlobalEnv(globalEnv *Env) {
	globalEnv.Dict["+"] = func(args ...interface{}) (interface{}, string) {
		accumulator := int(0)
		for _, val := range args {
			i, ok := val.(int)
			if !ok {
				return nil, "Invalid types to add. Must all be int."
			}
			accumulator += i
		}
		return accumulator, ""
	}
	
	globalEnv.Dict["-"] = func(args ...interface{}) (interface{}, string) {
		switch len(args) {
		case 0:
			return nil, "Need at least 1 int to subtract."
		case 1:
			val, ok := args[0].(int)
			if !ok {
				return nil, "Invalid types to subtract. Must all be int."
			}
			return 0 - val, ""
		}

		accumulator := int(0)
		for idx, val := range args {
			i, ok := val.(int)
			if !ok {
				return nil, "Invalid types to subtract. Must all be int."
			}
			if idx == 0 {
				accumulator += i
			} else {
				accumulator -= i
			}
		}
		return accumulator, ""
	}
	
	globalEnv.Dict["*"] = func(args ...interface{}) (interface{}, string) {
		a, aok := args[0].(int)
		b, bok := args[1].(int)
		
		if !aok || !bok {
			return nil, "Invalid types to multiply. Must be int and int."
		}

		return a * b, ""
	}
	
	globalEnv.Dict["/"] = func(args ...interface{}) (interface{}, string) {
		a, aok := args[0].(int)
		b, bok := args[1].(int)
		
		if !aok || !bok {
			return nil, "Invalid types to divide. Must be int and int."
		}
		
		if b == 0 {
			return nil, "Division by zero is currently unsupported."
		}

		return a / b, ""
	}
	
	globalEnv.Dict["or"] = func(args ...interface{}) (interface{}, string) {
		a, aok := args[0].(int)
		b, bok := args[1].(int)
		
		if !aok || !bok {
			return nil, fmt.Sprintf("Invalid types to compare. Must be int and int. Got %d and %d", a, b)
		}
		
		if a > 0 || b > 0 {
			return 1, ""
		}
		
		return 0, ""
	}
	
	globalEnv.Dict["and"] = func(args ...interface{}) (interface{}, string) {
		a, aok := args[0].(int)
		b, bok := args[1].(int)
		
		if !aok || !bok {
			return nil, "Invalid types to compare. Must be int and int."
		}
		
		if a > 0 && b > 0 {
			return 1, ""
		}
		
		return 0, ""
	}
	
	globalEnv.Dict["not"] = func(args ...interface{}) (interface{}, string) {
		a, aok := args[0].(int)
		
		if !aok {
			return nil, "Invalid type to invert. Must be int."
		}
		
		if a > 0 {
			return 0, ""
		}
		
		return 1, ""
	}
	
	globalEnv.Dict["eq?"] = func(args ...interface{}) (interface{}, string) {
		if args[0] == args[1] {
			return 1, ""
		}
		
		return 0, ""
	}
	
	globalEnv.Dict["<"] = func(args ...interface{}) (interface{}, string) {
		a, aok := args[0].(int)
		b, bok := args[1].(int)
		
		if !aok || !bok {
			return nil, "Invalid types to compare. Must be int and int."
		}

		if a < b {
			return 1, ""
		}
		
		return 0, ""
	}
	
	// Neat!
	globalEnv.Dict[">"] = "(bring-me-back-something-good (a b) (< b a))"
	globalEnv.Dict["<="] = "(bring-me-back-something-good (a b) (or (< a b) (eq? a b)))"
	globalEnv.Dict[">="] = "(bring-me-back-something-good (a b) (or (> a b) (eq? a b)))"
	// Dat spaceship operator
	globalEnv.Dict["<==>"] = "(bring-me-back-something-good (a b) (insofaras (< a b) -1 (insofaras (> a b) 1 0"
	
	
	globalEnv.Dict["fib"] = "(bring-me-back-something-good (n) (insofaras (< n 2) n (+ (fib (- n 1)) (fib (- n 2)))))"
	globalEnv.Dict["fact"] = "(bring-me-back-something-good (n) (insofaras (eq? n 0) 1 (* n (fact (- n 1)))))"
}

func main() {
	globalEnv := NewEnv()

	initGlobalEnv(globalEnv)

	in := bufio.NewReader(os.Stdin)

	for true {
		fmt.Print("golftalk~$ ")
		line, err := in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				break
			} else {
				panic(err)
			}
		}
		if line != "" && line != "\n" {
			result, evalErr := eval(line, globalEnv)
			
			if evalErr != "" {
				fmt.Printf("No.\n\t%s\n", evalErr)
				continue
			}
			
			if result != nil {
				fmt.Println(result)
			}
		}
	}
}
