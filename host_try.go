package host_try
//host_try：主要是目的是在多个host的下，进行连接的重试
//host_try 只是提供重试连接的接口，具体连接的方法需要自己写
// host_try的工作原理：
// 存在多个host的请求提供相同的服务如(ntp)，存在连接超时的情况，需要不断的尝试连接，
// 只要一个host连接请求成功就返回
//提供三种退避策略：
// 第一种策略是：延时时间* 因子**重连次数(因子默认为2)
// 第二种策略是：随机延迟时间
// 第三种策略是：以相同的时间进行延时
// 三种策略可以同时使用
// 提供三种重试的方案：
// 方案1：轮询host进行重连, 重连次数达到后,在将换下一host, 直到重新连接成功或所有的host都重连完成结束
// 方案2：每一个host进行重连, 在将换下一host, 所有的host失败了,在进行下一次重连。直到重新连接成功或重连次数达到后结束
// 方案3：重试直到成功
//三种方案只能选择其中的一个
// 可以知道连接是否成功，成功的host
// 失败的host和失败的原因
import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)


const (
	Factor = 2  // 超时因子
	// 退避策略
	BackOffDelay  uint = iota
	FixedDelay
	RandomDelay
)

type tryConfig struct {
	attemptNum    uint           	 // 尝试的重连的次数
	nowAttempt    uint               // 现在第几次重连
	hosts         []string        	 // 连接的host组
	successHost   string             // 连接成功的host
	errorHost     map[string]string  // 连接失败的host和错误原因
	attemptType   string          	 // 重连方式
	attemptStatus bool               // 最终重连的状态
	delay         time.Duration    	 // 延时时间
	maxDelay      time.Duration    	 // 最大延时时间
	maxJitter     time.Duration      // 最大随机数时间
	delayType     []uint   	         // 退避策略类型
	contest       context.Context
}

type Option func( *tryConfig)

// 连接的接口
type RetryableFunc func(string) error

func AttemptNums(attempts uint) Option {
	return func(t *tryConfig) {
		t.attemptNum = attempts
	}
}

// 参数的设定
func AttemptType(s string) Option {
	return func(t *tryConfig) {
		t.attemptType = s
	}
}

func Delay(d time.Duration) Option {
	return func(t *tryConfig) {
		t.delay = d
	}
}

func MaxDelay(d time.Duration) Option {
	return func(t *tryConfig) {
		t.maxDelay = d
	}
}

func MaxJitter(d time.Duration) Option {
	return func(t *tryConfig) {
		t.maxJitter = d
	}
}

func DelayType(types []uint) Option {
	return func(t *tryConfig) {
		t.delayType = types
	}
}

// 可以知道连接是否成功，成功的host
func (t *tryConfig)GetSuccessHost()string{
	return t.successHost
}

// 失败的host和失败的原因
func (t *tryConfig) GetErrorHost()map[string]string {
	return t.errorHost
}

// 退避策略
// 策略1：延时时间* 因子**重连次数(因子默认为2)
func (t *tryConfig)backOffDelay(n uint) time.Duration {
	dur := float64(t.delay) * math.Pow(Factor, float64(n))
	if dur > float64(t.maxDelay) {
		return t.maxDelay
	}
	return time.Duration(dur)
}

// 策略2：随机延迟时间
func (t *tryConfig)fixedDelay() time.Duration {
	return t.delay
}

// 策略3：以相同的时间进行延时
func (t *tryConfig)randomDelay() time.Duration {
	return time.Duration(rand.Int63n(int64(t.maxJitter)))
}

// 通过设置退避策略，会的延时时间，默认为策略1
func (t *tryConfig)combineDelay() time.Duration {
	const maxInt64 = uint64(math.MaxInt64)
	var total uint64
	total = 0
	for _, delay := range t.delayType {
		switch delay {
		case BackOffDelay:
			total += uint64(t.backOffDelay(t.nowAttempt))
		case FixedDelay:
			total += uint64(t.fixedDelay())
		case RandomDelay:
			total += uint64(t.randomDelay())

		}
	}
	if total > maxInt64 {
		total = maxInt64
	}
	if total == 0{
		total = uint64(t.backOffDelay(t.nowAttempt))
	}
	delayTime := time.Duration(total)
	if t.maxDelay > 0 && delayTime > t.maxDelay {
		delayTime = t.maxDelay
	}
	return delayTime
}

// 重连方式
// 方式1：轮询host进行重连, 重连次数达到后,在将换下一host, 直到重新连接成功或所有的host都重连完成结束
func (t *tryConfig) directConnection (retryableFunc RetryableFunc)  {
	t.errorHost = make(map[string]string)
	for _, host := range t.hosts{
		t.nowAttempt = 1
		errStr := ""   // 失败的错误原因
		for t.nowAttempt <= t.attemptNum{
			err := retryableFunc(host)
			if err == nil{
				// 连接成功
				t.successHost = host
				t.attemptStatus = true
				return
			}else{
				// 连接失败, 进入等待
				// 最后一次尝试就不等待
				errStr = err.Error()
				if t.nowAttempt == t.attemptNum {
					break
				}
				delayTime := t.combineDelay()
				fmt.Println("连接失败, 进入等待", t.nowAttempt, delayTime )
				select {
				case <- time.After(delayTime):
				case <- t.contest.Done():
				}
			}
			t.nowAttempt ++
		}
		// 尝试失败后
		t.errorHost[host] = errStr
	}
	t.attemptStatus = false
}

// 方式2：每一个host进行重连, 在将换下一host, 所有的host失败了,在进行下一次重连。直到重新连接成功或重连次数达到后结束
func (t *tryConfig) staggeredConnection (retryableFunc RetryableFunc)  {
	t.errorHost = make(map[string]string)
	for num := uint(1); num <= t.attemptNum; num ++{
		t.nowAttempt = num
		for i, host := range t.hosts{
			err := retryableFunc(host)
			if err == nil{
				// 连接成功
				t.successHost = host
				t.attemptStatus = true
				return
			}else{
				// 连接失败, 进入等待
				fmt.Println("连接失败, 进入等待")
				t.errorHost[host] = err.Error()
				// 最后host尝试就不等待
				if i >= len(t.hosts)-1{
					break
				}
				delayTime := t.combineDelay()
				select {
				case <- time.After(delayTime):
				case <- t.contest.Done():
				}
			}
		}
	}
	t.attemptStatus = false
}

// 方式3：重试直到成功
func (t *tryConfig) untilConnection (retryableFunc RetryableFunc)  {
	t.attemptNum = 1
	for {
		for _, host := range t.hosts{
			err := retryableFunc(host)
			if err == nil{
				t.successHost = host
				t.attemptStatus = true
				return
			}else{
				delayTime := t.combineDelay()
				select {
				case <- time.After(delayTime):
				case <- t.contest.Done():
				}
			}
			t.attemptNum ++
		}
	}

}

// 重连的入口，当轮询时间为0时，使用方式3，默认为：方式2
func (t *tryConfig) DoTry(retryableFunc RetryableFunc) {
	if t.attemptNum == 0{
		t.untilConnection(retryableFunc)
		return
	}
	switch t.attemptType {
	case "directConnection":
		t.directConnection(retryableFunc)
	case "staggeredConnection":
		t.staggeredConnection(retryableFunc)
	case "untilConnection":
		t.untilConnection(retryableFunc)
	default:
		t.staggeredConnection(retryableFunc)
	}

}

// 尝试的重连的次数默认为：5次
// 重连方式默认为：方式2
// 延时时间默认为：10*time.Minute
// 最大延时时间默认为：100*time.Minute
// 最大随机数时间默认为：100*time.Minute
// 退避策略类型默认为：策略1
func New(hots []string, opts ...Option) *tryConfig {
	delayType := []uint{0}
	try := &tryConfig{
		attemptNum: 5,
		hosts:hots,
		attemptType:"staggeredConnection",
		delay: 10*time.Minute,
		maxDelay: 100*time.Minute,
		maxJitter: 100*time.Minute,
		delayType: delayType,
		contest: context.Background(),
	}
	for _, opt:= range opts{
		opt(try)
	}
	return try
}


