package jppool

import (
	"errors"
	"fmt"
	"github.com/NBjjp/JpWebFrame/config"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultExpire = 3

var ErrorInvalidCap = errors.New("pool cap can not <= 0")
var ErrorInvalidExp = errors.New("pool expire can not <= 0")
var ErrorHasClosed = errors.New("pool has been released")

type sig struct {
}
type Pool struct {
	//空闲worker
	workers []*Worker
	//容量 pool max cap
	cap int32
	//正在运行的worker数量
	running int32
	//过期时间    空闲的worker超过这个时间就进行回收
	expire time.Duration
	//release 释放资源  pool不能使用了
	release chan sig
	//保护pool里面相关资源的安全
	lock sync.Mutex
	//once 释放只能调用一次，不能调用多次
	once sync.Once
	//workerCache 缓存
	workerCache sync.Pool
	//cond 条件
	cond         *sync.Cond
	PanicHandler func()
}

func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DefaultExpire)
}

//通过配置文件配置
func NewPoolConf() (*Pool, error) {
	capacity, ok := config.Conf.Pool["cap"]
	if !ok {
		panic("conf pool.cap not config")
	}
	return NewTimePool(int(capacity.(int64)), DefaultExpire)
}

func NewTimePool(cap int, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInvalidCap
	}
	if expire <= 0 {
		return nil, ErrorInvalidExp
	}
	p := &Pool{
		cap:     int32(cap),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	//sync.pool
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	//sync.cond
	p.cond = sync.NewCond(&p.lock)
	//单独开启一个协程，清楚长期未使用的worker
	go p.expireWorker()
	return p, nil
}

//定时清除无用的worker
func (p *Pool) expireWorker() {
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		if p.IsClosed() {
			break
		}
		p.lock.Lock()
		//循环空闲的worker    如果当前时间和worker最后运行时间 插值大于expire  进行清理
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n >= 0 {
			var clearN = -1
			for i, w := range idleWorkers {
				if time.Now().Sub(w.lastTime) <= p.expire {
					break
				}
				clearN = i
				w.task <- nil
				idleWorkers[i] = nil
			}
			// 3 2
			if clearN != -1 {
				if clearN >= len(idleWorkers)-1 {
					p.workers = idleWorkers[:0]
				} else {
					// len=3 0,1 del 2
					p.workers = idleWorkers[clearN+1:]
				}
				fmt.Printf("清除完成,running:%d, workers:%v \n", p.running, p.workers)
			}
		}
		p.lock.Unlock()
	}
}

//提交任务
func (p *Pool) Submit(task func()) error {
	//判断pool是否被释放
	if len(p.release) > 0 {
		return ErrorHasClosed
	}
	//获取池里面的worker，然后执行任务就可以
	w := p.GetWorker()
	//执行队列
	w.task <- task

	return nil
}

//获取pool里面的worker
func (p *Pool) GetWorker() *Worker {
	//获取pool里面的worker
	//如果有空闲的worker 直接获取
	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 { //有空闲的worker
		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
	//如果没有空闲的worker，新建一个worker
	if p.running < p.cap {
		p.lock.Unlock()
		//pool的容量没有满，新建一个worker
		c := p.workerCache.Get()
		var w *Worker
		if c == nil {
			w = &Worker{
				pool: p,
				task: make(chan func(), 1),
			}
		} else {
			w = c.(*Worker)
		}
		w.run()
		return w
	}
	p.lock.Unlock()
	//如果正在运行的worker + 空闲worker 如果大于pool容量，阻塞等待worker释放。
	//for {
	return p.WaitIdleWorker()
	//}
}
func (p *Pool) WaitIdleWorker() *Worker {
	p.lock.Lock()
	//等通知
	p.cond.Wait()
	fmt.Println("得到通知，有空闲worker了")
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 {
		p.lock.Unlock()
		if p.running < p.cap {
			//还不够pool的容量，直接新建一个
			c := p.workerCache.Get()
			var w *Worker
			if c == nil {
				w = &Worker{
					pool: p,
					task: make(chan func(), 1),
				}
			} else {
				w = c.(*Worker)
			}
			w.run()
			return w
		}
		return p.WaitIdleWorker()
	}
	w := idleWorkers[n]
	idleWorkers[n] = nil
	p.workers = idleWorkers[:n]
	p.lock.Unlock()
	return w
}
func (p *Pool) increRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) Putworker(w *Worker) {
	w.lastTime = time.Now() //任务最后运行的时间
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.cond.Signal()
	p.lock.Unlock()

}

func (p *Pool) decreRunning() {
	atomic.AddInt32(&p.running, -1)
}

//释放pool资源
func (p *Pool) Release() {
	p.once.Do(func() {
		//只执行一次
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		//发一个信号
		p.release <- sig{}
	})
}

//判断pool是否被释放
func (p *Pool) IsClosed() bool {
	return len(p.release) > 0
}

//重启
func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	go p.expireWorker()
	return true
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}
func (p *Pool) Free() int {
	return int(p.cap - p.running)
}
