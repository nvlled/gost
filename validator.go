package main

import "fmt"

type validateError string

type validator func(vars Vars) (bool, string)

func handleValidation() {
	err := recover()
	if err == nil {
		return
	}
	if err, ok := err.(validateError); ok {
		fmt.Printf("invalid options: %s\n", err)

	} else {
		panic(err)
	}
}

func newValidator(p predicate, failMsg string) validator {
	return func(vars Vars) (bool, string) {
		if !p.apply(vars) {
			return false, failMsg
		}
		return true, ""
	}
}

func validateOpts(opts *gostOpts, vs ...validator) {
	vars := func(name string) string {
		switch name {
		case "srcDir":
			return *opts.srcDir
		case "destDir":
			return *opts.destDir
		}
		return ""
	}
	for _, v := range vs {
		if ok, msg := v(vars); !ok {
			panic(validateError(msg))
		}
	}
}
