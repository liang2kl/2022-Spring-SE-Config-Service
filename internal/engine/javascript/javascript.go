package javascript

import (
	"context"
	"fmt"
	"service/internal/engine"
	"sync"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/spf13/viper"
)

const code = `
function run(code, p) {
    "use strict"
    p = JSON.parse(p)

    eval("function runner (p) {\n" + code + "\n}")
    var ret = eval("runner(p)")

    if (typeof ret === "object" ||
        typeof ret === "array") {
        return JSON.stringify(ret)
    }
    return ret
}
`

type jsEngine struct {
	mu      sync.Mutex
	engine  *otto.Otto
	running bool
}

// max concurrent runners
const runnerNum = 20

// runner pool
var engines [runnerNum]*jsEngine

func getEngine() *jsEngine {
	for _, engine := range engines {
		if !engine.running {
			engine.running = true
			return engine
		}
	}
	return engines[0]
}

func Init() error {
	// create template
	templateEngine := otto.New()
	if _, err := templateEngine.Run(code); err != nil {
		return err
	}

	for i := 0; i < runnerNum; i++ {
		engines[i] = &jsEngine{engine: templateEngine.Copy()}
	}

	return nil
}

func Run(id string, code string, params string) engine.RunResult {
	e := getEngine()
	// acquire lock
	e.mu.Lock()
	// release lock on exit
	defer e.mu.Unlock()

	timeout := viper.GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Millisecond)
	defer cancel()

	defer func() {
		// reset the running state
		e.running = false
	}()

	// interrupt channel
	e.engine.Interrupt = make(chan func(), 1)
	// result channel
	ch := make(chan engine.RunResult, 1)

	go func() {
		res := engine.RunResult{}

		val, err := e.engine.Call("run", nil, code, params)
		res.Err = err

		if err == nil {
			res.Val, res.Err = val.ToString()
		}
		ch <- res
	}()

	var res engine.RunResult

	select {
	case <-ctx.Done():
		e.engine.Interrupt <- func() {
			// nothing to do
		}
		res.Err = fmt.Errorf("execution timeout: %dms", timeout)
		return res
	case res := <-ch:
		return res
	}
}
