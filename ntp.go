package host_try

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const (
	//Unix 时间是一个开始于 1970 年的纪元（或者说从 1970 年开始的秒数）.然而 NTP 使用的是另外一个纪元,
	//从 1900 年开始的秒数.因此,从 NTP 服务端获取到的值要正确地转成 Unix 时间必须减掉这 70 年间的秒数 （1970-1900）
	// 或者说 2208988800 秒
	ntpEpochOffset = 2208988800
	//ntpHost = "ntp.ntsc.ac.cn:123"  // 国家授时中心 NTP 服务器
	ntpPort = ":123"
	RTC = 'p'
)

type ntpTime struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}

// 远程连接ntp服务器
// 获取ntp服务器事件
// 参数为ntp服务器的地址
func SetNtpTime(host string)  error {
	ntpHost := host + ntpPort
	conn, err := net.Dial("udp", ntpHost)  // 建立连接
	if err != nil {
		return  fmt.Errorf("%s Ntp server failed to connect: %s", ntpHost,err )
	}
	defer conn.Close()
	//我们通过 UDP 协议，使用　net.Dial 函数去启动一个 socket，与 NTP 服务器联系，并设定 15 秒的超时时间。
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return  fmt.Errorf("failed to set deadline:%s", err)
	}
	req := &ntpTime{Settings: 0x1B}  //（或者二进制 00011011），代表客户端模式为 3，NTP版本为 3，润年为 0
	//使用 binary 库去自动地将 ntpTime 结构体封装成字节流，并以大端格式发送出去。
	err = binary.Write(conn, binary.BigEndian, req)
	if err != nil {
		return  fmt.Errorf("failed to send request:%s", err)
	}
	//使用 binary 包再次将从服务端读取的字节流自动地解封装成对应的 ntpTime 结构体
	rsp := &ntpTime{}
	err = binary.Read(conn, binary.BigEndian, rsp)
	if err != nil {
		return  fmt.Errorf("failed to read server response:%s", err)
	}
	secs := float64(rsp.TxTimeSec) - ntpEpochOffset
	sec := (int64(rsp.TxTimeFrac) * 1e9) >> 32
	ntpTime := time.Unix(int64(secs), sec)
	fmt.Printf("%d-%d-%d %d:%d:%d",ntpTime.Year(), ntpTime.Month(), ntpTime.Day(), ntpTime.Hour(),
		ntpTime.Minute(),ntpTime.Second())
	return  nil
}
