package jppool

import (
	jplog "github.com/NBjjp/JpWebFrame/log"
	"time"
)

type Worker struct {
	pool *Pool
	//任务队列
	task chan func()
	//执行任务最后的时间   空闲时间
	lastTime time.Time
}

func (w *Worker) run() {
	w.pool.increRunning()
	go w.running()

}

func (w *Worker) running() {
	defer func() {
		w.pool.workerCache.Put(w)
		w.pool.decreRunning()
		//捕获任务发生的panic
		if err := recover(); err != nil {
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				jplog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			w.pool.workerCache.Put(w)
			return
		}
		f()
		//任务运行完成 worker空闲 还回去
		w.pool.Putworker(w)
	}
}
