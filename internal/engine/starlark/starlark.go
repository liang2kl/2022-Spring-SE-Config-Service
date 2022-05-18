package starlark

import (
	"context"
	"fmt"
	"service/internal/engine"
	"service/internal/utils"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type codeCache struct {
	id         string
	program    *starlark.Program
	cachedTime time.Time
}

const runnerCode = `def run(p):
    p = decode(p)
`
const maxCaches = 100

var (
	codeCaches = make(map[string]codeCache)
	cacheKeys  []string
	cacheLock  sync.RWMutex
)

func ClearCache(id string) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	index := utils.Find(cacheKeys, id)
	if index < 0 {
		return
	}

	delete(codeCaches, id)
	cacheKeys = utils.Remove(cacheKeys, index)
}

func execFile(thread *starlark.Thread, id string, code string, predeclared starlark.StringDict) (starlark.StringDict, error) {
	cacheLock.RLock() // acquire read lock
	cache, cacheExist := codeCaches[id]
	cacheLock.RUnlock() // release read lock

	program := cache.program

	// check if cache has expired
	ttl := viper.GetDuration("code-cache-expiration")
	if cacheExist && time.Since(cache.cachedTime) > ttl {
		ClearCache(cache.id)
		cacheExist = false
	}

	if !cacheExist {
		file, err := syntax.Parse("runner "+id, code, 0)
		if err != nil {
			return nil, err
		}

		program, err = starlark.FileProgram(file, predeclared.Has)

		if err != nil {
			return nil, err
		}

		if id != "" { // empty string indicating no-cache
			cacheLock.Lock() // acquire mutex

			codeCaches[id] = codeCache{
				id:         id,
				program:    program,
				cachedTime: time.Now(),
			}
			cacheKeys = append(cacheKeys, id)

			if len(cacheKeys) > maxCaches {
				// remove oldest cache when exceeded
				delete(codeCaches, cacheKeys[0])
				cacheKeys = cacheKeys[1:]
			}

			cacheLock.Unlock() // release mutex
		}
	}

	globals, err := program.Init(thread, json.Module.Members)
	globals.Freeze()

	return globals, err
}

func Run(id string, code string, params string) engine.RunResult {
	// pre-process code
	thread := &starlark.Thread{}
	code = strings.Replace(code, "\n", "\n    ", -1)
	globals, err := execFile(thread, id, runnerCode+"\n    "+code, json.Module.Members)

	if err != nil {
		return engine.RunResult{Err: err}
	}

	// set timeout
	timeout := viper.GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Millisecond)
	defer cancel()

	// result channel
	ch := make(chan engine.RunResult, 1)

	go func() {
		res := engine.RunResult{}

		runnerFunc := globals["run"]

		if val, err := starlark.Call(thread, runnerFunc,
			starlark.Tuple{starlark.String(params)}, nil); err != nil {
			res.Err = err
		} else {
			res.Val = val.String()
		}

		ch <- res
	}()

	var res engine.RunResult

	select {
	case <-ctx.Done():
		thread.Cancel("")
		res.Err = fmt.Errorf("execution timeout: %dms", timeout)
		return res
	case res := <-ch:
		return res
	}
}
