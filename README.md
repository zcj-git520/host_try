# host_try

## host_try
* host_try：主要是目的是在多个host的下，进行连接的重试
* host_try 只是提供重试连接的接口，具体连接的方法需要自己写

## host_try的工作原理
* 存在多个host的请求提供相同的服务如(ntp)，存在连接超时的情况，需要不断的尝试连接，
* 只要一个host连接请求成功就返回

## 提供三种退避策略
### 第一种策略
* 延时时间* 因子**重连次数(因子默认为2)
```func (t *tryConfig)backOffDelay(n uint) time.Duration```
### 第二种策略
* 随机延迟时间
``` func (t *tryConfig)fixedDelay() time.Duration```
### 第三种策略
* 以相同的时间进行延时
``` func (t *tryConfig)randomDelay() time.Duration```

## 重连方式
### 方式1
* 轮询host进行重连, 重连次数达到后,在将换下一host, 直到重新连接成功或所有的host都重连完成结束
```func (t *tryConfig) directConnection (retryableFunc RetryableFunc)```
### 方式2
* 每一个host进行重连, 在将换下一host, 所有的host失败了,在进行下一次重连。直到重新连接成功或重连次数达到后结束
``` func (t *tryConfig) staggeredConnection (retryableFunc RetryableFunc)```
### 方式3
* 重试直到成功
```func (t *tryConfig) untilConnection (retryableFunc RetryableFunc)```

## 使用
```
host := []string{"172.16.21.1","ntp.ntsc1.ac.cn","cn.ntp1.org.cn","cn.pool.ntp1.org","time.pool.aliyun1.com", "172.16.2.1"}
		demo := New(host, AttemptType("directConnection"))
		demo.DoTry(SetNtpTime)
```
* 重连的入口，当轮询时间为0时，使用方式3，默认为：方式2
```
// 尝试的重连的次数默认为：5次
// 重连方式默认为：方式2
// 延时时间默认为：10*time.Minute
// 最大延时时间默认为：100*time.Minute
// 最大随机数时间默认为：100*time.Minute
// 退避策略类型默认为：策略1
func New(hots []string, opts ...Option) *tryConfig
func (t *tryConfig) DoTry(retryableFunc RetryableFunc)
```
